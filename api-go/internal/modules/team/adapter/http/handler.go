package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/audit"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

// Handler serves team HTTP endpoints.
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler constructs a team Handler.
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

type createTeamRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type updateTeamRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type setLeadRequest struct {
	EmployeeID *string `json:"employee_id"`
}

// ListTeams handles GET /companies/{companyId}/teams.
func (h *Handler) ListTeams(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	rows, err := h.pool.Query(r.Context(), `
		SELECT id FROM teams WHERE company_id = $1 ORDER BY name`, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list teams failed", err.Error())
		return
	}
	defer rows.Close()

	list := []map[string]interface{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			response.Fail(w, http.StatusInternalServerError, "list teams failed", err.Error())
			return
		}
		payload, err := h.teamPayload(r.Context(), id, companyID)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "list teams failed", err.Error())
			return
		}
		list = append(list, payload)
	}
	response.OK(w, "", list)
}

// CreateTeam handles POST /companies/{companyId}/teams.
func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	member, _ := companyauth.MemberFromContext(r.Context())

	var req createTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "name is required", nil)
		return
	}

	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO teams (id, company_id, name, description)
		VALUES ($1, $2, $3, $4)`,
		id, companyID, name, req.Description,
	)
	if err != nil {
		if strings.Contains(err.Error(), "teams_company_id_name_key") || strings.Contains(err.Error(), "duplicate") {
			response.Fail(w, http.StatusUnprocessableEntity, "Team name already exists in this company", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "create team failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "team.created", member.EmployeeID, strPtr("team"), &id, map[string]interface{}{
		"name": name,
	})

	payload, err := h.teamPayload(r.Context(), id, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create team failed", err.Error())
		return
	}
	response.Created(w, "Team created", payload)
}

// ShowTeam handles GET /companies/{companyId}/teams/{teamId}.
func (h *Handler) ShowTeam(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	teamID := chi.URLParam(r, "teamId")
	payload, err := h.teamPayload(r.Context(), teamID, companyID)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(w, http.StatusNotFound, "Team not found", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "get team failed", err.Error())
		return
	}
	response.OK(w, "", payload)
}

// UpdateTeam handles PATCH /companies/{companyId}/teams/{teamId}.
func (h *Handler) UpdateTeam(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	teamID := chi.URLParam(r, "teamId")
	member, _ := companyauth.MemberFromContext(r.Context())

	var req updateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}

	var name string
	var description *string
	err := h.pool.QueryRow(r.Context(), `
		SELECT name, description FROM teams WHERE id = $1 AND company_id = $2`,
		teamID, companyID,
	).Scan(&name, &description)
	if err != nil {
		response.Fail(w, http.StatusNotFound, "Team not found", nil)
		return
	}

	newName := name
	if req.Name != nil {
		newName = strings.TrimSpace(*req.Name)
		if newName == "" {
			response.Fail(w, http.StatusUnprocessableEntity, "name is required", nil)
			return
		}
	}
	newDesc := description
	if req.Description != nil {
		newDesc = req.Description
	}

	_, err = h.pool.Exec(r.Context(), `
		UPDATE teams SET name = $1, description = $2, updated_at = now()
		WHERE id = $3 AND company_id = $4`,
		newName, newDesc, teamID, companyID,
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update team failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "team.updated", member.EmployeeID, strPtr("team"), &teamID, map[string]interface{}{
		"name": newName,
	})

	payload, err := h.teamPayload(r.Context(), teamID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update team failed", err.Error())
		return
	}
	response.OK(w, "Team updated", payload)
}

// DeleteTeam handles DELETE /companies/{companyId}/teams/{teamId}.
func (h *Handler) DeleteTeam(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	teamID := chi.URLParam(r, "teamId")
	member, _ := companyauth.MemberFromContext(r.Context())

	var name string
	err := h.pool.QueryRow(r.Context(), `
		SELECT name FROM teams WHERE id = $1 AND company_id = $2`, teamID, companyID,
	).Scan(&name)
	if err != nil {
		response.Fail(w, http.StatusNotFound, "Team not found", nil)
		return
	}

	_, err = h.pool.Exec(r.Context(), `DELETE FROM teams WHERE id = $1 AND company_id = $2`, teamID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete team failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "team.deleted", member.EmployeeID, nil, nil, map[string]interface{}{
		"name":    name,
		"team_id": teamID,
	})
	response.OK(w, "Team deleted", nil)
}

// AddMember handles POST /companies/{companyId}/teams/{teamId}/members/{employeeId}.
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	teamID := chi.URLParam(r, "teamId")
	employeeID := chi.URLParam(r, "employeeId")
	member, _ := companyauth.MemberFromContext(r.Context())

	if !h.teamExists(r.Context(), companyID, teamID) || !h.employeeExists(r.Context(), companyID, employeeID) {
		response.Fail(w, http.StatusNotFound, "Team or employee not found", nil)
		return
	}

	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO employee_team (employee_id, team_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, employeeID, teamID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "add member failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "team.member_added", member.EmployeeID, strPtr("team"), &teamID, map[string]interface{}{
		"employee_id": employeeID,
	})

	payload, _ := h.teamPayload(r.Context(), teamID, companyID)
	response.OK(w, "Member added", payload)
}

// RemoveMember handles DELETE /companies/{companyId}/teams/{teamId}/members/{employeeId}.
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	teamID := chi.URLParam(r, "teamId")
	employeeID := chi.URLParam(r, "employeeId")
	member, _ := companyauth.MemberFromContext(r.Context())

	if !h.teamExists(r.Context(), companyID, teamID) {
		response.Fail(w, http.StatusNotFound, "Team not found", nil)
		return
	}

	_, _ = h.pool.Exec(r.Context(), `
		UPDATE teams SET team_leader_id = NULL, updated_at = now()
		WHERE id = $1 AND company_id = $2 AND team_leader_id = $3`,
		teamID, companyID, employeeID,
	)
	_, err := h.pool.Exec(r.Context(), `
		DELETE FROM employee_team WHERE team_id = $1 AND employee_id = $2`, teamID, employeeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "remove member failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "team.member_removed", member.EmployeeID, strPtr("team"), &teamID, map[string]interface{}{
		"employee_id": employeeID,
	})

	payload, _ := h.teamPayload(r.Context(), teamID, companyID)
	response.OK(w, "Member removed", payload)
}

// SetLead handles PUT /companies/{companyId}/teams/{teamId}/lead.
func (h *Handler) SetLead(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	teamID := chi.URLParam(r, "teamId")
	member, _ := companyauth.MemberFromContext(r.Context())

	var req setLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "invalid JSON body", nil)
		return
	}

	if !h.teamExists(r.Context(), companyID, teamID) {
		response.Fail(w, http.StatusNotFound, "Team not found", nil)
		return
	}

	if req.EmployeeID == nil || *req.EmployeeID == "" {
		_, err := h.pool.Exec(r.Context(), `
			UPDATE teams SET team_leader_id = NULL, updated_at = now()
			WHERE id = $1 AND company_id = $2`, teamID, companyID)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "clear lead failed", err.Error())
			return
		}
		_ = audit.LogEmployee(r.Context(), h.pool, companyID, "team.lead_cleared", member.EmployeeID, strPtr("team"), &teamID, nil)
	} else {
		leadID := *req.EmployeeID
		if !h.employeeExists(r.Context(), companyID, leadID) {
			response.Fail(w, http.StatusNotFound, "Employee not found", nil)
			return
		}
		_, err := h.pool.Exec(r.Context(), `
			INSERT INTO employee_team (employee_id, team_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, leadID, teamID)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "set lead failed", err.Error())
			return
		}
		_, err = h.pool.Exec(r.Context(), `
			UPDATE teams SET team_leader_id = $1, updated_at = now()
			WHERE id = $2 AND company_id = $3`, leadID, teamID, companyID)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "set lead failed", err.Error())
			return
		}
		_ = audit.LogEmployee(r.Context(), h.pool, companyID, "team.lead_set", member.EmployeeID, strPtr("team"), &teamID, map[string]interface{}{
			"employee_id": leadID,
		})
	}

	payload, _ := h.teamPayload(r.Context(), teamID, companyID)
	response.OK(w, "Team lead updated", payload)
}

