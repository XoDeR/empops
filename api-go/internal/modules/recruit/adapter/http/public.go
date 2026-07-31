package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

type publicApplyRequest struct {
	Name          string  `json:"name"`
	Email         string  `json:"email"`
	URL           *string `json:"url"`
	DesiredSalary *string `json:"desired_salary"`
	Notes         *string `json:"notes"`
}

func (h *Handler) ListCompanies(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `
		SELECT c.slug, c.name, COUNT(jo.id) AS openings_count
		FROM companies c
		JOIN job_openings jo ON jo.company_id = c.id AND jo.active = true AND jo.fulfilled = false
		GROUP BY c.id, c.slug, c.name
		ORDER BY c.name`)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list companies failed", err.Error())
		return
	}
	defer rows.Close()

	out := []map[string]interface{}{}
	for rows.Next() {
		var slug, name string
		var count int
		if err := rows.Scan(&slug, &name, &count); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		out = append(out, map[string]interface{}{
			"slug":           slug,
			"name":           name,
			"openings_count": count,
		})
	}
	response.OK(w, "", out)
}

func (h *Handler) ListCompanyJobs(w http.ResponseWriter, r *http.Request) {
	companySlug := chi.URLParam(r, "companySlug")
	companyID, err := h.companyIDBySlug(r.Context(), companySlug)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Company not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "company lookup failed", err.Error())
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT title, slug, reference_number FROM job_openings
		WHERE company_id = $1 AND active = true AND fulfilled = false
		ORDER BY title`, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list jobs failed", err.Error())
		return
	}
	defer rows.Close()

	out := []map[string]interface{}{}
	for rows.Next() {
		var title, slug string
		var refNum *string
		if err := rows.Scan(&title, &slug, &refNum); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		out = append(out, map[string]interface{}{
			"title":            title,
			"slug":             slug,
			"reference_number": refNum,
		})
	}
	response.OK(w, "", out)
}

func (h *Handler) ShowJob(w http.ResponseWriter, r *http.Request) {
	companySlug := chi.URLParam(r, "companySlug")
	jobSlug := chi.URLParam(r, "jobSlug")

	companyID, companyName, err := h.companyBySlug(r.Context(), companySlug)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Company not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "company lookup failed", err.Error())
		return
	}

	var title, description, slug string
	var refNum *string
	var pageViews int
	err = h.pool.QueryRow(r.Context(), `
		SELECT title, description, slug, reference_number, page_views FROM job_openings
		WHERE company_id = $1 AND slug = $2 AND active = true AND fulfilled = false`,
		companyID, jobSlug,
	).Scan(&title, &description, &slug, &refNum, &pageViews)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Job opening not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "job lookup failed", err.Error())
		return
	}

	_, _ = h.pool.Exec(r.Context(), `
		UPDATE job_openings SET page_views = page_views + 1, updated_at = now()
		WHERE company_id = $1 AND slug = $2`, companyID, jobSlug)

	response.OK(w, "", map[string]interface{}{
		"title":            title,
		"slug":             slug,
		"description":      description,
		"reference_number": refNum,
		"company": map[string]interface{}{
			"slug": companySlug,
			"name": companyName,
		},
	})
}

func (h *Handler) Apply(w http.ResponseWriter, r *http.Request) {
	companySlug := chi.URLParam(r, "companySlug")
	jobSlug := chi.URLParam(r, "jobSlug")

	companyID, _, err := h.companyBySlug(r.Context(), companySlug)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Company not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "company lookup failed", err.Error())
		return
	}

	var req publicApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	if req.Name == "" || req.Email == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "name and email are required", nil)
		return
	}

	var openingID, templateID string
	err = h.pool.QueryRow(r.Context(), `
		SELECT id, recruiting_stage_template_id FROM job_openings
		WHERE company_id = $1 AND slug = $2 AND active = true AND fulfilled = false`,
		companyID, jobSlug,
	).Scan(&openingID, &templateID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Job opening not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "job lookup failed", err.Error())
		return
	}
	if templateID == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "Opening has no stage template", nil)
		return
	}

	candidateID := uuidv7.New()
	candidateUUID := uuid.NewString()

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(), `
		INSERT INTO candidates (
			id, company_id, job_opening_id, name, email, uuid, url, desired_salary, notes,
			application_completed, rejected, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, false, now(), now())`,
		candidateID, companyID, openingID, req.Name, req.Email, candidateUUID,
		req.URL, req.DesiredSalary, req.Notes)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create candidate failed", err.Error())
		return
	}

	stageRows, err := tx.Query(r.Context(), `
		SELECT name, position FROM recruiting_stages
		WHERE recruiting_stage_template_id = $1 ORDER BY position`, templateID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "load stages failed", err.Error())
		return
	}
	defer stageRows.Close()
	for stageRows.Next() {
		var stageName string
		var stagePos int
		if err := stageRows.Scan(&stageName, &stagePos); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan stages failed", err.Error())
			return
		}
		stageID := uuidv7.New()
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO candidate_stages (id, candidate_id, stage_name, stage_position, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'pending', now(), now())`,
			stageID, candidateID, stageName, stagePos); err != nil {
			response.Fail(w, http.StatusInternalServerError, "create candidate stage failed", err.Error())
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}

	response.Created(w, "Application started", map[string]interface{}{
		"uuid":  candidateUUID,
		"name":  req.Name,
		"email": req.Email,
	})
}

