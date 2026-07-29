package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/XoDeR/empops/api-go/pkg/audit"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

type createWorklogRequest struct {
	Content  string  `json:"content"`
	LoggedOn *string `json:"logged_on"`
}

// CreateWorklog handles POST /companies/{companyId}/worklogs.
func (h *Handler) CreateWorklog(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")

	var req createWorklogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		response.Fail(w, http.StatusBadRequest, "content is required", nil)
		return
	}

	loggedOn := time.Now().UTC().Truncate(24 * time.Hour)
	if req.LoggedOn != nil && strings.TrimSpace(*req.LoggedOn) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*req.LoggedOn))
		if err != nil {
			response.Fail(w, http.StatusBadRequest, "invalid logged_on", nil)
			return
		}
		loggedOn = parsed
	}

	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO worklogs (id, company_id, employee_id, content, logged_on)
		VALUES ($1, $2, $3, $4, $5)`,
		id, companyID, member.EmployeeID, req.Content, loggedOn,
	)
	if err != nil {
		if isUniqueViolation(err) {
			response.Fail(w, http.StatusConflict, "Worklog already logged for this day", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "create worklog failed", err.Error())
		return
	}

	_, _ = h.pool.Exec(r.Context(), `
		UPDATE employees SET consecutive_worklog_missed = 0, updated_at = now() WHERE id = $1`,
		member.EmployeeID,
	)

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "worklog.created", member.EmployeeID, strPtr("worklog"), &id, map[string]interface{}{
		"logged_on": loggedOn.Format("2006-01-02"),
	})

	payload, err := h.worklogPayload(r.Context(), id)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "worklog payload failed", err.Error())
		return
	}
	response.Created(w, "Worklog created", payload)
}

// ListEmployeeWorklogs handles GET /companies/{companyId}/employees/{employeeId}/worklogs.
func (h *Handler) ListEmployeeWorklogs(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	employeeID := chi.URLParam(r, "employeeId")

	if !h.employeeExists(r, companyID, employeeID) {
		response.Fail(w, http.StatusNotFound, "Employee not found", nil)
		return
	}
	if !h.canAccessWorklogs(r, member, companyID, employeeID, "worklogs.view") {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	from := strings.TrimSpace(r.URL.Query().Get("from"))
	to := strings.TrimSpace(r.URL.Query().Get("to"))

	query := `
		SELECT id, company_id, employee_id, content, logged_on, created_at, updated_at
		FROM worklogs
		WHERE company_id = $1 AND employee_id = $2`
	args := []interface{}{companyID, employeeID}
	if from != "" {
		args = append(args, from)
		query += " AND logged_on >= $" + strconv.Itoa(len(args))
	}
	if to != "" {
		args = append(args, to)
		query += " AND logged_on <= $" + strconv.Itoa(len(args))
	}
	query += " ORDER BY logged_on DESC"

	rows, err := h.pool.Query(r.Context(), query, args...)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list worklogs failed", err.Error())
		return
	}
	defer rows.Close()

	list := []map[string]interface{}{}
	for rows.Next() {
		var id, cid, empID, content string
		var loggedOn, createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &cid, &empID, &content, &loggedOn, &createdAt, &updatedAt); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		list = append(list, map[string]interface{}{
			"id":         id,
			"company_id": cid,
			"employee_id": empID,
			"content":    content,
			"logged_on":  loggedOn.Format("2006-01-02"),
			"created_at": createdAt.UTC().Format(time.RFC3339),
			"updated_at": updatedAt.UTC().Format(time.RFC3339),
		})
	}
	response.OK(w, "", list)
}

// DeleteWorklog handles DELETE /companies/{companyId}/employees/{employeeId}/worklogs/{worklogId}.
func (h *Handler) DeleteWorklog(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	employeeID := chi.URLParam(r, "employeeId")
	worklogID := chi.URLParam(r, "worklogId")

	if !h.employeeExists(r, companyID, employeeID) {
		response.Fail(w, http.StatusNotFound, "Employee not found", nil)
		return
	}
	if !h.canAccessWorklogs(r, member, companyID, employeeID, "worklogs.delete") {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	tag, err := h.pool.Exec(r.Context(), `
		DELETE FROM worklogs WHERE id = $1 AND company_id = $2 AND employee_id = $3`,
		worklogID, companyID, employeeID,
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete worklog failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Worklog not found", nil)
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "worklog.deleted", member.EmployeeID, strPtr("worklog"), &worklogID, map[string]interface{}{
		"employee_id": employeeID,
	})

	response.OK(w, "Worklog deleted", nil)
}

// ListTeamWorklogs handles GET /companies/{companyId}/teams/{teamId}/worklogs.
func (h *Handler) ListTeamWorklogs(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	teamID := chi.URLParam(r, "teamId")

	if !h.teamExistsInCompany(r, companyID, teamID) {
		response.Fail(w, http.StatusNotFound, "Team not found", nil)
		return
	}

	inTeam := h.employeeInTeam(r, teamID, member.EmployeeID)
	if !inTeam && !member.HasPermission("worklogs.view") {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT w.id, w.company_id, w.employee_id, w.content, w.logged_on, w.created_at, w.updated_at,
			e.first_name, e.last_name, e.email
		FROM worklogs w
		JOIN employee_team et ON et.employee_id = w.employee_id
		JOIN employees e ON e.id = w.employee_id
		WHERE et.team_id = $1 AND w.company_id = $2 AND w.logged_on = $3
		ORDER BY e.last_name, e.first_name`, teamID, companyID, date)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list team worklogs failed", err.Error())
		return
	}
	defer rows.Close()

	list := []map[string]interface{}{}
	for rows.Next() {
		var id, cid, empID, content, firstName, lastName, email string
		var loggedOn, createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &cid, &empID, &content, &loggedOn, &createdAt, &updatedAt, &firstName, &lastName, &email); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		list = append(list, map[string]interface{}{
			"id":          id,
			"company_id":  cid,
			"employee_id": empID,
			"content":     content,
			"logged_on":   loggedOn.Format("2006-01-02"),
			"created_at":  createdAt.UTC().Format(time.RFC3339),
			"updated_at":  updatedAt.UTC().Format(time.RFC3339),
			"employee": map[string]interface{}{
				"id":         empID,
				"first_name": firstName,
				"last_name":  lastName,
				"email":      email,
			},
		})
	}

	response.OK(w, "", map[string]interface{}{
		"date":     date,
		"worklogs": list,
	})
}

