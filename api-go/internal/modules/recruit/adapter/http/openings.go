package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

type jobOpeningInput struct {
	Title                     string   `json:"title"`
	Description               string   `json:"description"`
	PositionID                string   `json:"position_id"`
	RecruitingStageTemplateID string   `json:"recruiting_stage_template_id"`
	TeamID                    *string  `json:"team_id"`
	ReferenceNumber           *string  `json:"reference_number"`
	SponsorIDs []string `json:"sponsor_ids"`
}

type jobOpeningPatch struct {
	Title                     *string  `json:"title"`
	Description               *string  `json:"description"`
	PositionID                *string  `json:"position_id"`
	RecruitingStageTemplateID *string  `json:"recruiting_stage_template_id"`
	TeamID                    *string  `json:"team_id"`
	ReferenceNumber           *string  `json:"reference_number"`
	SponsorIDs *[]string `json:"sponsor_ids"`
}

func (h *Handler) ListOpenings(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}

	var fulfilledFilter *bool
	if v := r.URL.Query().Get("fulfilled"); v != "" {
		b := v == "true" || v == "1"
		fulfilledFilter = &b
	}

	q := `
		SELECT id FROM job_openings WHERE company_id = $1`
	args := []interface{}{member.CompanyID}
	argN := 2

	if fulfilledFilter != nil {
		q += ` AND fulfilled = $` + itoa(argN)
		args = append(args, *fulfilledFilter)
		argN++
	}

	if !hasAnyRecruitingPerm(member) {
		q += ` AND EXISTS(
			SELECT 1 FROM job_opening_sponsor jos
			WHERE jos.job_opening_id = job_openings.id AND jos.employee_id = $` + itoa(argN) + `)`
		args = append(args, member.EmployeeID)
	}

	q += ` ORDER BY created_at DESC`

	rows, err := h.pool.Query(r.Context(), q, args...)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list openings failed", err.Error())
		return
	}
	defer rows.Close()

	out := []map[string]interface{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		payload, err := h.openingPayload(r.Context(), id)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "opening payload failed", err.Error())
			return
		}
		out = append(out, payload)
	}
	response.OK(w, "", out)
}

func (h *Handler) CreateOpening(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireManage(w, r); !ok {
		return
	}
	member, _ := companyauth.MemberFromContext(r.Context())

	var req jobOpeningInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.PositionID = strings.TrimSpace(req.PositionID)
	req.RecruitingStageTemplateID = strings.TrimSpace(req.RecruitingStageTemplateID)
	if req.Title == "" || req.Description == "" || req.PositionID == "" || req.RecruitingStageTemplateID == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "title, description, position_id and recruiting_stage_template_id are required", nil)
		return
	}

	if !h.templateBelongsToCompany(r.Context(), member.CompanyID, req.RecruitingStageTemplateID) {
		response.Fail(w, http.StatusBadRequest, "Invalid stage template", nil)
		return
	}

	slug, err := h.uniqueJobSlug(r.Context(), member.CompanyID, req.Title, "")
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "slug generation failed", err.Error())
		return
	}

	openingID := uuidv7.New()
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(), `
		INSERT INTO job_openings (
			id, company_id, position_id, recruiting_stage_template_id, team_id,
			title, description, slug, reference_number, active, fulfilled, page_views,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false, false, 0, now(), now())`,
		openingID, member.CompanyID, req.PositionID, req.RecruitingStageTemplateID, req.TeamID,
		req.Title, req.Description, slug, req.ReferenceNumber)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create opening failed", err.Error())
		return
	}

	for _, sponsorID := range req.SponsorIDs {
		if sponsorID == "" {
			continue
		}
		_, _ = tx.Exec(r.Context(), `
			INSERT INTO job_opening_sponsor (job_opening_id, employee_id, created_at, updated_at)
			VALUES ($1, $2, now(), now()) ON CONFLICT DO NOTHING`, openingID, sponsorID)
	}

	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}

	payload, err := h.openingPayload(r.Context(), openingID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "opening payload failed", err.Error())
		return
	}
	response.Created(w, "Job opening created", payload)
}

