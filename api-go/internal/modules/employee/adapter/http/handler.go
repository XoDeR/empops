// Package http contains the employee module's Chi handlers.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/mediaurl"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

var validRoles = map[string]bool{
	"administrator": true,
	"hr":            true,
	"employee":      true,
}

var validStatusTypes = map[string]bool{
	"internal": true,
	"external": true,
}

// Handler serves the employee module's HTTP routes.
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler builds an employee Handler backed by pool.
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

type db interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

type createEmployeeRequest struct {
	Email            string  `json:"email"`
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	HiredAt          *string `json:"hired_at"`
	PositionID       *string `json:"position_id"`
	EmployeeStatusID *string `json:"employee_status_id"`
	Role             *string `json:"role"`
}

type updateEmployeeRequest struct {
	Email            *string `json:"email"`
	FirstName        *string `json:"first_name"`
	LastName         *string `json:"last_name"`
	HiredAt          *string `json:"hired_at"`
	PositionID       *string `json:"position_id"`
	EmployeeStatusID *string `json:"employee_status_id"`
	Locked           *bool   `json:"locked"`
	Role             *string `json:"role"`
}

type createPositionRequest struct {
	Title string `json:"title"`
}

type updatePositionRequest struct {
	Title string `json:"title"`
}

type createEmployeeStatusRequest struct {
	Name *string `json:"name"`
	Type *string `json:"type"`
}

type updateEmployeeStatusRequest struct {
	Name *string `json:"name"`
	Type *string `json:"type"`
}

// ListEmployees handles GET /companies/{companyId}/employees.
func (h *Handler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")

	rows, err := h.pool.Query(r.Context(), `
		SELECT e.id FROM employees e
		WHERE e.company_id = $1
		ORDER BY e.last_name, e.first_name`, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list employees failed", err.Error())
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		payload, err := employeePayload(r.Context(), h.pool, id, companyID, false)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "employee payload failed", err.Error())
			return
		}
		list = append(list, payload)
	}
	if list == nil {
		list = []map[string]interface{}{}
	}

	response.OK(w, "", list)
}

// CreateEmployee handles POST /companies/{companyId}/employees.
func (h *Handler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")

	var req createEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	if req.Email == "" || req.FirstName == "" || req.LastName == "" {
		response.Fail(w, http.StatusBadRequest, "email, first_name and last_name are required", nil)
		return
	}

	role := "employee"
	if req.Role != nil {
		role = strings.TrimSpace(*req.Role)
		if !validRoles[role] {
			response.Fail(w, http.StatusBadRequest, "invalid role", nil)
			return
		}
	}

	hiredAt, err := parseOptionalDate(req.HiredAt)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, "invalid hired_at", nil)
		return
	}

	employeeID := uuidv7.New()
	var id string
	err = h.pool.QueryRow(r.Context(), `
		INSERT INTO employees (
			id, company_id, user_id, email, first_name, last_name, hired_at,
			position_id, employee_status_id, locked, created_at, updated_at
		) VALUES ($1, $2, NULL, $3, $4, $5, $6, $7, $8, false, now(), now())
		RETURNING id`,
		employeeID, companyID, req.Email, req.FirstName, req.LastName, hiredAt,
		req.PositionID, req.EmployeeStatusID,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			response.Fail(w, http.StatusConflict, "Employee email already exists in this company", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "create employee failed", err.Error())
		return
	}

	if err := assignRole(r.Context(), h.pool, id, role); err != nil {
		response.Fail(w, http.StatusInternalServerError, "assign role failed", err.Error())
		return
	}

	payload, err := employeePayload(r.Context(), h.pool, id, companyID, false)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "employee payload failed", err.Error())
		return
	}

	response.Created(w, "Employee created", payload)
}

// ShowEmployee handles GET /companies/{companyId}/employees/{employeeId}.
func (h *Handler) ShowEmployee(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	employeeID := chi.URLParam(r, "employeeId")

	if !h.employeeExists(r, companyID, employeeID) {
		response.Fail(w, http.StatusNotFound, "Employee not found", nil)
		return
	}

	payload, err := employeePayload(r.Context(), h.pool, employeeID, companyID, false)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "employee payload failed", err.Error())
		return
	}

	response.OK(w, "", payload)
}