func (h *Handler) canAccessWorklogs(r *http.Request, member companyauth.Member, companyID, employeeID, perm string) bool {
	if member.EmployeeID == employeeID {
		return true
	}
	if member.HasPermission(perm) {
		return true
	}
	return h.isManagerOf(r, companyID, member.EmployeeID, employeeID)
}

func (h *Handler) isManagerOf(r *http.Request, companyID, managerID, employeeID string) bool {
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM direct_reports WHERE company_id = $1 AND manager_id = $2 AND employee_id = $3)`,
		companyID, managerID, employeeID,
	).Scan(&exists)
	return exists
}

func (h *Handler) teamExistsInCompany(r *http.Request, companyID, teamID string) bool {
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM teams WHERE id = $1 AND company_id = $2)`, teamID, companyID,
	).Scan(&exists)
	return exists
}

func (h *Handler) employeeInTeam(r *http.Request, teamID, employeeID string) bool {
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM employee_team WHERE team_id = $1 AND employee_id = $2)`, teamID, employeeID,
	).Scan(&exists)
	return exists
}

func (h *Handler) worklogPayload(ctx context.Context, worklogID string) (map[string]interface{}, error) {
	var id, cid, employeeID, content string
	var loggedOn, createdAt, updatedAt time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT id, company_id, employee_id, content, logged_on, created_at, updated_at
		FROM worklogs WHERE id = $1`, worklogID,
	).Scan(&id, &cid, &employeeID, &content, &loggedOn, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":          id,
		"company_id":  cid,
		"employee_id": employeeID,
		"content":     content,
		"logged_on":   loggedOn.Format("2006-01-02"),
		"created_at":  createdAt.UTC().Format(time.RFC3339),
		"updated_at":  updatedAt.UTC().Format(time.RFC3339),
	}, nil
}