func (h *Handler) ShowOpening(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")

	if !h.openingBelongsToCompany(r.Context(), companyID, openingID) {
		response.Fail(w, http.StatusNotFound, "Job opening not found", nil)
		return
	}
	if _, ok := h.requireAccess(w, r, openingID); !ok {
		return
	}

	payload, err := h.openingPayload(r.Context(), openingID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "opening payload failed", err.Error())
		return
	}
	response.OK(w, "", payload)
}

func (h *Handler) UpdateOpening(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireManage(w, r); !ok {
		return
	}
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")

	var title, description, slug string
	var positionID, templateID string
	var teamID, refNum *string
	err := h.pool.QueryRow(r.Context(), `
		SELECT title, description, slug, position_id, recruiting_stage_template_id, team_id, reference_number
		FROM job_openings WHERE id = $1 AND company_id = $2`,
		openingID, companyID,
	).Scan(&title, &description, &slug, &positionID, &templateID, &teamID, &refNum)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Job opening not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "opening lookup failed", err.Error())
		return
	}

	var req jobOpeningPatch
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	oldTitle := title
	if req.Title != nil {
		title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		description = strings.TrimSpace(*req.Description)
	}
	if req.PositionID != nil {
		positionID = strings.TrimSpace(*req.PositionID)
	}
	if req.RecruitingStageTemplateID != nil {
		templateID = strings.TrimSpace(*req.RecruitingStageTemplateID)
	}
	if req.TeamID != nil {
		teamID = req.TeamID
	}
	if req.ReferenceNumber != nil {
		refNum = req.ReferenceNumber
	}

	if title != oldTitle {
		slug, err = h.uniqueJobSlug(r.Context(), companyID, title, openingID)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "slug generation failed", err.Error())
			return
		}
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(), `
		UPDATE job_openings SET
			title = $1, description = $2, slug = $3, position_id = $4,
			recruiting_stage_template_id = $5, team_id = $6, reference_number = $7, updated_at = now()
		WHERE id = $8 AND company_id = $9`,
		title, description, slug, positionID, templateID, teamID, refNum, openingID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update opening failed", err.Error())
		return
	}

	if req.SponsorIDs != nil {
		_, _ = tx.Exec(r.Context(), `DELETE FROM job_opening_sponsor WHERE job_opening_id = $1`, openingID)
		for _, sponsorID := range *req.SponsorIDs {
			if sponsorID == "" {
				continue
			}
			_, _ = tx.Exec(r.Context(), `
				INSERT INTO job_opening_sponsor (job_opening_id, employee_id, created_at, updated_at)
				VALUES ($1, $2, now(), now()) ON CONFLICT DO NOTHING`, openingID, sponsorID)
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}

	payload, err := h.openingPayload(r.Context(), openingID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "opening payload failed", err.Error())
		return
	}
	response.OK(w, "Job opening updated", payload)
}

func (h *Handler) DeleteOpening(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireManage(w, r); !ok {
		return
	}
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")

	tag, err := h.pool.Exec(r.Context(), `
		DELETE FROM job_openings WHERE id = $1 AND company_id = $2`, openingID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete opening failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Job opening not found", nil)
		return
	}
	response.OK(w, "Job opening deleted", nil)
}

func (h *Handler) ToggleOpening(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireManage(w, r); !ok {
		return
	}
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")

	var active bool
	var activatedAt *time.Time
	err := h.pool.QueryRow(r.Context(), `
		SELECT active, activated_at FROM job_openings WHERE id = $1 AND company_id = $2`,
		openingID, companyID,
	).Scan(&active, &activatedAt)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Job opening not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "opening lookup failed", err.Error())
		return
	}

	active = !active
	if active && activatedAt == nil {
		now := time.Now()
		activatedAt = &now
	}

	_, err = h.pool.Exec(r.Context(), `
		UPDATE job_openings SET active = $1, activated_at = $2, updated_at = now()
		WHERE id = $3 AND company_id = $4`, active, activatedAt, openingID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "toggle opening failed", err.Error())
		return
	}

	payload, err := h.openingPayload(r.Context(), openingID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "opening payload failed", err.Error())
		return
	}
	response.OK(w, "Job opening toggled", payload)
}

