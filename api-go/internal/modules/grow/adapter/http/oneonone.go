package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

func (h *Handler) ensureOpenOneOnOnesForEmployee(ctx context.Context, companyID, employeeID string) {
	rows, err := h.pool.Query(ctx, `
		SELECT manager_id FROM direct_reports WHERE company_id=$1 AND employee_id=$2`, companyID, employeeID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var managerID string
		if rows.Scan(&managerID) == nil {
			_, _ = h.createOrGetOpenOneOnOne(ctx, companyID, managerID, employeeID)
		}
	}
}

func (h *Handler) ensureOpenOneOnOnesForManager(ctx context.Context, companyID, managerID string) {
	rows, err := h.pool.Query(ctx, `
		SELECT employee_id FROM direct_reports dr
		JOIN employees e ON e.id=dr.employee_id
		WHERE dr.company_id=$1 AND dr.manager_id=$2 AND e.locked=false`, companyID, managerID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var employeeID string
		if rows.Scan(&employeeID) == nil {
			_, _ = h.createOrGetOpenOneOnOne(ctx, companyID, managerID, employeeID)
		}
	}
}

func (h *Handler) createOrGetOpenOneOnOne(ctx context.Context, companyID, managerID, employeeID string) (string, error) {
	var id string
	err := h.pool.QueryRow(ctx, `
		SELECT id FROM one_on_one_entries
		WHERE company_id=$1 AND manager_id=$2 AND employee_id=$3 AND happened=false
		LIMIT 1`, companyID, managerID, employeeID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != pgx.ErrNoRows {
		return "", err
	}
	return h.createOneOnOneEntry(ctx, companyID, managerID, employeeID)
}

func (h *Handler) createOneOnOneEntry(ctx context.Context, companyID, managerID, employeeID string) (string, error) {
	id := uuidv7.New()
	_, err := h.pool.Exec(ctx, `
		INSERT INTO one_on_one_entries (id, company_id, manager_id, employee_id, happened)
		VALUES ($1,$2,$3,$4,false)`, id, companyID, managerID, employeeID)
	if err != nil {
		return "", err
	}
	var prevID string
	err = h.pool.QueryRow(ctx, `
		SELECT id FROM one_on_one_entries
		WHERE company_id=$1 AND manager_id=$2 AND employee_id=$3 AND id<>$4
		ORDER BY created_at DESC LIMIT 1`, companyID, managerID, employeeID, id).Scan(&prevID)
	if err == nil {
		rows, qerr := h.pool.Query(ctx, `
			SELECT description FROM one_on_one_action_items
			WHERE one_on_one_entry_id=$1 AND checked=false`, prevID)
		if qerr == nil {
			for rows.Next() {
				var desc string
				if rows.Scan(&desc) == nil {
					_, _ = h.pool.Exec(ctx, `
						INSERT INTO one_on_one_talking_points (id, one_on_one_entry_id, description, checked)
						VALUES ($1,$2,$3,false)`, uuidv7.New(), id, desc)
				}
			}
			rows.Close()
		}
	}
	return id, nil
}

func (h *Handler) oneOnOnePayload(ctx context.Context, entryID string) (map[string]interface{}, error) {
	var companyID, managerID, employeeID string
	var happened bool
	var happenedAt *time.Time
	var createdAt time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT company_id, manager_id, employee_id, happened, happened_at, created_at
		FROM one_on_one_entries WHERE id=$1`, entryID,
	).Scan(&companyID, &managerID, &employeeID, &happened, &happenedAt, &createdAt)
	if err != nil {
		return nil, err
	}

	talking := []map[string]interface{}{}
	rows, _ := h.pool.Query(ctx, `SELECT id, description, checked FROM one_on_one_talking_points WHERE one_on_one_entry_id=$1 ORDER BY created_at`, entryID)
	if rows != nil {
		for rows.Next() {
			var id, desc string
			var checked bool
			if rows.Scan(&id, &desc, &checked) == nil {
				talking = append(talking, map[string]interface{}{"id": id, "description": desc, "checked": checked})
			}
		}
		rows.Close()
	}
	actions := []map[string]interface{}{}
	rows, _ = h.pool.Query(ctx, `SELECT id, description, checked FROM one_on_one_action_items WHERE one_on_one_entry_id=$1 ORDER BY created_at`, entryID)
	if rows != nil {
		for rows.Next() {
			var id, desc string
			var checked bool
			if rows.Scan(&id, &desc, &checked) == nil {
				actions = append(actions, map[string]interface{}{"id": id, "description": desc, "checked": checked})
			}
		}
		rows.Close()
	}
	notes := []map[string]interface{}{}
	rows, _ = h.pool.Query(ctx, `SELECT id, note, created_at FROM one_on_one_notes WHERE one_on_one_entry_id=$1 ORDER BY created_at`, entryID)
	if rows != nil {
		for rows.Next() {
			var id, note string
			var ca time.Time
			if rows.Scan(&id, &note, &ca) == nil {
				notes = append(notes, map[string]interface{}{"id": id, "note": note, "created_at": ca.UTC().Format(time.RFC3339)})
			}
		}
		rows.Close()
	}

	return map[string]interface{}{
		"id":             entryID,
		"company_id":     companyID,
		"manager":        h.employeeSummary(ctx, managerID),
		"employee":       h.employeeSummary(ctx, employeeID),
		"happened":       happened,
		"happened_at":    isoTime(happenedAt),
		"talking_points": talking,
		"action_items":   actions,
		"notes":          notes,
		"created_at":     createdAt.UTC().Format(time.RFC3339),
	}, nil
}

func (h *Handler) loadEntryForActor(w http.ResponseWriter, r *http.Request) (string, string, string, bool) {
	member, _ := companyauth.MemberFromContext(r.Context())
	entryID := chi.URLParam(r, "entryId")
	var companyID, managerID, employeeID string
	err := h.pool.QueryRow(r.Context(), `
		SELECT company_id, manager_id, employee_id FROM one_on_one_entries WHERE id=$1 AND company_id=$2`,
		entryID, member.CompanyID,
	).Scan(&companyID, &managerID, &employeeID)
	if err == pgx.ErrNoRows {
		response.Fail(w, 404, "Not found", nil)
		return "", "", "", false
	}
	if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return "", "", "", false
	}
	if member.EmployeeID != managerID && member.EmployeeID != employeeID && !h.isHr(member) {
		response.Fail(w, 403, "Forbidden", nil)
		return "", "", "", false
	}
	return entryID, managerID, employeeID, true
}

func (h *Handler) MyOneOnOnes(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	h.ensureOpenOneOnOnesForEmployee(r.Context(), member.CompanyID, member.EmployeeID)
	rows, err := h.pool.Query(r.Context(), `
		SELECT id FROM one_on_one_entries
		WHERE company_id=$1 AND employee_id=$2 AND happened=false ORDER BY created_at`,
		member.CompanyID, member.EmployeeID)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			if p, err := h.oneOnOnePayload(r.Context(), id); err == nil {
				out = append(out, p)
			}
		}
	}
	response.OK(w, "", out)
}

func (h *Handler) ManagerOneOnOnes(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	h.ensureOpenOneOnOnesForManager(r.Context(), member.CompanyID, member.EmployeeID)
	rows, err := h.pool.Query(r.Context(), `
		SELECT id FROM one_on_one_entries
		WHERE company_id=$1 AND manager_id=$2 AND happened=false ORDER BY created_at`,
		member.CompanyID, member.EmployeeID)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			if p, err := h.oneOnOnePayload(r.Context(), id); err == nil {
				out = append(out, p)
			}
		}
	}
	response.OK(w, "", out)
}

func (h *Handler) ShowOneOnOne(w http.ResponseWriter, r *http.Request) {
	entryID, _, _, ok := h.loadEntryForActor(w, r)
	if !ok {
		return
	}
	p, err := h.oneOnOnePayload(r.Context(), entryID)
	if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return
	}
	response.OK(w, "", p)
}

func (h *Handler) MarkOneOnOneHappened(w http.ResponseWriter, r *http.Request) {
	entryID, managerID, employeeID, ok := h.loadEntryForActor(w, r)
	if !ok {
		return
	}
	member, _ := companyauth.MemberFromContext(r.Context())
	var happened bool
	_ = h.pool.QueryRow(r.Context(), `SELECT happened FROM one_on_one_entries WHERE id=$1`, entryID).Scan(&happened)
	if happened {
		response.Fail(w, 409, "One-on-one already marked happened", nil)
		return
	}
	_, err := h.pool.Exec(r.Context(), `
		UPDATE one_on_one_entries SET happened=true, happened_at=now(), updated_at=now() WHERE id=$1`, entryID)
	if err != nil {
		response.Fail(w, 500, "update failed", err.Error())
		return
	}
	_, _ = h.createOneOnOneEntry(r.Context(), member.CompanyID, managerID, employeeID)
	p, _ := h.oneOnOnePayload(r.Context(), entryID)
	response.OK(w, "", p)
}

func (h *Handler) StoreTalkingPoint(w http.ResponseWriter, r *http.Request) {
	h.storeChecklistChild(w, r, "one_on_one_talking_points")
}

func (h *Handler) StoreActionItem(w http.ResponseWriter, r *http.Request) {
	h.storeChecklistChild(w, r, "one_on_one_action_items")
}

func (h *Handler) storeChecklistChild(w http.ResponseWriter, r *http.Request, table string) {
	entryID, _, _, ok := h.loadEntryForActor(w, r)
	if !ok {
		return
	}
	var body struct {
		Description string `json:"description"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Description == "" {
		response.Fail(w, 422, "description required", nil)
		return
	}
	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO `+table+` (id, one_on_one_entry_id, description, checked) VALUES ($1,$2,$3,false)`,
		id, entryID, body.Description)
	if err != nil {
		response.Fail(w, 500, "create failed", err.Error())
		return
	}
	response.OK(w, "", map[string]interface{}{"id": id, "description": body.Description, "checked": false})
}

func (h *Handler) ToggleTalkingPoint(w http.ResponseWriter, r *http.Request) {
	h.toggleChecklistChild(w, r, "one_on_one_talking_points", "pointId")
}

func (h *Handler) ToggleActionItem(w http.ResponseWriter, r *http.Request) {
	h.toggleChecklistChild(w, r, "one_on_one_action_items", "itemId")
}

func (h *Handler) toggleChecklistChild(w http.ResponseWriter, r *http.Request, table, param string) {
	entryID, _, _, ok := h.loadEntryForActor(w, r)
	if !ok {
		return
	}
	childID := chi.URLParam(r, param)
	var desc string
	var checked bool
	err := h.pool.QueryRow(r.Context(), `
		UPDATE `+table+` SET checked = NOT checked, updated_at=now()
		WHERE id=$1 AND one_on_one_entry_id=$2 RETURNING description, checked`,
		childID, entryID).Scan(&desc, &checked)
	if err == pgx.ErrNoRows {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, 500, "update failed", err.Error())
		return
	}
	response.OK(w, "", map[string]interface{}{"id": childID, "description": desc, "checked": checked})
}

func (h *Handler) DestroyTalkingPoint(w http.ResponseWriter, r *http.Request) {
	h.destroyChild(w, r, "one_on_one_talking_points", "pointId")
}

func (h *Handler) DestroyActionItem(w http.ResponseWriter, r *http.Request) {
	h.destroyChild(w, r, "one_on_one_action_items", "itemId")
}

func (h *Handler) DestroyNote(w http.ResponseWriter, r *http.Request) {
	h.destroyChild(w, r, "one_on_one_notes", "noteId")
}

func (h *Handler) destroyChild(w http.ResponseWriter, r *http.Request, table, param string) {
	entryID, _, _, ok := h.loadEntryForActor(w, r)
	if !ok {
		return
	}
	childID := chi.URLParam(r, param)
	tag, err := h.pool.Exec(r.Context(), `DELETE FROM `+table+` WHERE id=$1 AND one_on_one_entry_id=$2`, childID, entryID)
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

func (h *Handler) StoreNote(w http.ResponseWriter, r *http.Request) {
	entryID, _, _, ok := h.loadEntryForActor(w, r)
	if !ok {
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Note == "" {
		response.Fail(w, 422, "note required", nil)
		return
	}
	id := uuidv7.New()
	var createdAt time.Time
	err := h.pool.QueryRow(r.Context(), `
		INSERT INTO one_on_one_notes (id, one_on_one_entry_id, note) VALUES ($1,$2,$3) RETURNING created_at`,
		id, entryID, body.Note).Scan(&createdAt)
	if err != nil {
		response.Fail(w, 500, "create failed", err.Error())
		return
	}
	response.OK(w, "", map[string]interface{}{"id": id, "note": body.Note, "created_at": createdAt.UTC().Format(time.RFC3339)})
}
