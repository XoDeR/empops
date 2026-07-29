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
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

type createTeamNewsRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type updateTeamNewsRequest struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}

// ListTeamNews handles GET /companies/{companyId}/teams/{teamId}/news.
func (h *Handler) ListTeamNews(w http.ResponseWriter, r *http.Request) {
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
	if !h.canViewTeamNews(r.Context(), actor, teamID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT id FROM team_news WHERE team_id = $1 ORDER BY created_at DESC`, teamID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list team news failed", err.Error())
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
		payload, err := h.teamNewsPayload(r.Context(), id)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "team news payload failed", err.Error())
			return
		}
		list = append(list, payload)
	}
	response.OK(w, "", list)
}

// CreateTeamNews handles POST /companies/{companyId}/teams/{teamId}/news.
func (h *Handler) CreateTeamNews(w http.ResponseWriter, r *http.Request) {
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
	if !h.canCreateTeamNews(r.Context(), actor, teamID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	var req createTeamNewsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	if req.Title == "" || req.Content == "" {
		response.Fail(w, http.StatusBadRequest, "title and content are required", nil)
		return
	}

	authorName, err := h.employeeFullName(r.Context(), actor.EmployeeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "author lookup failed", err.Error())
		return
	}

	id := uuidv7.New()
	_, err = h.pool.Exec(r.Context(), `
		INSERT INTO team_news (id, company_id, team_id, author_id, author_name, title, content)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, companyID, teamID, actor.EmployeeID, authorName, req.Title, req.Content,
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create team news failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "team_news.created", actor.EmployeeID, strPtr("team_news"), &id, map[string]interface{}{
		"team_id": teamID,
	})

	payload, err := h.teamNewsPayload(r.Context(), id)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "team news payload failed", err.Error())
		return
	}
	response.Created(w, "Team news created", payload)
}

// ShowTeamNews handles GET /companies/{companyId}/teams/{teamId}/news/{newsId}.
func (h *Handler) ShowTeamNews(w http.ResponseWriter, r *http.Request) {
	actor, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	teamID := chi.URLParam(r, "teamId")
	newsID := chi.URLParam(r, "newsId")

	if !h.teamExists(r.Context(), companyID, teamID) {
		response.Fail(w, http.StatusNotFound, "Team not found", nil)
		return
	}
	if !h.canViewTeamNews(r.Context(), actor, teamID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	payload, err := h.teamNewsPayload(r.Context(), newsID)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(w, http.StatusNotFound, "Team news not found", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "team news payload failed", err.Error())
		return
	}
	response.OK(w, "", payload)
}

// UpdateTeamNews handles PATCH /companies/{companyId}/teams/{teamId}/news/{newsId}.
func (h *Handler) UpdateTeamNews(w http.ResponseWriter, r *http.Request) {
	actor, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	teamID := chi.URLParam(r, "teamId")
	newsID := chi.URLParam(r, "newsId")

	var authorID *string
	var title, content string
	err := h.pool.QueryRow(r.Context(), `
		SELECT author_id, title, content FROM team_news
		WHERE id = $1 AND team_id = $2 AND company_id = $3`, newsID, teamID, companyID,
	).Scan(&authorID, &title, &content)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Team news not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "team news lookup failed", err.Error())
		return
	}

	isAuthor := authorID != nil && *authorID == actor.EmployeeID
	if !isAuthor && !actor.HasPermission("team-news.update") {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	var req updateTeamNewsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	if req.Title != nil {
		title = strings.TrimSpace(*req.Title)
		if title == "" {
			response.Fail(w, http.StatusBadRequest, "title cannot be empty", nil)
			return
		}
	}
	if req.Content != nil {
		content = strings.TrimSpace(*req.Content)
		if content == "" {
			response.Fail(w, http.StatusBadRequest, "content cannot be empty", nil)
			return
		}
	}

	_, err = h.pool.Exec(r.Context(), `
		UPDATE team_news SET title = $2, content = $3, updated_at = now() WHERE id = $1`,
		newsID, title, content,
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update team news failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "team_news.updated", actor.EmployeeID, strPtr("team_news"), &newsID, nil)

	payload, err := h.teamNewsPayload(r.Context(), newsID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "team news payload failed", err.Error())
		return
	}
	response.OK(w, "Team news updated", payload)
}

// DeleteTeamNews handles DELETE /companies/{companyId}/teams/{teamId}/news/{newsId}.
func (h *Handler) DeleteTeamNews(w http.ResponseWriter, r *http.Request) {
	actor, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	teamID := chi.URLParam(r, "teamId")
	newsID := chi.URLParam(r, "newsId")

	var authorID *string
	err := h.pool.QueryRow(r.Context(), `
		SELECT author_id FROM team_news WHERE id = $1 AND team_id = $2 AND company_id = $3`,
		newsID, teamID, companyID,
	).Scan(&authorID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Team news not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "team news lookup failed", err.Error())
		return
	}

	isAuthor := authorID != nil && *authorID == actor.EmployeeID
	if !isAuthor && !actor.HasPermission("team-news.delete") {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	_, err = h.pool.Exec(r.Context(), `DELETE FROM team_news WHERE id = $1`, newsID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete team news failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "team_news.deleted", actor.EmployeeID, strPtr("team_news"), &newsID, nil)
	response.OK(w, "Team news deleted", nil)
}

func (h *Handler) canViewTeamNews(ctx context.Context, actor companyauth.Member, teamID string) bool {
	if actor.HasPermission("team-news.view") {
		return true
	}
	return h.employeeInTeam(ctx, teamID, actor.EmployeeID)
}

func (h *Handler) canCreateTeamNews(ctx context.Context, actor companyauth.Member, teamID string) bool {
	if actor.HasPermission("team-news.create") {
		return true
	}
	return h.employeeInTeam(ctx, teamID, actor.EmployeeID)
}

func (h *Handler) employeeInTeam(ctx context.Context, teamID, employeeID string) bool {
	var exists bool
	_ = h.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM employee_team WHERE team_id = $1 AND employee_id = $2)`,
		teamID, employeeID,
	).Scan(&exists)
	return exists
}

func (h *Handler) employeeFullName(ctx context.Context, employeeID string) (string, error) {
	var first, last string
	err := h.pool.QueryRow(ctx, `SELECT first_name, last_name FROM employees WHERE id = $1`, employeeID).Scan(&first, &last)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(first + " " + last), nil
}

func (h *Handler) teamNewsPayload(ctx context.Context, newsID string) (map[string]interface{}, error) {
	var id, cid, teamID, authorName, title, content string
	var authorID *string
	var createdAt, updatedAt time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT id, company_id, team_id, author_id, author_name, title, content, created_at, updated_at
		FROM team_news WHERE id = $1`, newsID,
	).Scan(&id, &cid, &teamID, &authorID, &authorName, &title, &content, &createdAt, &updatedAt)
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
		"content":     content,
		"created_at":  createdAt.UTC().Format(time.RFC3339),
		"updated_at":  updatedAt.UTC().Format(time.RFC3339),
	}, nil
}
