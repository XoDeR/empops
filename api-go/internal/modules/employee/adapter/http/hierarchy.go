package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/audit"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

type assignManagerRequest struct {
	ManagerID string `json:"manager_id"`
}

// ListManagers handles GET /companies/{companyId}/employees/{employeeId}/managers.
func (h *Handler) ListManagers(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	employeeID := chi.URLParam(r, "employeeId")
	if !h.employeeExists(r, companyID, employeeID) {
		response.Fail(w, http.StatusNotFound, "Employee not found", nil)
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT e.id, e.first_name, e.last_name, e.email
		FROM direct_reports dr
		JOIN employees e ON e.id = dr.manager_id
		WHERE dr.company_id = $1 AND dr.employee_id = $2
		ORDER BY e.last_name, e.first_name`, companyID, employeeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list managers failed", err.Error())
		return
	}
	defer rows.Close()

	list := []map[string]interface{}{}
	for rows.Next() {
		var id, first, last, email string
		if err := rows.Scan(&id, &first, &last, &email); err != nil {
			response.Fail(w, http.StatusInternalServerError, "list managers failed", err.Error())
			return
		}
		list = append(list, map[string]interface{}{
			"id": id, "first_name": first, "last_name": last, "email": email,
		})
	}
	response.OK(w, "", list)
}

// ListDirectReports handles GET /companies/{companyId}/employees/{employeeId}/direct-reports.
func (h *Handler) ListDirectReports(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	employeeID := chi.URLParam(r, "employeeId")
	if !h.employeeExists(r, companyID, employeeID) {
		response.Fail(w, http.StatusNotFound, "Employee not found", nil)
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT e.id, e.first_name, e.last_name, e.email
		FROM direct_reports dr
		JOIN employees e ON e.id = dr.employee_id
		WHERE dr.company_id = $1 AND dr.manager_id = $2
		ORDER BY e.last_name, e.first_name`, companyID, employeeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list direct reports failed", err.Error())
		return
	}
	defer rows.Close()

	list := []map[string]interface{}{}
	for rows.Next() {
		var id, first, last, email string
		if err := rows.Scan(&id, &first, &last, &email); err != nil {
			response.Fail(w, http.StatusInternalServerError, "list direct reports failed", err.Error())
			return
		}
		list = append(list, map[string]interface{}{
			"id": id, "first_name": first, "last_name": last, "email": email,
		})
	}
	response.OK(w, "", list)
}

// AssignManager handles POST /companies/{companyId}/employees/{employeeId}/managers.
func (h *Handler) AssignManager(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	employeeID := chi.URLParam(r, "employeeId")
	member, _ := companyauth.MemberFromContext(r.Context())

	var req assignManagerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ManagerID == "" {
		response.Fail(w, http.StatusBadRequest, "manager_id is required", nil)
		return
	}

	if employeeID == req.ManagerID {
		response.Fail(w, http.StatusUnprocessableEntity, "Cannot assign employee as their own manager", nil)
		return
	}
	if !h.employeeExists(r, companyID, employeeID) || !h.employeeExists(r, companyID, req.ManagerID) {
		response.Fail(w, http.StatusNotFound, "Employee not found", nil)
		return
	}

	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO direct_reports (id, company_id, manager_id, employee_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (manager_id, employee_id) DO NOTHING`,
		id, companyID, req.ManagerID, employeeID,
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "assign manager failed", err.Error())
		return
	}

	_ = syncManagerRole(r.Context(), h.pool, req.ManagerID)
	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "hierarchy.manager_assigned", member.EmployeeID, strPtr("employee"), &employeeID, map[string]interface{}{
		"manager_id":  req.ManagerID,
		"employee_id": employeeID,
	})

	manager, _ := employeeSummary(r.Context(), h.pool, req.ManagerID)
	employee, _ := employeeSummary(r.Context(), h.pool, employeeID)
	response.Created(w, "Manager assigned", map[string]interface{}{
		"manager":  manager,
		"employee": employee,
	})
}

// UnassignManager handles DELETE /companies/{companyId}/employees/{employeeId}/managers/{managerId}.
func (h *Handler) UnassignManager(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	employeeID := chi.URLParam(r, "employeeId")
	managerID := chi.URLParam(r, "managerId")
	member, _ := companyauth.MemberFromContext(r.Context())

	tag, err := h.pool.Exec(r.Context(), `
		DELETE FROM direct_reports
		WHERE company_id = $1 AND manager_id = $2 AND employee_id = $3`,
		companyID, managerID, employeeID,
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "unassign manager failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Manager relationship not found", nil)
		return
	}

	_ = syncManagerRole(r.Context(), h.pool, managerID)
	_, _ = h.pool.Exec(r.Context(), `
		UPDATE expenses SET status = 'accounting_approval', updated_at = now()
		WHERE company_id = $1 AND employee_id = $2 AND status = 'manager_approval'
		  AND NOT EXISTS (
			SELECT 1 FROM direct_reports
			WHERE company_id = $1 AND employee_id = $2
		  )`,
		companyID, employeeID,
	)
	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "hierarchy.manager_unassigned", member.EmployeeID, strPtr("employee"), &employeeID, map[string]interface{}{
		"manager_id":  managerID,
		"employee_id": employeeID,
	})
	response.OK(w, "Manager unassigned", nil)
}

func syncManagerRole(ctx context.Context, pool *pgxpool.Pool, managerID string) error {
	var count int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM direct_reports WHERE manager_id = $1`, managerID,
	).Scan(&count); err != nil {
		return err
	}

	var hasManager bool
	_ = pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM employee_roles er
			JOIN roles r ON r.id = er.role_id
			WHERE er.employee_id = $1 AND r.name = 'manager'
		)`, managerID,
	).Scan(&hasManager)

	if count > 0 && !hasManager {
		return assignRole(ctx, pool, managerID, "manager")
	}
	if count == 0 && hasManager {
		_, err := pool.Exec(ctx, `
			DELETE FROM employee_roles
			WHERE employee_id = $1 AND role_id = (SELECT id FROM roles WHERE name = 'manager')`,
			managerID,
		)
		return err
	}
	return nil
}

func employeeSummary(ctx context.Context, pool *pgxpool.Pool, employeeID string) (map[string]interface{}, error) {
	var id, first, last, email string
	err := pool.QueryRow(ctx, `
		SELECT id, first_name, last_name, email FROM employees WHERE id = $1`, employeeID,
	).Scan(&id, &first, &last, &email)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id": id, "first_name": first, "last_name": last, "email": email,
	}, nil
}

func strPtr(s string) *string { return &s }