// UpdateEmployee handles PATCH /companies/{companyId}/employees/{employeeId}.
func (h *Handler) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
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

	canManage := member.HasPermission("employees.update")
	isSelf := member.EmployeeID == employeeID
	if !canManage && !isSelf {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	var req updateEmployeeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	var email, firstName, lastName string
	var hiredAt *time.Time
	var positionID, statusID *string
	var locked bool

	err := h.pool.QueryRow(r.Context(), `
		SELECT email, first_name, last_name, hired_at, position_id, employee_status_id, locked
		FROM employees WHERE id = $1 AND company_id = $2`,
		employeeID, companyID,
	).Scan(&email, &firstName, &lastName, &hiredAt, &positionID, &statusID, &locked)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "employee lookup failed", err.Error())
		return
	}

	if req.FirstName != nil {
		firstName = strings.TrimSpace(*req.FirstName)
	}
	if req.LastName != nil {
		lastName = strings.TrimSpace(*req.LastName)
	}

	if canManage {
		if req.Email != nil {
			email = strings.TrimSpace(*req.Email)
		}
		if req.HiredAt != nil {
			hiredAt, err = parseOptionalDate(req.HiredAt)
			if err != nil {
				response.Fail(w, http.StatusBadRequest, "invalid hired_at", nil)
				return
			}
		}
		if req.PositionID != nil {
			positionID = req.PositionID
		}
		if req.EmployeeStatusID != nil {
			statusID = req.EmployeeStatusID
		}
		if req.Locked != nil {
			locked = *req.Locked
		}
		if req.Role != nil {
			role := strings.TrimSpace(*req.Role)
			if !validRoles[role] {
				response.Fail(w, http.StatusBadRequest, "invalid role", nil)
				return
			}
			if err := syncRole(r.Context(), h.pool, employeeID, role); err != nil {
				response.Fail(w, http.StatusInternalServerError, "sync role failed", err.Error())
				return
			}
		}
	}

	_, err = h.pool.Exec(r.Context(), `
		UPDATE employees
		SET email = $3, first_name = $4, last_name = $5, hired_at = $6,
			position_id = $7, employee_status_id = $8, locked = $9, updated_at = now()
		WHERE id = $1 AND company_id = $2`,
		employeeID, companyID, email, firstName, lastName, hiredAt, positionID, statusID, locked,
	)
	if err != nil {
		if isUniqueViolation(err) {
			response.Fail(w, http.StatusConflict, "Employee email already exists in this company", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "update employee failed", err.Error())
		return
	}

	payload, err := employeePayload(r.Context(), h.pool, employeeID, companyID, false)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "employee payload failed", err.Error())
		return
	}

	response.OK(w, "Employee updated", payload)
}

// DeleteEmployee handles DELETE /companies/{companyId}/employees/{employeeId}.
func (h *Handler) DeleteEmployee(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	employeeID := chi.URLParam(r, "employeeId")

	tag, err := h.pool.Exec(r.Context(), `DELETE FROM employees WHERE id = $1 AND company_id = $2`, employeeID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete employee failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Employee not found", nil)
		return
	}

	response.OK(w, "Employee deleted", nil)
}

// InviteEmployee handles POST /companies/{companyId}/employees/{employeeId}/invite.
func (h *Handler) InviteEmployee(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	employeeID := chi.URLParam(r, "employeeId")

	var userID *string
	err := h.pool.QueryRow(r.Context(), `
		SELECT user_id FROM employees WHERE id = $1 AND company_id = $2`, employeeID, companyID,
	).Scan(&userID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Employee not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "employee lookup failed", err.Error())
		return
	}
	if userID != nil {
		response.Fail(w, http.StatusConflict, "Employee already linked to a user", nil)
		return
	}

	link := uuidv7.New()
	_, err = h.pool.Exec(r.Context(), `
		UPDATE employees SET invitation_link = $2, invitation_used_at = NULL, updated_at = now()
		WHERE id = $1`, employeeID, link)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "set invitation failed", err.Error())
		return
	}

	payload, err := employeePayload(r.Context(), h.pool, employeeID, companyID, true)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "employee payload failed", err.Error())
		return
	}

	response.OK(w, "Invitation created", payload)
}