func (h *Handler) AddSponsor(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireManage(w, r); !ok {
		return
	}
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")
	employeeID := chi.URLParam(r, "employeeId")

	if !h.openingBelongsToCompany(r.Context(), companyID, openingID) {
		response.Fail(w, http.StatusNotFound, "Job opening not found", nil)
		return
	}

	var empExists bool
	_ = h.pool.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM employees WHERE id = $1 AND company_id = $2)`,
		employeeID, companyID,
	).Scan(&empExists)
	if !empExists {
		response.Fail(w, http.StatusNotFound, "Employee not found", nil)
		return
	}

	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO job_opening_sponsor (job_opening_id, employee_id, created_at, updated_at)
		VALUES ($1, $2, now(), now()) ON CONFLICT DO NOTHING`, openingID, employeeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "add sponsor failed", err.Error())
		return
	}

	payload, err := h.openingPayload(r.Context(), openingID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "opening payload failed", err.Error())
		return
	}
	response.OK(w, "Sponsor added", payload)
}

func (h *Handler) RemoveSponsor(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireManage(w, r); !ok {
		return
	}
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")
	employeeID := chi.URLParam(r, "employeeId")

	if !h.openingBelongsToCompany(r.Context(), companyID, openingID) {
		response.Fail(w, http.StatusNotFound, "Job opening not found", nil)
		return
	}

	_, err := h.pool.Exec(r.Context(), `
		DELETE FROM job_opening_sponsor WHERE job_opening_id = $1 AND employee_id = $2`,
		openingID, employeeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "remove sponsor failed", err.Error())
		return
	}

	payload, err := h.openingPayload(r.Context(), openingID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "opening payload failed", err.Error())
		return
	}
	response.OK(w, "Sponsor removed", payload)
}

func (h *Handler) openingPayload(ctx context.Context, openingID string) (map[string]interface{}, error) {
	var (
		id, companyID, title, description, slug, positionID string
		templateID                                          *string
		teamID, refNum                                      *string
		active, fulfilled                                   bool
		pageViews                                           int
		activatedAt, fulfilledAt                            *time.Time
		posTitle                                            *string
	)

	err := h.pool.QueryRow(ctx, `
		SELECT jo.id, jo.company_id, jo.title, jo.description, jo.slug, jo.position_id,
			jo.recruiting_stage_template_id, jo.team_id, jo.reference_number,
			jo.active, jo.fulfilled, jo.page_views, jo.activated_at, jo.fulfilled_at,
			p.title
		FROM job_openings jo
		LEFT JOIN positions p ON p.id = jo.position_id
		WHERE jo.id = $1`, openingID,
	).Scan(&id, &companyID, &title, &description, &slug, &positionID,
		&templateID, &teamID, &refNum, &active, &fulfilled, &pageViews,
		&activatedAt, &fulfilledAt, &posTitle)
	if err != nil {
		return nil, err
	}

	sponsors, err := h.listSponsors(ctx, openingID)
	if err != nil {
		return nil, err
	}

	var position interface{}
	if posTitle != nil {
		position = map[string]interface{}{"id": positionID, "title": *posTitle}
	}

	payload := map[string]interface{}{
		"id":                           id,
		"company_id":                   companyID,
		"title":                        title,
		"description":                  description,
		"slug":                         slug,
		"reference_number":             refNum,
		"position_id":                  positionID,
		"position":                     position,
		"recruiting_stage_template_id": templateID,
		"team_id":                      teamID,
		"active":                       active,
		"fulfilled":                    fulfilled,
		"page_views":                   pageViews,
		"activated_at":                 formatTimePtr(activatedAt),
		"fulfilled_at":                 formatTimePtr(fulfilledAt),
		"sponsors":                     sponsors,
	}
	return payload, nil
}

func (h *Handler) listSponsors(ctx context.Context, openingID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT e.id, e.first_name, e.last_name
		FROM job_opening_sponsor jos
		JOIN employees e ON e.id = jos.employee_id
		WHERE jos.job_opening_id = $1
		ORDER BY e.last_name, e.first_name`, openingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, first, last string
		if err := rows.Scan(&id, &first, &last); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"id":         id,
			"first_name": first,
			"last_name":  last,
		})
	}
	return out, nil
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
