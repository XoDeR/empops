package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

const (
	modelTemporaryUpload = "temporary_upload"
	modelDisciplineEvent = "discipline_event"
	collectionDiscipline = "discipline"
)

type Handler struct{ pool *pgxpool.Pool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

func decodeJSON(r *http.Request, dst interface{}) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (h *Handler) employeeSummary(ctx context.Context, id string) map[string]interface{} {
	var first, last, email string
	_ = h.pool.QueryRow(ctx, `SELECT first_name, last_name, email FROM employees WHERE id=$1`, id).
		Scan(&first, &last, &email)
	return map[string]interface{}{"id": id, "first_name": first, "last_name": last, "email": email}
}

func (h *Handler) isHr(m companyauth.Member) bool {
	for _, r := range m.Roles {
		if r == "administrator" || r == "hr" {
			return true
		}
	}
	return false
}

func (h *Handler) isManagerOf(ctx context.Context, companyID, managerID, employeeID string) bool {
	var ok bool
	_ = h.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM direct_reports
			WHERE company_id=$1 AND manager_id=$2 AND employee_id=$3)`,
		companyID, managerID, employeeID).Scan(&ok)
	return ok
}

func isoTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func isoDate(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}

// --- Morale ---

func (h *Handler) TodayMorale(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	today := time.Now().UTC().Format("2006-01-02")
	var id, employeeID string
	var emotion int
	var comment *string
	var createdAt time.Time
	err := h.pool.QueryRow(r.Context(), `
		SELECT id, employee_id, emotion, comment, created_at FROM morales
		WHERE employee_id=$1 AND created_at::date=$2::date LIMIT 1`,
		member.EmployeeID, today,
	).Scan(&id, &employeeID, &emotion, &comment, &createdAt)
	if err == pgx.ErrNoRows {
		response.OK(w, "", nil)
		return
	}
	if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return
	}
	response.OK(w, "", map[string]interface{}{
		"id": id, "employee_id": employeeID, "emotion": emotion, "comment": comment,
		"created_at": createdAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) LogMorale(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	var body struct {
		Emotion int     `json:"emotion"`
		Comment *string `json:"comment"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.Fail(w, 422, "invalid body", err.Error())
		return
	}
	if body.Emotion < 1 || body.Emotion > 3 {
		response.Fail(w, 422, "Emotion must be 1, 2, or 3", nil)
		return
	}
	today := time.Now().UTC().Format("2006-01-02")
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM morales WHERE employee_id=$1 AND created_at::date=$2::date)`,
		member.EmployeeID, today).Scan(&exists)
	if exists {
		response.Fail(w, 409, "Morale already logged today", nil)
		return
	}
	id := uuidv7.New()
	var createdAt time.Time
	err := h.pool.QueryRow(r.Context(), `
		INSERT INTO morales (id, company_id, employee_id, emotion, comment)
		VALUES ($1,$2,$3,$4,$5) RETURNING created_at`,
		id, member.CompanyID, member.EmployeeID, body.Emotion, body.Comment,
	).Scan(&createdAt)
	if err != nil {
		response.Fail(w, 500, "create failed", err.Error())
		return
	}
	response.OK(w, "", map[string]interface{}{
		"id": id, "employee_id": member.EmployeeID, "emotion": body.Emotion, "comment": body.Comment,
		"created_at": createdAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) CompanyMoraleHistory(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, average, number_of_employees, created_at FROM morale_company_histories
		WHERE company_id=$1 ORDER BY created_at DESC LIMIT 30`, member.CompanyID)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id string
		var avg float64
		var n int
		var createdAt time.Time
		if rows.Scan(&id, &avg, &n, &createdAt) == nil {
			out = append(out, map[string]interface{}{
				"id": id, "average": avg, "number_of_employees": n,
				"created_at": createdAt.UTC().Format(time.RFC3339),
			})
		}
	}
	response.OK(w, "", out)
}

func (h *Handler) TeamMoraleHistory(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	teamID := chi.URLParam(r, "teamId")
	var ok bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM teams WHERE id=$1 AND company_id=$2)`, teamID, member.CompanyID).Scan(&ok)
	if !ok {
		response.Fail(w, 404, "Team not found", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, average, number_of_team_members, created_at FROM morale_team_histories
		WHERE team_id=$1 ORDER BY created_at DESC LIMIT 30`, teamID)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id string
		var avg float64
		var n int
		var createdAt time.Time
		if rows.Scan(&id, &avg, &n, &createdAt) == nil {
			out = append(out, map[string]interface{}{
				"id": id, "average": avg, "number_of_team_members": n,
				"created_at": createdAt.UTC().Format(time.RFC3339),
			})
		}
	}
	response.OK(w, "", out)
}

func (h *Handler) EmployeeMorale(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	employeeID := chi.URLParam(r, "employeeId")
	if employeeID != member.EmployeeID && !h.isHr(member) && !h.isManagerOf(r.Context(), member.CompanyID, member.EmployeeID, employeeID) {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, employee_id, emotion, comment, created_at FROM morales
		WHERE company_id=$1 AND employee_id=$2 ORDER BY created_at DESC LIMIT 30`,
		member.CompanyID, employeeID)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, eid string
		var emotion int
		var comment *string
		var createdAt time.Time
		if rows.Scan(&id, &eid, &emotion, &comment, &createdAt) == nil {
			out = append(out, map[string]interface{}{
				"id": id, "employee_id": eid, "emotion": emotion, "comment": comment,
				"created_at": createdAt.UTC().Format(time.RFC3339),
			})
		}
	}
	response.OK(w, "", out)
}

func normalizeSkill(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func (h *Handler) filePayload(id int64, fileName, mimeType string, size int64) map[string]interface{} {
	return map[string]interface{}{
		"id": id, "file_name": fileName, "mime_type": mimeType, "size": size,
		"url": fmt.Sprintf("/api/v1/media/%d/file", id),
	}
}
