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

type createCompanyNewsRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type updateCompanyNewsRequest struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}

// ListCompanyNews handles GET /companies/{companyId}/news.
func (h *Handler) ListCompanyNews(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")

	rows, err := h.pool.Query(r.Context(), `
		SELECT id FROM company_news WHERE company_id = $1 ORDER BY created_at DESC`, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list news failed", err.Error())
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
		payload, err := h.companyNewsPayload(r.Context(), id, companyID)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "news payload failed", err.Error())
			return
		}
		list = append(list, payload)
	}
	response.OK(w, "", list)
}

// CreateCompanyNews handles POST /companies/{companyId}/news.
func (h *Handler) CreateCompanyNews(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")

	var req createCompanyNewsRequest
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

	authorName, err := h.employeeFullName(r.Context(), member.EmployeeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "author lookup failed", err.Error())
		return
	}

	id := uuidv7.New()
	_, err = h.pool.Exec(r.Context(), `
		INSERT INTO company_news (id, company_id, author_id, author_name, title, content)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, companyID, member.EmployeeID, authorName, req.Title, req.Content,
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create news failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "news.created", member.EmployeeID, strPtr("news"), &id, map[string]interface{}{
		"title": req.Title,
	})

	payload, err := h.companyNewsPayload(r.Context(), id, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "news payload failed", err.Error())
		return
	}
	response.Created(w, "News created", payload)
}

// ShowCompanyNews handles GET /companies/{companyId}/news/{newsId}.
func (h *Handler) ShowCompanyNews(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	newsID := chi.URLParam(r, "newsId")

	payload, err := h.companyNewsPayload(r.Context(), newsID, companyID)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(w, http.StatusNotFound, "News not found", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "news payload failed", err.Error())
		return
	}
	response.OK(w, "", payload)
}

// UpdateCompanyNews handles PATCH /companies/{companyId}/news/{newsId}.
func (h *Handler) UpdateCompanyNews(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	newsID := chi.URLParam(r, "newsId")

	var req updateCompanyNewsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	var title, content string
	err := h.pool.QueryRow(r.Context(), `
		SELECT title, content FROM company_news WHERE id = $1 AND company_id = $2`, newsID, companyID,
	).Scan(&title, &content)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "News not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "news lookup failed", err.Error())
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
		UPDATE company_news SET title = $3, content = $4, updated_at = now()
		WHERE id = $1 AND company_id = $2`, newsID, companyID, title, content)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update news failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "news.updated", member.EmployeeID, strPtr("news"), &newsID, nil)

	payload, err := h.companyNewsPayload(r.Context(), newsID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "news payload failed", err.Error())
		return
	}
	response.OK(w, "News updated", payload)
}

// DeleteCompanyNews handles DELETE /companies/{companyId}/news/{newsId}.
func (h *Handler) DeleteCompanyNews(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	newsID := chi.URLParam(r, "newsId")

	tag, err := h.pool.Exec(r.Context(), `DELETE FROM company_news WHERE id = $1 AND company_id = $2`, newsID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete news failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "News not found", nil)
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "news.deleted", member.EmployeeID, strPtr("news"), &newsID, nil)
	response.OK(w, "News deleted", nil)
}

func (h *Handler) companyNewsPayload(ctx context.Context, newsID, companyID string) (map[string]interface{}, error) {
	var id, cid, title, content, authorName string
	var authorID *string
	var createdAt, updatedAt time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT id, company_id, author_id, author_name, title, content, created_at, updated_at
		FROM company_news WHERE id = $1 AND company_id = $2`, newsID, companyID,
	).Scan(&id, &cid, &authorID, &authorName, &title, &content, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":          id,
		"company_id":  cid,
		"author_id":   authorID,
		"author_name": authorName,
		"title":       title,
		"content":     content,
		"created_at":  createdAt.UTC().Format(time.RFC3339),
		"updated_at":  updatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (h *Handler) employeeFullName(ctx context.Context, employeeID string) (string, error) {
	var first, last string
	err := h.pool.QueryRow(ctx, `SELECT first_name, last_name FROM employees WHERE id = $1`, employeeID).Scan(&first, &last)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(first + " " + last), nil
}

func strPtr(s string) *string { return &s }
