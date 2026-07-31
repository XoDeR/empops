package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

type nameRequest struct {
	Name string `json:"name"`
}

type updateStageRequest struct {
	Name     *string `json:"name"`
	Position *int    `json:"position"`
}

func (h *Handler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, company_id, name FROM recruiting_stage_templates
		WHERE company_id = $1 ORDER BY name`, member.CompanyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list templates failed", err.Error())
		return
	}
	defer rows.Close()

	out := []map[string]interface{}{}
	for rows.Next() {
		var id, companyID, name string
		if err := rows.Scan(&id, &companyID, &name); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		payload, err := h.templatePayload(r.Context(), id, companyID, name)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "template payload failed", err.Error())
			return
		}
		out = append(out, payload)
	}
	response.OK(w, "", out)
}

func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireManage(w, r); !ok {
		return
	}
	member, _ := companyauth.MemberFromContext(r.Context())

	var req nameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "name is required", nil)
		return
	}

	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO recruiting_stage_templates (id, company_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, now(), now())`, id, member.CompanyID, req.Name)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create template failed", err.Error())
		return
	}

	payload, err := h.templatePayload(r.Context(), id, member.CompanyID, req.Name)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "template payload failed", err.Error())
		return
	}
	response.Created(w, "Template created", payload)
}

func (h *Handler) ShowTemplate(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	templateID := chi.URLParam(r, "templateId")

	var id, cid, name string
	err := h.pool.QueryRow(r.Context(), `
		SELECT id, company_id, name FROM recruiting_stage_templates
		WHERE id = $1 AND company_id = $2`, templateID, companyID,
	).Scan(&id, &cid, &name)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Template not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "template lookup failed", err.Error())
		return
	}

	payload, err := h.templatePayload(r.Context(), id, cid, name)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "template payload failed", err.Error())
		return
	}
	response.OK(w, "", payload)
}

func (h *Handler) UpdateTemplate(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireManage(w, r); !ok {
		return
	}
	companyID := chi.URLParam(r, "companyId")
	templateID := chi.URLParam(r, "templateId")

	var req nameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "name is required", nil)
		return
	}

	tag, err := h.pool.Exec(r.Context(), `
		UPDATE recruiting_stage_templates SET name = $1, updated_at = now()
		WHERE id = $2 AND company_id = $3`, req.Name, templateID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update template failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Template not found", nil)
		return
	}

	payload, err := h.templatePayload(r.Context(), templateID, companyID, req.Name)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "template payload failed", err.Error())
		return
	}
	response.OK(w, "Template updated", payload)
}

func (h *Handler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireManage(w, r); !ok {
		return
	}
	companyID := chi.URLParam(r, "companyId")
	templateID := chi.URLParam(r, "templateId")

	tag, err := h.pool.Exec(r.Context(), `
		DELETE FROM recruiting_stage_templates WHERE id = $1 AND company_id = $2`, templateID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete template failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Template not found", nil)
		return
	}
	response.OK(w, "Template deleted", nil)
}