// ListPositions handles GET /companies/{companyId}/positions.
func (h *Handler) ListPositions(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")

	rows, err := h.pool.Query(r.Context(), `
		SELECT id, company_id, title FROM positions
		WHERE company_id = $1 ORDER BY title`, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list positions failed", err.Error())
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var id, cid, title string
		if err := rows.Scan(&id, &cid, &title); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		list = append(list, map[string]interface{}{
			"id":         id,
			"company_id": cid,
			"title":      title,
		})
	}
	if list == nil {
		list = []map[string]interface{}{}
	}

	response.OK(w, "", list)
}

// CreatePosition handles POST /companies/{companyId}/positions.
func (h *Handler) CreatePosition(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")

	var req createPositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		response.Fail(w, http.StatusBadRequest, "title is required", nil)
		return
	}

	id := uuidv7.New()
	var outID, outCID, outTitle string
	err := h.pool.QueryRow(r.Context(), `
		INSERT INTO positions (id, company_id, title, created_at, updated_at)
		VALUES ($1, $2, $3, now(), now())
		RETURNING id, company_id, title`, id, companyID, req.Title,
	).Scan(&outID, &outCID, &outTitle)
	if err != nil {
		if isUniqueViolation(err) {
			response.Fail(w, http.StatusConflict, "Position title already exists in this company", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "create position failed", err.Error())
		return
	}

	response.Created(w, "Position created", map[string]interface{}{
		"id":         outID,
		"company_id": outCID,
		"title":      outTitle,
	})
}

// UpdatePosition handles PATCH /companies/{companyId}/positions/{positionId}.
func (h *Handler) UpdatePosition(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	positionID := chi.URLParam(r, "positionId")

	var req updatePositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		response.Fail(w, http.StatusBadRequest, "title is required", nil)
		return
	}

	var outID, outCID, outTitle string
	err := h.pool.QueryRow(r.Context(), `
		UPDATE positions SET title = $3, updated_at = now()
		WHERE id = $1 AND company_id = $2
		RETURNING id, company_id, title`, positionID, companyID, req.Title,
	).Scan(&outID, &outCID, &outTitle)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Position not found", nil)
		return
	}
	if err != nil {
		if isUniqueViolation(err) {
			response.Fail(w, http.StatusConflict, "Position title already exists in this company", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "update position failed", err.Error())
		return
	}

	response.OK(w, "Position updated", map[string]interface{}{
		"id":         outID,
		"company_id": outCID,
		"title":      outTitle,
	})
}

// DeletePosition handles DELETE /companies/{companyId}/positions/{positionId}.
func (h *Handler) DeletePosition(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	positionID := chi.URLParam(r, "positionId")

	tag, err := h.pool.Exec(r.Context(), `DELETE FROM positions WHERE id = $1 AND company_id = $2`, positionID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete position failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Position not found", nil)
		return
	}

	response.OK(w, "Position deleted", nil)
}

// ListEmployeeStatuses handles GET /companies/{companyId}/employee-statuses.
func (h *Handler) ListEmployeeStatuses(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")

	rows, err := h.pool.Query(r.Context(), `
		SELECT id, company_id, name, type FROM employee_statuses
		WHERE company_id = $1 ORDER BY name`, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list employee statuses failed", err.Error())
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var id, cid, name, typ string
		if err := rows.Scan(&id, &cid, &name, &typ); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		list = append(list, map[string]interface{}{
			"id":         id,
			"company_id": cid,
			"name":       name,
			"type":       typ,
		})
	}
	if list == nil {
		list = []map[string]interface{}{}
	}

	response.OK(w, "", list)
}

