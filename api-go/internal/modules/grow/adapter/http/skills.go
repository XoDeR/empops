package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

func (h *Handler) ListSkills(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	rows, err := h.pool.Query(r.Context(), `
		SELECT s.id, s.name, COUNT(es.employee_id)::int
		FROM skills s
		LEFT JOIN employee_skill es ON es.skill_id=s.id
		WHERE s.company_id=$1
		GROUP BY s.id, s.name
		ORDER BY s.name`, member.CompanyID)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, name string
		var count int
		if rows.Scan(&id, &name, &count) == nil {
			out = append(out, map[string]interface{}{"id": id, "name": name, "employees_count": count})
		}
	}
	response.OK(w, "", out)
}

func (h *Handler) SearchSkills(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	q := normalizeSkill(r.URL.Query().Get("q"))
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, name FROM skills WHERE company_id=$1 AND name LIKE $2 ORDER BY name LIMIT 20`,
		member.CompanyID, "%"+q+"%")
	if err != nil {
		response.Fail(w, 500, "search failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, name string
		if rows.Scan(&id, &name) == nil {
			out = append(out, map[string]interface{}{"id": id, "name": name})
		}
	}
	response.OK(w, "", out)
}

func (h *Handler) ShowSkill(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	skillID := chi.URLParam(r, "skillId")
	var name string
	err := h.pool.QueryRow(r.Context(), `SELECT name FROM skills WHERE id=$1 AND company_id=$2`, skillID, member.CompanyID).Scan(&name)
	if err == pgx.ErrNoRows {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return
	}
	rows, _ := h.pool.Query(r.Context(), `
		SELECT e.id FROM employee_skill es JOIN employees e ON e.id=es.employee_id WHERE es.skill_id=$1`, skillID)
	emps := []map[string]interface{}{}
	if rows != nil {
		for rows.Next() {
			var eid string
			if rows.Scan(&eid) == nil {
				emps = append(emps, h.employeeSummary(r.Context(), eid))
			}
		}
		rows.Close()
	}
	response.OK(w, "", map[string]interface{}{"id": skillID, "name": name, "employees": emps})
}

func (h *Handler) UpdateSkill(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	skillID := chi.URLParam(r, "skillId")
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.Fail(w, 422, "invalid body", nil)
		return
	}
	name := normalizeSkill(body.Name)
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM skills WHERE company_id=$1 AND name=$2 AND id<>$3)`,
		member.CompanyID, name, skillID).Scan(&exists)
	if exists {
		response.Fail(w, 409, "Skill name already exists", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `
		UPDATE skills SET name=$2, updated_at=now() WHERE id=$1 AND company_id=$3`,
		skillID, name, member.CompanyID)
	if err != nil {
		response.Fail(w, 500, "update failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	response.OK(w, "", map[string]interface{}{"id": skillID, "name": name})
}

func (h *Handler) DestroySkill(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	skillID := chi.URLParam(r, "skillId")
	_, _ = h.pool.Exec(r.Context(), `DELETE FROM employee_skill WHERE skill_id=$1`, skillID)
	tag, err := h.pool.Exec(r.Context(), `DELETE FROM skills WHERE id=$1 AND company_id=$2`, skillID, member.CompanyID)
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

func (h *Handler) EmployeeSkills(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	employeeID := chi.URLParam(r, "employeeId")
	var ok bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND company_id=$2)`, employeeID, member.CompanyID).Scan(&ok)
	if !ok {
		response.Fail(w, 404, "Employee not found", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT s.id, s.name FROM skills s
		JOIN employee_skill es ON es.skill_id=s.id
		WHERE es.employee_id=$1 ORDER BY s.name`, employeeID)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, name string
		if rows.Scan(&id, &name) == nil {
			out = append(out, map[string]interface{}{"id": id, "name": name})
		}
	}
	response.OK(w, "", out)
}

func (h *Handler) AttachSkill(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	employeeID := chi.URLParam(r, "employeeId")
	if employeeID != member.EmployeeID && !h.isHr(member) {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.Fail(w, 422, "invalid body", nil)
		return
	}
	name := normalizeSkill(body.Name)
	if name == "" {
		response.Fail(w, 422, "Skill name required", nil)
		return
	}
	var skillID string
	err := h.pool.QueryRow(r.Context(), `SELECT id FROM skills WHERE company_id=$1 AND name=$2`, member.CompanyID, name).Scan(&skillID)
	if err == pgx.ErrNoRows {
		skillID = uuidv7.New()
		_, err = h.pool.Exec(r.Context(), `INSERT INTO skills (id, company_id, name) VALUES ($1,$2,$3)`, skillID, member.CompanyID, name)
		if err != nil {
			response.Fail(w, 500, "create skill failed", err.Error())
			return
		}
	} else if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return
	}
	_, _ = h.pool.Exec(r.Context(), `
		INSERT INTO employee_skill (employee_id, skill_id) VALUES ($1,$2)
		ON CONFLICT DO NOTHING`, employeeID, skillID)
	response.OK(w, "", map[string]interface{}{"id": skillID, "name": name})
}

func (h *Handler) DetachSkill(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	employeeID := chi.URLParam(r, "employeeId")
	skillID := chi.URLParam(r, "skillId")
	if employeeID != member.EmployeeID && !h.isHr(member) {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	_, _ = h.pool.Exec(r.Context(), `DELETE FROM employee_skill WHERE employee_id=$1 AND skill_id=$2`, employeeID, skillID)
	var count int
	_ = h.pool.QueryRow(r.Context(), `SELECT COUNT(*) FROM employee_skill WHERE skill_id=$1`, skillID).Scan(&count)
	if count == 0 {
		_, _ = h.pool.Exec(r.Context(), `DELETE FROM skills WHERE id=$1 AND company_id=$2`, skillID, member.CompanyID)
	}
	response.OK(w, "", nil)
}
