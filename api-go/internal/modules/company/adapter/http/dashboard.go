package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
)

// DashboardMe handles GET /companies/{companyId}/dashboard/me.
func (h *Handler) DashboardMe(w http.ResponseWriter, r *http.Request) {
	h.dashboardShell(w, r, "me", true)
}

// DashboardTeam handles GET /companies/{companyId}/dashboard/team.
func (h *Handler) DashboardTeam(w http.ResponseWriter, r *http.Request) {
	h.dashboardShell(w, r, "team", true)
}

// DashboardManager handles GET /companies/{companyId}/dashboard/manager.
func (h *Handler) DashboardManager(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	if !h.isManager(r, member) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	h.dashboardShell(w, r, "manager", true)
}

// DashboardHR handles GET /companies/{companyId}/dashboard/hr.
func (h *Handler) DashboardHR(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	if !hasAnyRole(member.Roles, "administrator", "hr") {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	h.dashboardShell(w, r, "hr", true)
}

func (h *Handler) dashboardShell(w http.ResponseWriter, r *http.Request, view string, allowed bool) {
	if !allowed {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	member, _ := companyauth.MemberFromContext(r.Context())

	widgets := []interface{}{}
	if view == "me" {
		widgets = h.meDashboardWidgets(r, member)
	}

	response.OK(w, "", map[string]interface{}{
		"view":    view,
		"widgets": widgets,
		"flags": map[string]interface{}{
			"is_manager":    h.isManager(r, member),
			"can_manage_hr": hasAnyRole(member.Roles, "administrator", "hr"),
			"is_admin":      hasAnyRole(member.Roles, "administrator"),
		},
	})
}

// meDashboardWidgets builds the widgets shown on the "me" dashboard: whether
// the actor logged today's worklog, the active company question (and whether
// they've answered it), and their unread notification count.
func (h *Handler) meDashboardWidgets(r *http.Request, member companyauth.Member) []interface{} {
	today := time.Now().UTC().Format("2006-01-02")

	var worklogID, content string
	var loggedOn time.Time
	var worklog interface{}
	logged := false
	if err := h.pool.QueryRow(r.Context(), `
		SELECT id, content, logged_on FROM worklogs WHERE employee_id = $1 AND logged_on = $2`,
		member.EmployeeID, today,
	).Scan(&worklogID, &content, &loggedOn); err == nil {
		logged = true
		worklog = map[string]interface{}{
			"id":        worklogID,
			"content":   content,
			"logged_on": loggedOn.Format("2006-01-02"),
		}
	}

	var consecutiveMissed int
	_ = h.pool.QueryRow(r.Context(), `
		SELECT consecutive_worklog_missed FROM employees WHERE id = $1`, member.EmployeeID,
	).Scan(&consecutiveMissed)

	var activeQuestion interface{}
	var questionID, questionTitle string
	if err := h.pool.QueryRow(r.Context(), `
		SELECT id, title FROM questions WHERE company_id = $1 AND active = true LIMIT 1`,
		member.CompanyID,
	).Scan(&questionID, &questionTitle); err == nil {
		var answerID string
		answered := h.pool.QueryRow(r.Context(), `
			SELECT id FROM answers WHERE question_id = $1 AND employee_id = $2`,
			questionID, member.EmployeeID,
		).Scan(&answerID) == nil
		activeQuestion = map[string]interface{}{
			"id":       questionID,
			"title":    questionTitle,
			"answered": answered,
		}
	}

	var unreadCount int
	_ = h.pool.QueryRow(r.Context(), `
		SELECT COUNT(*) FROM notifications WHERE employee_id = $1 AND read_at IS NULL`,
		member.EmployeeID,
	).Scan(&unreadCount)

	return []interface{}{
		map[string]interface{}{
			"type": "worklog_today",
			"data": map[string]interface{}{
				"logged":             logged,
				"worklog":            worklog,
				"consecutive_missed": consecutiveMissed,
			},
		},
		map[string]interface{}{
			"type": "active_question",
			"data": activeQuestion,
		},
		map[string]interface{}{
			"type": "unread_notifications",
			"data": map[string]interface{}{
				"count": unreadCount,
			},
		},
	}
}

func (h *Handler) isManager(r *http.Request, member companyauth.Member) bool {
	if hasAnyRole(member.Roles, "manager") {
		return true
	}
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM direct_reports WHERE manager_id = $1 AND company_id = $2)`,
		member.EmployeeID, member.CompanyID,
	).Scan(&exists)
	return exists
}

func hasAnyRole(roles []string, names ...string) bool {
	set := map[string]bool{}
	for _, r := range roles {
		set[r] = true
	}
	for _, n := range names {
		if set[n] {
			return true
		}
	}
	return false
}

// ListAuditLogs handles GET /companies/{companyId}/audit-logs.
func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 {
		perPage = 50
	}
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	var total int
	_ = h.pool.QueryRow(r.Context(), `
		SELECT COUNT(*) FROM activity_logs WHERE company_id = $1`, companyID,
	).Scan(&total)

	rows, err := h.pool.Query(r.Context(), `
		SELECT id, company_id, event, description, subject_type, subject_id,
			causer_type, causer_id, properties, created_at
		FROM activity_logs
		WHERE company_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, companyID, perPage, offset)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list audit logs failed", err.Error())
		return
	}
	defer rows.Close()

	items := []map[string]interface{}{}
	for rows.Next() {
		var (
			id, cid, event, description string
			subjectType, subjectID      *string
			causerType, causerID        *string
			propsRaw                    []byte
			createdAt                   time.Time
		)
		if err := rows.Scan(&id, &cid, &event, &description, &subjectType, &subjectID, &causerType, &causerID, &propsRaw, &createdAt); err != nil {
			response.Fail(w, http.StatusInternalServerError, "list audit logs failed", err.Error())
			return
		}
		var props map[string]interface{}
		_ = json.Unmarshal(propsRaw, &props)
		if props == nil {
			props = map[string]interface{}{}
		}
		items = append(items, map[string]interface{}{
			"id":           id,
			"company_id":   cid,
			"event":        event,
			"description":  description,
			"subject_type": subjectType,
			"subject_id":   subjectID,
			"causer_type":  causerType,
			"causer_id":    causerID,
			"properties":   props,
			"created_at":   createdAt.UTC().Format(time.RFC3339),
		})
	}

	response.OK(w, "", map[string]interface{}{
		"items":    items,
		"page":     page,
		"per_page": perPage,
		"total":    total,
	})
}
