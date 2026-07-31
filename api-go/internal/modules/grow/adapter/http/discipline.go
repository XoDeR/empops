package http

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

func (h *Handler) listDisciplineFiles(r *http.Request, eventID string) []map[string]interface{} {
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, file_name, mime_type, size FROM media
		WHERE model_type=$1 AND model_id=$2 AND collection_name=$3 ORDER BY id`,
		modelDisciplineEvent, eventID, collectionDiscipline)
	out := []map[string]interface{}{}
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var fileName, mimeType string
		var size int64
		if rows.Scan(&id, &fileName, &mimeType, &size) == nil {
			out = append(out, h.filePayload(id, fileName, mimeType, size))
		}
	}
	return out
}

func (h *Handler) disciplineCasePayload(r *http.Request, caseID string, withEvents bool) (map[string]interface{}, error) {
	var employeeID string
	var openedByID *string
	var openedByName *string
	var active bool
	var createdAt time.Time
	err := h.pool.QueryRow(r.Context(), `
		SELECT employee_id, opened_by_employee_id, opened_by_employee_name, active, created_at
		FROM discipline_cases WHERE id=$1`, caseID,
	).Scan(&employeeID, &openedByID, &openedByName, &active, &createdAt)
	if err != nil {
		return nil, err
	}
	var openedBy interface{}
	if openedByID != nil {
		openedBy = h.employeeSummary(r.Context(), *openedByID)
	}
	payload := map[string]interface{}{
		"id":                       caseID,
		"employee":                 h.employeeSummary(r.Context(), employeeID),
		"opened_by":                openedBy,
		"opened_by_employee_name":  openedByName,
		"active":                   active,
		"created_at":               createdAt.UTC().Format(time.RFC3339),
	}
	if withEvents {
		events := []map[string]interface{}{}
		rows, _ := h.pool.Query(r.Context(), `
			SELECT id, author_id, author_name, happened_at, description, created_at
			FROM discipline_events WHERE discipline_case_id=$1 ORDER BY happened_at, created_at`, caseID)
		if rows != nil {
			for rows.Next() {
				var eid string
				var authorID *string
				var authorName, description string
				var happenedAt, ca time.Time
				if rows.Scan(&eid, &authorID, &authorName, &happenedAt, &description, &ca) == nil {
					var author interface{}
					if authorID != nil {
						author = h.employeeSummary(r.Context(), *authorID)
					}
					events = append(events, map[string]interface{}{
						"id": eid, "author_name": authorName, "author": author,
						"happened_at": happenedAt.Format("2006-01-02"),
						"description": description,
						"files":       h.listDisciplineFiles(r, eid),
						"created_at":  ca.UTC().Format(time.RFC3339),
					})
				}
			}
			rows.Close()
		}
		payload["events"] = events
	}
	return payload, nil
}

func (h *Handler) ListDisciplineCases(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	q := `SELECT id FROM discipline_cases WHERE company_id=$1`
	args := []interface{}{member.CompanyID}
	if !h.isHr(member) {
		q += ` AND employee_id IN (SELECT employee_id FROM direct_reports WHERE company_id=$1 AND manager_id=$2)`
		args = append(args, member.EmployeeID)
	}
	if raw := r.URL.Query().Get("active"); raw != "" {
		active, _ := strconv.ParseBool(raw)
		q += fmt.Sprintf(` AND active=$%d`, len(args)+1)
		args = append(args, active)
	}
	q += ` ORDER BY created_at DESC`
	rows, err := h.pool.Query(r.Context(), q, args...)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			if p, err := h.disciplineCasePayload(r, id, false); err == nil {
				out = append(out, p)
			}
		}
	}
	response.OK(w, "", out)
}

func (h *Handler) ShowDisciplineCase(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	caseID := chi.URLParam(r, "caseId")
	var employeeID, companyID string
	err := h.pool.QueryRow(r.Context(), `SELECT employee_id, company_id FROM discipline_cases WHERE id=$1`, caseID).Scan(&employeeID, &companyID)
	if err == pgx.ErrNoRows || companyID != member.CompanyID {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if !h.isHr(member) && !h.isManagerOf(r.Context(), member.CompanyID, member.EmployeeID, employeeID) {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	p, err := h.disciplineCasePayload(r, caseID, true)
	if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return
	}
	response.OK(w, "", p)
}

func (h *Handler) StoreDisciplineCase(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	var body struct {
		EmployeeID string `json:"employee_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.Fail(w, 422, "invalid body", nil)
		return
	}
	var ok bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND company_id=$2)`, body.EmployeeID, member.CompanyID).Scan(&ok)
	if !ok {
		response.Fail(w, 404, "Employee not found", nil)
		return
	}
	if !h.isHr(member) && !h.isManagerOf(r.Context(), member.CompanyID, member.EmployeeID, body.EmployeeID) {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	sum := h.employeeSummary(r.Context(), member.EmployeeID)
	name := fmt.Sprintf("%v %v", sum["first_name"], sum["last_name"])
	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO discipline_cases (id, company_id, employee_id, opened_by_employee_id, opened_by_employee_name, active)
		VALUES ($1,$2,$3,$4,$5,true)`, id, member.CompanyID, body.EmployeeID, member.EmployeeID, name)
	if err != nil {
		response.Fail(w, 500, "create failed", err.Error())
		return
	}
	p, _ := h.disciplineCasePayload(r, id, true)
	response.OK(w, "", p)
}