func (h *Handler) PublicListFiles(w http.ResponseWriter, r *http.Request) {
	candidateID, err := h.findIncompletePublicCandidate(r)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Application not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	files, err := h.listCandidateFiles(r.Context(), candidateID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list files failed", err.Error())
		return
	}
	response.OK(w, "", files)
}

func (h *Handler) PublicAttachFile(w http.ResponseWriter, r *http.Request) {
	candidateID, err := h.findIncompletePublicCandidate(r)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Application not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	var req attachFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	payload, err := h.attachCandidateFile(r.Context(), candidateID, req.TemporaryUploadID, req.MediaID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Media not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "attach file failed", err.Error())
		return
	}
	response.Created(w, "File attached", payload)
}

func (h *Handler) PublicDeleteFile(w http.ResponseWriter, r *http.Request) {
	candidateID, err := h.findIncompletePublicCandidate(r)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Application not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	mediaID := chi.URLParam(r, "mediaId")
	err = h.deleteCandidateFile(r.Context(), candidateID, mediaID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "File not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete file failed", err.Error())
		return
	}
	response.OK(w, "File deleted", nil)
}

func (h *Handler) CompleteApplication(w http.ResponseWriter, r *http.Request) {
	candidateID, candidateUUID, err := h.findIncompletePublicCandidateWithUUID(r)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Application not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	_, err = h.pool.Exec(r.Context(), `
		UPDATE candidates SET application_completed = true, updated_at = now() WHERE id = $1`, candidateID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "complete application failed", err.Error())
		return
	}

	response.OK(w, "Application completed", map[string]interface{}{
		"uuid":                  candidateUUID,
		"application_completed": true,
	})
}

func (h *Handler) AbandonApplication(w http.ResponseWriter, r *http.Request) {
	candidateID, err := h.findIncompletePublicCandidate(r)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Application not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	tag, err := h.pool.Exec(r.Context(), `DELETE FROM candidates WHERE id = $1 AND application_completed = false`, candidateID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "abandon application failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusConflict, "Cannot abandon completed application", nil)
		return
	}
	response.OK(w, "Application abandoned", nil)
}

func (h *Handler) companyIDBySlug(ctx context.Context, slug string) (string, error) {
	var id string
	err := h.pool.QueryRow(ctx, `SELECT id FROM companies WHERE slug = $1`, slug).Scan(&id)
	return id, err
}

func (h *Handler) companyBySlug(ctx context.Context, slug string) (id, name string, err error) {
	err = h.pool.QueryRow(ctx, `SELECT id, name FROM companies WHERE slug = $1`, slug).Scan(&id, &name)
	return id, name, err
}

func (h *Handler) findIncompletePublicCandidate(r *http.Request) (string, error) {
	id, _, err := h.findIncompletePublicCandidateWithUUID(r)
	return id, err
}

func (h *Handler) findIncompletePublicCandidateWithUUID(r *http.Request) (candidateID, candidateUUID string, err error) {
	companySlug := chi.URLParam(r, "companySlug")
	jobSlug := chi.URLParam(r, "jobSlug")
	candidateUUIDParam := chi.URLParam(r, "candidateUuid")

	companyID, err := h.companyIDBySlug(r.Context(), companySlug)
	if err != nil {
		return "", "", err
	}

	var openingID string
	err = h.pool.QueryRow(r.Context(), `
		SELECT id FROM job_openings WHERE company_id = $1 AND slug = $2`, companyID, jobSlug,
	).Scan(&openingID)
	if err != nil {
		return "", "", err
	}

	err = h.pool.QueryRow(r.Context(), `
		SELECT id, uuid FROM candidates
		WHERE job_opening_id = $1 AND uuid = $2 AND application_completed = false`,
		openingID, candidateUUIDParam,
	).Scan(&candidateID, &candidateUUID)
	return candidateID, candidateUUID, err
}
