package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/audit"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/notify"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

type createShipRequest struct {
	Title       string   `json:"title"`
	Description *string  `json:"description"`
	EmployeeIDs []string `json:"employee_ids"`
}

// ListShips handles GET /companies/{companyId}/teams/{teamId}/ships.
func (h *Handler) ListShips(w http.ResponseWriter, r *http.Request) {
	actor, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	teamID := chi.URLParam(r, "teamId")

	if !h.teamExists(r.Context(), companyID, teamID) {
		response.Fail(w, http.StatusNotFound, "Team not found", nil)
		return
	}
	if !h.canViewShips(r.Context(), actor, teamID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT id FROM ships WHERE team_id = $1 ORDER BY created_at DESC`, teamID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list ships failed", err.Error())
		return
	}
	defer rows.Close()

	list := []map[string]interface{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		payload, err := h.shipPayload(r.Context(), id)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "ship payload failed", err.Error())
			return
		}
		list = append(list, payload)
	}
	response.OK(w, "", list)
}

// CreateShip handles POST /companies/{companyId}/teams/{teamId}/ships.
// Attaching employee_ids notifies each attached employee (except the actor)
// with an "employee_attached_to_recent_ship" notification.
func (h *Handler) CreateShip(w http.ResponseWriter, r *http.Request) {
	actor, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	teamID := chi.URLParam(r, "teamId")

	if !h.teamExists(r.Context(), companyID, teamID) {
		response.Fail(w, http.StatusNotFound, "Team not found", nil)
		return
	}
	if !h.canCreateShip(r.Context(), actor, teamID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	var req createShipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		response.Fail(w, http.StatusBadRequest, "title is required", nil)
		return
	}

	authorName, err := h.employeeFullName(r.Context(), actor.EmployeeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "author lookup failed", err.Error())
		return
	}

	validEmployeeIDs := make([]string, 0, len(req.EmployeeIDs))
	for _, employeeID := range req.EmployeeIDs {
		employeeID = strings.TrimSpace(employeeID)
		if employeeID == "" {
			continue
		}
		if h.employeeExists(r.Context(), companyID, employeeID) {
			validEmployeeIDs = append(validEmployeeIDs, employeeID)
		}
	}

	id := uuidv7.New()
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(), `
		INSERT INTO ships (id, company_id, team_id, author_id, author_name, title, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, companyID, teamID, actor.EmployeeID, authorName, req.Title, req.Description,
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create ship failed", err.Error())
		return
	}

	for _, employeeID := range validEmployeeIDs {
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO employee_ship (employee_id, ship_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, employeeID, id); err != nil {
			response.Fail(w, http.StatusInternalServerError, "attach employee failed", err.Error())
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "ship.created", actor.EmployeeID, strPtr("ship"), &id, map[string]interface{}{
		"team_id": teamID,
		"title":   req.Title,
	})

	for _, employeeID := range validEmployeeIDs {
		if employeeID == actor.EmployeeID {
			continue
		}
		_ = notify.Create(r.Context(), h.pool, companyID, employeeID, "employee_attached_to_recent_ship", map[string]interface{}{
			"ship_title":  req.Title,
			"team_id":     teamID,
			"ship_id":     id,
			"author_name": authorName,
		})
	}

	payload, err := h.shipPayload(r.Context(), id)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "ship payload failed", err.Error())
		return
	}
	response.Created(w, "Ship created", payload)
}

// ShowShip handles GET /companies/{companyId}/teams/{teamId}/ships/{shipId}.
func (h *Handler) ShowShip(w http.ResponseWriter, r *http.Request) {
	actor, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	teamID := chi.URLParam(r, "teamId")
	shipID := chi.URLParam(r, "shipId")

	if !h.teamExists(r.Context(), companyID, teamID) {
		response.Fail(w, http.StatusNotFound, "Team not found", nil)
		return
	}
	if !h.canViewShips(r.Context(), actor, teamID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	payload, err := h.shipPayload(r.Context(), shipID)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(w, http.StatusNotFound, "Ship not found", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "ship payload failed", err.Error())
		return
	}
	response.OK(w, "", payload)
}

// DeleteShip handles DELETE /companies/{companyId}/teams/{teamId}/ships/{shipId}.
func (h *Handler) DeleteShip(w http.ResponseWriter, r *http.Request) {
	actor, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	teamID := chi.URLParam(r, "teamId")
	shipID := chi.URLParam(r, "shipId")

	var authorID *string
	err := h.pool.QueryRow(r.Context(), `
		SELECT author_id FROM ships WHERE id = $1 AND team_id = $2 AND company_id = $3`,
		shipID, teamID, companyID,
	).Scan(&authorID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Ship not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "ship lookup failed", err.Error())
		return
	}

	isAuthor := authorID != nil && *authorID == actor.EmployeeID
	if !isAuthor && !actor.HasPermission("ships.delete") {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	_, err = h.pool.Exec(r.Context(), `DELETE FROM ships WHERE id = $1`, shipID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete ship failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "ship.deleted", actor.EmployeeID, strPtr("ship"), &shipID, nil)
	response.OK(w, "Ship deleted", nil)
}

func (h *Handler) canViewShips(ctx context.Context, actor companyauth.Member, teamID string) bool {
	if actor.HasPermission("ships.view") {
		return true
	}
	return h.employeeInTeam(ctx, teamID, actor.EmployeeID)
}

func (h *Handler) canCreateShip(ctx context.Context, actor companyauth.Member, teamID string) bool {
	if actor.HasPermission("ships.create") {
		return true
	}
	return h.employeeInTeam(ctx, teamID, actor.EmployeeID)
}

func (h *Handler) shipPayload(ctx context.Context, shipID string) (map[string]interface{}, error) {
	var id, cid, teamID, authorName, title string
	var description *string
	var authorID *string
	var createdAt, updatedAt time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT id, company_id, team_id, author_id, author_name, title, description, created_at, updated_at
		FROM ships WHERE id = $1`, shipID,
	).Scan(&id, &cid, &teamID, &authorID, &authorName, &title, &description, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	employees, err := h.shipEmployees(ctx, id)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":          id,
		"company_id":  cid,
		"team_id":     teamID,
		"author_id":   authorID,
		"author_name": authorName,
		"title":       title,
		"description": description,
		"employees":   employees,
		"created_at":  createdAt.UTC().Format(time.RFC3339),
		"updated_at":  updatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (h *Handler) shipEmployees(ctx context.Context, shipID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT e.id, e.first_name, e.last_name, e.email
		FROM employee_ship es
		JOIN employees e ON e.id = es.employee_id
		WHERE es.ship_id = $1
		ORDER BY e.last_name, e.first_name`, shipID)
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
			"id":         id,
			"first_name": first,
			"last_name":  last,
			"email":      email,
		})
	}
	return list, nil
}