func (h *Handler) ToggleDisciplineCase(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	if !h.isHr(member) {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	caseID := chi.URLParam(r, "caseId")
	tag, err := h.pool.Exec(r.Context(), `
		UPDATE discipline_cases SET active = NOT active, updated_at=now()
		WHERE id=$1 AND company_id=$2`, caseID, member.CompanyID)
	if err != nil {
		response.Fail(w, 500, "update failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	p, _ := h.disciplineCasePayload(r, caseID, true)
	response.OK(w, "", p)
}

func (h *Handler) DestroyDisciplineCase(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	if !h.isHr(member) {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	caseID := chi.URLParam(r, "caseId")
	tag, err := h.pool.Exec(r.Context(), `DELETE FROM discipline_cases WHERE id=$1 AND company_id=$2`, caseID, member.CompanyID)
	if err != nil {
		response.Fail(w, 500, "delete failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	response.OK(w, "", nil)
}

func (h *Handler) StoreDisciplineEvent(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	caseID := chi.URLParam(r, "caseId")
	var employeeID, companyID string
	err := h.pool.QueryRow(r.Context(), `SELECT employee_id, company_id FROM discipline_cases WHERE id=$1`, caseID).Scan(&employeeID, &companyID)
	if err == pgx.ErrNoRows || companyID != member.CompanyID {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if !h.isHr(member) && !h.isManagerOf(r.Context(), member.CompanyID, member.EmployeeID, employeeID) {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	var body struct {
		HappenedAt  string `json:"happened_at"`
		Description string `json:"description"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Description == "" {
		response.Fail(w, 422, "invalid body", nil)
		return
	}
	sum := h.employeeSummary(r.Context(), member.EmployeeID)
	name := fmt.Sprintf("%v %v", sum["first_name"], sum["last_name"])
	id := uuidv7.New()
	var createdAt time.Time
	err = h.pool.QueryRow(r.Context(), `
		INSERT INTO discipline_events (id, discipline_case_id, author_id, author_name, happened_at, description)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING created_at`,
		id, caseID, member.EmployeeID, name, body.HappenedAt, body.Description,
	).Scan(&createdAt)
	if err != nil {
		response.Fail(w, 500, "create failed", err.Error())
		return
	}
	response.OK(w, "", map[string]interface{}{
		"id": id, "author_name": name, "happened_at": body.HappenedAt,
		"description": body.Description, "files": []interface{}{},
		"created_at": createdAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) DestroyDisciplineEvent(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	caseID := chi.URLParam(r, "caseId")
	eventID := chi.URLParam(r, "eventId")
	var employeeID, companyID string
	err := h.pool.QueryRow(r.Context(), `SELECT employee_id, company_id FROM discipline_cases WHERE id=$1`, caseID).Scan(&employeeID, &companyID)
	if err == pgx.ErrNoRows || companyID != member.CompanyID {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if !h.isHr(member) && !h.isManagerOf(r.Context(), member.CompanyID, member.EmployeeID, employeeID) {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `
		DELETE FROM discipline_events WHERE id=$1 AND discipline_case_id=$2`, eventID, caseID)
	if err != nil {
		response.Fail(w, 500, "delete failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	response.OK(w, "", nil)
}

func (h *Handler) AttachDisciplineFile(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	caseID := chi.URLParam(r, "caseId")
	eventID := chi.URLParam(r, "eventId")
	var employeeID, companyID string
	err := h.pool.QueryRow(r.Context(), `SELECT employee_id, company_id FROM discipline_cases WHERE id=$1`, caseID).Scan(&employeeID, &companyID)
	if err == pgx.ErrNoRows || companyID != member.CompanyID {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if !h.isHr(member) && !h.isManagerOf(r.Context(), member.CompanyID, member.EmployeeID, employeeID) {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM discipline_events WHERE id=$1 AND discipline_case_id=$2)`, eventID, caseID).Scan(&exists)
	if !exists {
		response.Fail(w, 404, "Event not found", nil)
		return
	}
	var body struct {
		TemporaryUploadID int64 `json:"temporary_upload_id"`
		MediaID           int64 `json:"media_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.Fail(w, 422, "invalid body", nil)
		return
	}
	var fileName string
	err = h.pool.QueryRow(r.Context(), `
		SELECT file_name FROM media
		WHERE id=$1 AND model_type=$2 AND model_id=$3`,
		body.MediaID, modelTemporaryUpload, fmt.Sprintf("%d", body.TemporaryUploadID),
	).Scan(&fileName)
	if err == pgx.ErrNoRows {
		response.Fail(w, 404, "Media not found on temporary upload", nil)
		return
	}
	if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return
	}
	_, err = h.pool.Exec(r.Context(), `
		UPDATE media SET model_type=$2, model_id=$3, collection_name=$4, updated_at=now()
		WHERE id=$1 AND model_type=$5 AND model_id=$6`,
		body.MediaID, modelDisciplineEvent, eventID, collectionDiscipline,
		modelTemporaryUpload, fmt.Sprintf("%d", body.TemporaryUploadID))
	if err != nil {
		response.Fail(w, 500, "attach failed", err.Error())
		return
	}
	var mimeType string
	var size int64
	_ = h.pool.QueryRow(r.Context(), `SELECT mime_type, size FROM media WHERE id=$1`, body.MediaID).Scan(&mimeType, &size)
	response.OK(w, "", h.filePayload(body.MediaID, fileName, mimeType, size))
}
