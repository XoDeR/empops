package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
)

func (h *Handler) GetECoffee(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	var enabled bool
	_ = h.pool.QueryRow(r.Context(), `SELECT e_coffee_enabled FROM companies WHERE id=$1`, member.CompanyID).Scan(&enabled)
	response.OK(w, "", map[string]interface{}{"enabled": enabled})
}

func (h *Handler) UpdateECoffee(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.Fail(w, 422, "invalid body", nil)
		return
	}
	_, err := h.pool.Exec(r.Context(), `UPDATE companies SET e_coffee_enabled=$2, updated_at=now() WHERE id=$1`, member.CompanyID, body.Enabled)
	if err != nil {
		response.Fail(w, 500, "update failed", err.Error())
		return
	}
	response.OK(w, "", map[string]interface{}{"enabled": body.Enabled})
}

func (h *Handler) buildECoffeeMatch(r *http.Request, matchID string) (map[string]interface{}, error) {
	var eCoffeeID, employeeID, withID string
	var happened bool
	var batch int
	err := h.pool.QueryRow(r.Context(), `
		SELECT m.e_coffee_id, m.employee_id, m.with_employee_id, m.happened, e.batch_number
		FROM e_coffee_matches m JOIN e_coffees e ON e.id=m.e_coffee_id
		WHERE m.id=$1`, matchID,
	).Scan(&eCoffeeID, &employeeID, &withID, &happened, &batch)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id": matchID, "e_coffee_id": eCoffeeID, "batch_number": batch,
		"employee":      h.employeeSummary(r.Context(), employeeID),
		"with_employee": h.employeeSummary(r.Context(), withID),
		"happened":      happened,
	}, nil
}

func (h *Handler) CurrentECoffee(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	var enabled bool
	_ = h.pool.QueryRow(r.Context(), `SELECT e_coffee_enabled FROM companies WHERE id=$1`, member.CompanyID).Scan(&enabled)
	if !enabled {
		response.OK(w, "", nil)
		return
	}
	var sessionID string
	err := h.pool.QueryRow(r.Context(), `
		SELECT id FROM e_coffees WHERE company_id=$1 AND active=true LIMIT 1`, member.CompanyID).Scan(&sessionID)
	if err == pgx.ErrNoRows {
		response.OK(w, "", nil)
		return
	}
	var matchID string
	err = h.pool.QueryRow(r.Context(), `
		SELECT id FROM e_coffee_matches
		WHERE e_coffee_id=$1 AND (employee_id=$2 OR with_employee_id=$2) LIMIT 1`,
		sessionID, member.EmployeeID).Scan(&matchID)
	if err == pgx.ErrNoRows {
		response.OK(w, "", nil)
		return
	}
	p, err := h.buildECoffeeMatch(r, matchID)
	if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return
	}
	response.OK(w, "", p)
}

func (h *Handler) MarkECoffeeHappened(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	matchID := chi.URLParam(r, "matchId")
	var employeeID, withID string
	err := h.pool.QueryRow(r.Context(), `
		SELECT m.employee_id, m.with_employee_id FROM e_coffee_matches m
		JOIN e_coffees e ON e.id=m.e_coffee_id
		WHERE m.id=$1 AND e.company_id=$2`, matchID, member.CompanyID,
	).Scan(&employeeID, &withID)
	if err == pgx.ErrNoRows {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if member.EmployeeID != employeeID && member.EmployeeID != withID {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	_, _ = h.pool.Exec(r.Context(), `UPDATE e_coffee_matches SET happened=true, updated_at=now() WHERE id=$1`, matchID)
	p, _ := h.buildECoffeeMatch(r, matchID)
	response.OK(w, "", p)
}

func (h *Handler) EmployeeECoffeeHistory(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	employeeID := chi.URLParam(r, "employeeId")
	if employeeID != member.EmployeeID && !h.isHr(member) {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT m.id FROM e_coffee_matches m
		JOIN e_coffees e ON e.id=m.e_coffee_id
		WHERE e.company_id=$1 AND (m.employee_id=$2 OR m.with_employee_id=$2)
		ORDER BY m.created_at DESC LIMIT 50`, member.CompanyID, employeeID)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			if p, err := h.buildECoffeeMatch(r, id); err == nil {
				out = append(out, p)
			}
		}
	}
	response.OK(w, "", out)
}
