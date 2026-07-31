package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
)

const (
	modelTemporaryUpload = "temporary_upload"
	modelCandidate       = "candidate"
	collectionCV         = "cv"
)

var recruitingPerms = []string{
	"recruiting.view",
	"recruiting.create",
	"recruiting.update",
	"recruiting.delete",
	"recruiting.hire",
	"recruiting.manage_templates",
}

type Handler struct{ pool *pgxpool.Pool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

func hasRole(member companyauth.Member, names ...string) bool {
	for _, r := range member.Roles {
		for _, n := range names {
			if r == n {
				return true
			}
		}
	}
	return false
}

func hasRecruitingPerm(member companyauth.Member, perm string) bool {
	return member.HasPermission(perm)
}

func hasAnyRecruitingPerm(member companyauth.Member) bool {
	for _, p := range recruitingPerms {
		if member.HasPermission(p) {
			return true
		}
	}
	if hasRole(member, "administrator", "hr") {
		return true
	}
	return false
}

func canManage(member companyauth.Member) bool {
	return member.HasPermission("recruiting.update") || hasRole(member, "administrator", "hr")
}

func (h *Handler) isSponsor(ctx context.Context, openingID, employeeID string) bool {
	var exists bool
	_ = h.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM job_opening_sponsor
			WHERE job_opening_id = $1 AND employee_id = $2
		)`, openingID, employeeID,
	).Scan(&exists)
	return exists
}

func (h *Handler) canAccessOpening(ctx context.Context, member companyauth.Member, openingID string) bool {
	if hasAnyRecruitingPerm(member) {
		return true
	}
	return h.isSponsor(ctx, openingID, member.EmployeeID)
}

func (h *Handler) requireAccess(w http.ResponseWriter, r *http.Request, openingID string) (companyauth.Member, bool) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return member, false
	}
	if !h.canAccessOpening(r.Context(), member, openingID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return member, false
	}
	return member, true
}

func (h *Handler) requireManage(w http.ResponseWriter, r *http.Request) (companyauth.Member, bool) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return member, false
	}
	if !canManage(member) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return member, false
	}
	return member, true
}

func (h *Handler) openingBelongsToCompany(ctx context.Context, companyID, openingID string) bool {
	var exists bool
	_ = h.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM job_openings WHERE id = $1 AND company_id = $2)`,
		openingID, companyID,
	).Scan(&exists)
	return exists
}

func (h *Handler) findOpeningIDForCandidate(ctx context.Context, companyID, openingID, candidateID string) bool {
	var exists bool
	_ = h.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM candidates c
			JOIN job_openings jo ON jo.id = c.job_opening_id
			WHERE c.id = $1 AND c.job_opening_id = $2 AND jo.company_id = $3
		)`, candidateID, openingID, companyID,
	).Scan(&exists)
	return exists
}

func (h *Handler) templateBelongsToCompany(ctx context.Context, companyID, templateID string) bool {
	var exists bool
	_ = h.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM recruiting_stage_templates WHERE id = $1 AND company_id = $2)`,
		templateID, companyID,
	).Scan(&exists)
	return exists
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "job"
	}
	return s
}

func randomSuffix(n int) string {
	b := make([]byte, (n+1)/2)
	_, _ = rand.Read(b)
	return strings.ToLower(hex.EncodeToString(b))[:n]
}

func (h *Handler) uniqueJobSlug(ctx context.Context, companyID, title, exceptID string) (string, error) {
	base := slugify(title)
	slug := base + "-" + randomSuffix(8)
	for i := 0; i < 100; i++ {
		var exists bool
		q := `SELECT EXISTS(SELECT 1 FROM job_openings WHERE company_id = $1 AND slug = $2`
		args := []interface{}{companyID, slug}
		if exceptID != "" {
			q += ` AND id != $3`
			args = append(args, exceptID)
		}
		q += `)`
		if err := h.pool.QueryRow(ctx, q, args...).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		suffix := ""
		if i > 0 {
			suffix = fmt.Sprintf("-%d", i)
		}
		slug = base + "-" + randomSuffix(8) + suffix
	}
	return slug, fmt.Errorf("could not generate unique slug")
}

func formatTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func filePayload(id int64, fileName string, mimeType *string, size int64) map[string]interface{} {
	return map[string]interface{}{
		"id":        id,
		"file_name": fileName,
		"mime_type": mimeType,
		"size":      size,
		"url":       fmt.Sprintf("/api/v1/media/%d/file", id),
	}
}

func (h *Handler) listCandidateFiles(ctx context.Context, candidateID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT id, file_name, mime_type, size FROM media
		WHERE model_type = $1 AND model_id = $2 AND collection_name = $3
		ORDER BY id`, modelCandidate, candidateID, collectionCV)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var fileName string
		var mimeType *string
		var size int64
		if err := rows.Scan(&id, &fileName, &mimeType, &size); err != nil {
			return nil, err
		}
		out = append(out, filePayload(id, fileName, mimeType, size))
	}
	return out, nil
}

func (h *Handler) attachCandidateFile(ctx context.Context, candidateID string, tempUploadID, mediaID int64) (map[string]interface{}, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var fileName string
	err = tx.QueryRow(ctx, `
		SELECT file_name FROM media
		WHERE id = $1 AND model_type = $2 AND model_id = $3`,
		mediaID, modelTemporaryUpload, fmt.Sprintf("%d", tempUploadID),
	).Scan(&fileName)
	if err == pgx.ErrNoRows {
		return nil, pgx.ErrNoRows
	}
	if err != nil {
		return nil, err
	}

	tag, err := tx.Exec(ctx, `
		UPDATE media SET model_type = $2, model_id = $3, collection_name = $4, updated_at = now()
		WHERE id = $1 AND model_type = $5 AND model_id = $6`,
		mediaID, modelCandidate, candidateID, collectionCV,
		modelTemporaryUpload, fmt.Sprintf("%d", tempUploadID))
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	var mimeType *string
	var size int64
	_ = h.pool.QueryRow(ctx, `SELECT mime_type, size FROM media WHERE id = $1`, mediaID).Scan(&mimeType, &size)
	return filePayload(mediaID, fileName, mimeType, size), nil
}

func (h *Handler) deleteCandidateFile(ctx context.Context, candidateID string, mediaID string) error {
	tag, err := h.pool.Exec(ctx, `
		DELETE FROM media
		WHERE id = $1 AND model_type = $2 AND model_id = $3 AND collection_name = $4`,
		mediaID, modelCandidate, candidateID, collectionCV)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func deciderName(ctx context.Context, pool *pgxpool.Pool, employeeID string) string {
	var first, last string
	err := pool.QueryRow(ctx, `SELECT first_name, last_name FROM employees WHERE id = $1`, employeeID).Scan(&first, &last)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(first + " " + last)
}