// CreateEmployeeStatus handles POST /companies/{companyId}/employee-statuses.
func (h *Handler) CreateEmployeeStatus(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")

	var req createEmployeeStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	if req.Name == nil || strings.TrimSpace(*req.Name) == "" {
		response.Fail(w, http.StatusBadRequest, "name is required", nil)
		return
	}
	name := strings.TrimSpace(*req.Name)
	typ := "internal"
	if req.Type != nil {
		typ = strings.TrimSpace(*req.Type)
		if !validStatusTypes[typ] {
			response.Fail(w, http.StatusBadRequest, "invalid type", nil)
			return
		}
	}

	id := uuidv7.New()
	var outID, outCID, outName, outType string
	err := h.pool.QueryRow(r.Context(), `
		INSERT INTO employee_statuses (id, company_id, name, type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, now(), now())
		RETURNING id, company_id, name, type`, id, companyID, name, typ,
	).Scan(&outID, &outCID, &outName, &outType)
	if err != nil {
		if isUniqueViolation(err) {
			response.Fail(w, http.StatusConflict, "Employee status name already exists in this company", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "create employee status failed", err.Error())
		return
	}

	response.Created(w, "Employee status created", map[string]interface{}{
		"id":         outID,
		"company_id": outCID,
		"name":       outName,
		"type":       outType,
	})
}

// UpdateEmployeeStatus handles PATCH /companies/{companyId}/employee-statuses/{statusId}.
func (h *Handler) UpdateEmployeeStatus(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	statusID := chi.URLParam(r, "statusId")

	var req updateEmployeeStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	var name, typ string
	err := h.pool.QueryRow(r.Context(), `
		SELECT name, type FROM employee_statuses WHERE id = $1 AND company_id = $2`,
		statusID, companyID,
	).Scan(&name, &typ)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Employee status not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "status lookup failed", err.Error())
		return
	}

	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
		if name == "" {
			response.Fail(w, http.StatusBadRequest, "name cannot be empty", nil)
			return
		}
	}
	if req.Type != nil {
		typ = strings.TrimSpace(*req.Type)
		if !validStatusTypes[typ] {
			response.Fail(w, http.StatusBadRequest, "invalid type", nil)
			return
		}
	}

	var outID, outCID, outName, outType string
	err = h.pool.QueryRow(r.Context(), `
		UPDATE employee_statuses SET name = $3, type = $4, updated_at = now()
		WHERE id = $1 AND company_id = $2
		RETURNING id, company_id, name, type`, statusID, companyID, name, typ,
	).Scan(&outID, &outCID, &outName, &outType)
	if err != nil {
		if isUniqueViolation(err) {
			response.Fail(w, http.StatusConflict, "Employee status name already exists in this company", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "update employee status failed", err.Error())
		return
	}

	response.OK(w, "Employee status updated", map[string]interface{}{
		"id":         outID,
		"company_id": outCID,
		"name":       outName,
		"type":       outType,
	})
}

// DeleteEmployeeStatus handles DELETE /companies/{companyId}/employee-statuses/{statusId}.
func (h *Handler) DeleteEmployeeStatus(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	statusID := chi.URLParam(r, "statusId")

	tag, err := h.pool.Exec(r.Context(), `DELETE FROM employee_statuses WHERE id = $1 AND company_id = $2`, statusID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete employee status failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Employee status not found", nil)
		return
	}

	response.OK(w, "Employee status deleted", nil)
}

func (h *Handler) employeeExists(r *http.Request, companyID, employeeID string) bool {
	var exists bool
	err := h.pool.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM employees WHERE id = $1 AND company_id = $2)`,
		employeeID, companyID,
	).Scan(&exists)
	return err == nil && exists
}

func assignRole(ctx context.Context, q db, employeeID, roleName string) error {
	var roleID string
	if err := q.QueryRow(ctx, `SELECT id FROM roles WHERE name = $1`, roleName).Scan(&roleID); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `
		INSERT INTO employee_roles (employee_id, role_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, employeeID, roleID)
	return err
}

func syncRole(ctx context.Context, q db, employeeID, roleName string) error {
	var keepManager bool
	_ = q.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM employee_roles er
			JOIN roles r ON r.id = er.role_id
			WHERE er.employee_id = $1 AND r.name = 'manager'
		)`, employeeID,
	).Scan(&keepManager)

	if _, err := q.Exec(ctx, `DELETE FROM employee_roles WHERE employee_id = $1`, employeeID); err != nil {
		return err
	}
	if err := assignRole(ctx, q, employeeID, roleName); err != nil {
		return err
	}
	if keepManager {
		return assignRole(ctx, q, employeeID, "manager")
	}
	return nil
}

func listEmployeeRoles(ctx context.Context, pool *pgxpool.Pool, employeeID string) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT r.name FROM employee_roles er
		JOIN roles r ON r.id = er.role_id
		WHERE er.employee_id = $1 ORDER BY r.name`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if roles == nil {
		roles = []string{}
	}
	return roles, nil
}

