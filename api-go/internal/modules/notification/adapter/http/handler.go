// Package http contains the notification module's Chi handlers.
package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
)

// Handler serves notification HTTP endpoints.
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler constructs a notification Handler backed by pool.
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

type markReadRequest struct {
	IDs []string `json:"ids"`
}

// ListNotifications handles GET /companies/{companyId}/notifications for the
// current employee, newest first, along with their unread count.
func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT id, company_id, employee_id, action, objects, read_at, created_at, updated_at
		FROM notifications
		WHERE company_id = $1 AND employee_id = $2
		ORDER BY created_at DESC
		LIMIT 100`, member.CompanyID, member.EmployeeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list notifications failed", err.Error())
		return
	}
	defer rows.Close()

	items := []map[string]interface{}{}
	unread := 0
	for rows.Next() {
		var id, cid, employeeID, action string
		var objectsRaw []byte
		var readAt *time.Time
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &cid, &employeeID, &action, &objectsRaw, &readAt, &createdAt, &updatedAt); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		var objects map[string]interface{}
		_ = json.Unmarshal(objectsRaw, &objects)
		if objects == nil {
			objects = map[string]interface{}{}
		}
		if readAt == nil {
			unread++
		}
		items = append(items, map[string]interface{}{
			"id":          id,
			"company_id":  cid,
			"employee_id": employeeID,
			"action":      action,
			"objects":     objects,
			"read":        readAt != nil,
			"read_at":     formatTimePtr(readAt),
			"created_at":  createdAt.UTC().Format(time.RFC3339),
			"updated_at":  updatedAt.UTC().Format(time.RFC3339),
		})
	}

	response.OK(w, "", map[string]interface{}{
		"items":        items,
		"unread_count": unread,
	})
}

// MarkRead handles POST /companies/{companyId}/notifications/read. An empty
// or missing ids list marks every unread notification as read for the actor.
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}

	var req markReadRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	var err error
	if len(req.IDs) == 0 {
		_, err = h.pool.Exec(r.Context(), `
			UPDATE notifications SET read_at = now(), updated_at = now()
			WHERE company_id = $1 AND employee_id = $2 AND read_at IS NULL`,
			member.CompanyID, member.EmployeeID,
		)
	} else {
		_, err = h.pool.Exec(r.Context(), `
			UPDATE notifications SET read_at = now(), updated_at = now()
			WHERE company_id = $1 AND employee_id = $2 AND id = ANY($3::uuid[]) AND read_at IS NULL`,
			member.CompanyID, member.EmployeeID, req.IDs,
		)
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "mark read failed", err.Error())
		return
	}

	response.OK(w, "Notifications marked as read", nil)
}

func formatTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