func (h *Handler) teamExists(ctx context.Context, companyID, teamID string) bool {
	var exists bool
	_ = h.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM teams WHERE id = $1 AND company_id = $2)`,
		teamID, companyID,
	).Scan(&exists)
	return exists
}

func (h *Handler) employeeExists(ctx context.Context, companyID, employeeID string) bool {
	var exists bool
	_ = h.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM employees WHERE id = $1 AND company_id = $2)`,
		employeeID, companyID,
	).Scan(&exists)
	return exists
}

func (h *Handler) teamPayload(ctx context.Context, teamID, companyID string) (map[string]interface{}, error) {
	var id, cid, name string
	var description, leaderID *string
	err := h.pool.QueryRow(ctx, `
		SELECT id, company_id, name, description, team_leader_id
		FROM teams WHERE id = $1 AND company_id = $2`,
		teamID, companyID,
	).Scan(&id, &cid, &name, &description, &leaderID)
	if err != nil {
		return nil, err
	}

	members, err := h.listMembers(ctx, teamID)
	if err != nil {
		return nil, err
	}

	var leader interface{}
	if leaderID != nil {
		leader, _ = h.employeeSummary(ctx, *leaderID)
	}

	return map[string]interface{}{
		"id":           id,
		"company_id":   cid,
		"name":         name,
		"description":  description,
		"leader":       leader,
		"members":      members,
		"member_count": len(members),
	}, nil
}

func (h *Handler) listMembers(ctx context.Context, teamID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT e.id, e.first_name, e.last_name, e.email
		FROM employee_team et
		JOIN employees e ON e.id = et.employee_id
		WHERE et.team_id = $1
		ORDER BY e.last_name, e.first_name`, teamID)
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

func (h *Handler) employeeSummary(ctx context.Context, employeeID string) (map[string]interface{}, error) {
	var id, first, last, email string
	err := h.pool.QueryRow(ctx, `
		SELECT id, first_name, last_name, email FROM employees WHERE id = $1`, employeeID,
	).Scan(&id, &first, &last, &email)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":         id,
		"first_name": first,
		"last_name":  last,
		"email":      email,
	}, nil
}

func strPtr(s string) *string { return &s }