func (h *Handler) CreateStage(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireManage(w, r); !ok {
		return
	}
	companyID := chi.URLParam(r, "companyId")
	templateID := chi.URLParam(r, "templateId")

	if !h.templateBelongsToCompany(r.Context(), companyID, templateID) {
		response.Fail(w, http.StatusNotFound, "Template not found", nil)
		return
	}

	var req nameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "name is required", nil)
		return
	}

	var maxPos int
	_ = h.pool.QueryRow(r.Context(), `
		SELECT COALESCE(MAX(position), -1) FROM recruiting_stages
		WHERE recruiting_stage_template_id = $1`, templateID,
	).Scan(&maxPos)

	stageID := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO recruiting_stages (id, recruiting_stage_template_id, name, position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, now(), now())`, stageID, templateID, req.Name, maxPos+1)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create stage failed", err.Error())
		return
	}

	response.Created(w, "Stage created", stagePayload(stageID, req.Name, maxPos+1))
}

func (h *Handler) UpdateStage(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireManage(w, r); !ok {
		return
	}
	companyID := chi.URLParam(r, "companyId")
	templateID := chi.URLParam(r, "templateId")
	stageID := chi.URLParam(r, "stageId")

	if !h.templateBelongsToCompany(r.Context(), companyID, templateID) {
		response.Fail(w, http.StatusNotFound, "Template not found", nil)
		return
	}

	var req updateStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	var name string
	var position int
	err := h.pool.QueryRow(r.Context(), `
		SELECT name, position FROM recruiting_stages
		WHERE id = $1 AND recruiting_stage_template_id = $2`, stageID, templateID,
	).Scan(&name, &position)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Stage not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "stage lookup failed", err.Error())
		return
	}

	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}

	if req.Position != nil && *req.Position != position {
		newPos := *req.Position
		if newPos < 0 {
			newPos = 0
		}
		rows, err := h.pool.Query(r.Context(), `
			SELECT id FROM recruiting_stages
			WHERE recruiting_stage_template_id = $1 AND id != $2
			ORDER BY position`, templateID, stageID)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "list stages failed", err.Error())
			return
		}
		var siblingIDs []string
		for rows.Next() {
			var sid string
			if err := rows.Scan(&sid); err != nil {
				rows.Close()
				response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
				return
			}
			siblingIDs = append(siblingIDs, sid)
		}
		rows.Close()

		ordered := make([]string, 0, len(siblingIDs)+1)
		insertAt := newPos
		if insertAt > len(siblingIDs) {
			insertAt = len(siblingIDs)
		}
		ordered = append(ordered, siblingIDs[:insertAt]...)
		ordered = append(ordered, stageID)
		ordered = append(ordered, siblingIDs[insertAt:]...)

		tx, err := h.pool.Begin(r.Context())
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
			return
		}
		defer tx.Rollback(r.Context())
		for i, sid := range ordered {
			if _, err := tx.Exec(r.Context(), `
				UPDATE recruiting_stages SET position = $1, updated_at = now() WHERE id = $2`,
				i, sid); err != nil {
				response.Fail(w, http.StatusInternalServerError, "reorder failed", err.Error())
				return
			}
		}
		position = insertAt
		if err := tx.Commit(r.Context()); err != nil {
			response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
			return
		}
	}

	_, err = h.pool.Exec(r.Context(), `
		UPDATE recruiting_stages SET name = $1, updated_at = now()
		WHERE id = $2 AND recruiting_stage_template_id = $3`, name, stageID, templateID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update stage failed", err.Error())
		return
	}

	response.OK(w, "Stage updated", stagePayload(stageID, name, position))
}

func (h *Handler) DeleteStage(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireManage(w, r); !ok {
		return
	}
	companyID := chi.URLParam(r, "companyId")
	templateID := chi.URLParam(r, "templateId")
	stageID := chi.URLParam(r, "stageId")

	if !h.templateBelongsToCompany(r.Context(), companyID, templateID) {
		response.Fail(w, http.StatusNotFound, "Template not found", nil)
		return
	}

	tag, err := h.pool.Exec(r.Context(), `
		DELETE FROM recruiting_stages WHERE id = $1 AND recruiting_stage_template_id = $2`, stageID, templateID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete stage failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Stage not found", nil)
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT id FROM recruiting_stages
		WHERE recruiting_stage_template_id = $1 ORDER BY position`, templateID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list stages failed", err.Error())
		return
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		var sid string
		if err := rows.Scan(&sid); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		_, _ = h.pool.Exec(r.Context(), `
			UPDATE recruiting_stages SET position = $1, updated_at = now() WHERE id = $2`, i, sid)
		i++
	}

	response.OK(w, "Stage deleted", nil)
}

func (h *Handler) templatePayload(ctx context.Context, id, companyID, name string) (map[string]interface{}, error) {
	stages, err := h.listTemplateStages(ctx, id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":         id,
		"company_id": companyID,
		"name":       name,
		"stages":     stages,
	}, nil
}

func (h *Handler) listTemplateStages(ctx context.Context, templateID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT id, name, position FROM recruiting_stages
		WHERE recruiting_stage_template_id = $1 ORDER BY position`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, name string
		var position int
		if err := rows.Scan(&id, &name, &position); err != nil {
			return nil, err
		}
		out = append(out, stagePayload(id, name, position))
	}
	return out, nil
}

func stagePayload(id, name string, position int) map[string]interface{} {
	return map[string]interface{}{
		"id":       id,
		"name":     name,
		"position": position,
	}
}