func employeePayload(ctx context.Context, pool *pgxpool.Pool, employeeID, companyID string, includeInvite bool) (map[string]interface{}, error) {
	var (
		id, cid, email, firstName, lastName string
		userID, inviteLink                    *string
		hiredAt, inviteUsedAt                 *time.Time
		locked                                bool
		posID, posTitle                       *string
		stID, stName, stType                  *string
	)

	err := pool.QueryRow(ctx, `
		SELECT e.id, e.company_id, e.user_id, e.email, e.first_name, e.last_name, e.hired_at,
			e.locked, e.invitation_link, e.invitation_used_at,
			p.id, p.title,
			s.id, s.name, s.type
		FROM employees e
		LEFT JOIN positions p ON p.id = e.position_id
		LEFT JOIN employee_statuses s ON s.id = e.employee_status_id
		WHERE e.id = $1 AND e.company_id = $2`,
		employeeID, companyID,
	).Scan(
		&id, &cid, &userID, &email, &firstName, &lastName, &hiredAt,
		&locked, &inviteLink, &inviteUsedAt,
		&posID, &posTitle,
		&stID, &stName, &stType,
	)
	if err != nil {
		return nil, err
	}

	roles, err := listEmployeeRoles(ctx, pool, id)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"id":         id,
		"company_id": cid,
		"user_id":    userID,
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
		"hired_at":   formatDate(hiredAt),
		"locked":     locked,
		"roles":      roles,
	}

	if posID != nil && posTitle != nil {
		payload["position"] = map[string]interface{}{
			"id":    *posID,
			"title": *posTitle,
		}
	} else {
		payload["position"] = nil
	}

	if stID != nil && stName != nil && stType != nil {
		payload["status"] = map[string]interface{}{
			"id":   *stID,
			"name": *stName,
			"type": *stType,
		}
	} else {
		payload["status"] = nil
	}

	if includeInvite {
		payload["invitation_link"] = inviteLink
		if inviteLink != nil {
			payload["invitation_url"] = "/invitations/" + *inviteLink + "/accept"
		} else {
			payload["invitation_url"] = nil
		}
	}

	managers, err := listManagersForEmployee(ctx, pool, id)
	if err != nil {
		return nil, err
	}
	payload["managers"] = managers
	if len(managers) > 0 {
		payload["manager"] = managers[0]
	} else {
		payload["manager"] = nil
	}

	teams, err := listTeamsForEmployee(ctx, pool, id)
	if err != nil {
		return nil, err
	}
	payload["teams"] = teams

	var isManager bool
	for _, role := range roles {
		if role == "manager" {
			isManager = true
			break
		}
	}
	if !isManager {
		_ = pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM direct_reports WHERE manager_id = $1)`, id,
		).Scan(&isManager)
	}
	payload["is_manager"] = isManager
	payload["avatar_url"] = mediaurl.AvatarURL(ctx, pool, id)

	return payload, nil
}

func listManagersForEmployee(ctx context.Context, pool *pgxpool.Pool, employeeID string) ([]map[string]interface{}, error) {
	rows, err := pool.Query(ctx, `
		SELECT e.id, e.first_name, e.last_name, e.email
		FROM direct_reports dr
		JOIN employees e ON e.id = dr.manager_id
		WHERE dr.employee_id = $1
		ORDER BY e.last_name, e.first_name`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []map[string]interface{}{}
	for rows.Next() {
		var id, first, last, email string
		if err := rows.Scan(&id, &first, &last, &email); err != nil {
			return nil, err
		}
		list = append(list, map[string]interface{}{
			"id": id, "first_name": first, "last_name": last, "email": email,
		})
	}
	return list, nil
}

func listTeamsForEmployee(ctx context.Context, pool *pgxpool.Pool, employeeID string) ([]map[string]interface{}, error) {
	rows, err := pool.Query(ctx, `
		SELECT t.id, t.name
		FROM employee_team et
		JOIN teams t ON t.id = et.team_id
		WHERE et.employee_id = $1
		ORDER BY t.name`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []map[string]interface{}{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		list = append(list, map[string]interface{}{"id": id, "name": name})
	}
	return list, nil
}

func formatDate(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

func parseOptionalDate(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
