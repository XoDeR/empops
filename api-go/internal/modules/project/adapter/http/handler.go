package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

const (
	modelTemporaryUpload = "temporary_upload"
	modelProject         = "project"
	commentTypeMessage   = "project_message"
	commentTypeTask      = "project_task"
)

var defaultIssueTypes = []struct{ name, icon string }{
	{"Bug", "bug"},
	{"Story", "story"},
	{"Task", "task"},
	{"Epic", "epic"},
}

type Handler struct{ pool *pgxpool.Pool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

type projectRow struct {
	ID, CompanyID, Name, Status string
	Code, ShortCode, Emoji      *string
	Summary, Description        *string
	ProjectLeadID               *string
	Completed                   bool
	StartedAt, PlannedFinished  *time.Time
	ActuallyFinished            *time.Time
}

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

func (h *Handler) isMember(ctx context.Context, projectID, employeeID string) bool {
	var exists bool
	_ = h.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM employee_project WHERE project_id=$1 AND employee_id=$2)`, projectID, employeeID).Scan(&exists)
	return exists
}

func (h *Handler) canAccessCtx(ctx context.Context, member companyauth.Member, projectID string) bool {
	if hasRole(member, "administrator", "hr") || member.HasPermission("projects.view") {
		return true
	}
	return h.isMember(ctx, projectID, member.EmployeeID)
}

func (h *Handler) canManageCtx(ctx context.Context, member companyauth.Member, projectID string) bool {
	if hasRole(member, "administrator", "hr") {
		return true
	}
	if member.HasPermission("projects.update") || member.HasPermission("projects.manage_members") {
		return true
	}
	return h.isMember(ctx, projectID, member.EmployeeID)
}

func (h *Handler) findProject(ctx context.Context, companyID, projectID string) (projectRow, error) {
	var p projectRow
	err := h.pool.QueryRow(ctx, `
		SELECT id, company_id, name, status, code, short_code, emoji, summary, description,
			project_lead_id, completed, started_at, planned_finished_at, actually_finished_at
		FROM projects WHERE company_id=$1 AND id=$2`,
		companyID, projectID,
	).Scan(&p.ID, &p.CompanyID, &p.Name, &p.Status, &p.Code, &p.ShortCode, &p.Emoji, &p.Summary, &p.Description,
		&p.ProjectLeadID, &p.Completed, &p.StartedAt, &p.PlannedFinished, &p.ActuallyFinished)
	return p, err
}

func (h *Handler) employeeSummary(ctx context.Context, id string) (map[string]interface{}, error) {
	var first, last, email string
	err := h.pool.QueryRow(ctx, `SELECT first_name, last_name, email FROM employees WHERE id=$1`, id).Scan(&first, &last, &email)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id, "first_name": first, "last_name": last, "email": email}, nil
}

func (h *Handler) employeeFullName(ctx context.Context, id string) (string, error) {
	var first, last string
	err := h.pool.QueryRow(ctx, `SELECT first_name, last_name FROM employees WHERE id=$1`, id).Scan(&first, &last)
	return strings.TrimSpace(first + " " + last), err
}

func (h *Handler) loadMembers(ctx context.Context, projectID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT e.id FROM employees e
		JOIN employee_project ep ON ep.employee_id=e.id
		WHERE ep.project_id=$1 ORDER BY e.last_name, e.first_name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		s, err := h.employeeSummary(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (h *Handler) loadTeams(ctx context.Context, projectID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT t.id, t.name FROM teams t
		JOIN project_team pt ON pt.team_id=t.id
		WHERE pt.project_id=$1 ORDER BY t.name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{"id": id, "name": name})
	}
	return out, rows.Err()
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

func (h *Handler) projectSummaryPayload(p projectRow) map[string]interface{} {
	return map[string]interface{}{
		"id": p.ID, "name": p.Name, "code": p.Code, "short_code": p.ShortCode,
		"status": p.Status, "emoji": p.Emoji,
	}
}

func (h *Handler) projectPayload(ctx context.Context, p projectRow) (map[string]interface{}, error) {
	members, err := h.loadMembers(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	teams, err := h.loadTeams(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	var lead interface{}
	if p.ProjectLeadID != nil {
		lead, err = h.employeeSummary(ctx, *p.ProjectLeadID)
		if err != nil && err != pgx.ErrNoRows {
			return nil, err
		}
	}
	return map[string]interface{}{
		"id": p.ID, "company_id": p.CompanyID, "name": p.Name, "code": p.Code,
		"short_code": p.ShortCode, "emoji": p.Emoji, "summary": p.Summary,
		"description": p.Description, "status": p.Status, "completed": p.Completed,
		"project_lead_id": p.ProjectLeadID, "lead": lead,
		"started_at": isoTime(p.StartedAt), "planned_finished_at": isoTime(p.PlannedFinished),
		"actually_finished_at": isoTime(p.ActuallyFinished),
		"members": members, "teams": teams, "member_count": len(members),
	}, nil
}

func (h *Handler) ensureIssueTypes(ctx context.Context, companyID string) error {
	var exists bool
	if err := h.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM issue_types WHERE company_id=$1)`, companyID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	for _, t := range defaultIssueTypes {
		_, err := h.pool.Exec(ctx, `INSERT INTO issue_types(id,company_id,name,icon) VALUES($1,$2,$3,$4)`,
			uuidv7.New(), companyID, t.name, t.icon)
		if err != nil {
			return err
		}
	}
	return nil
}

func generateShortCode(ctx context.Context, pool *pgxpool.Pool, companyID, name string) (string, error) {
	words := strings.Fields(strings.TrimSpace(name))
	code := ""
	for _, word := range words {
		if word == "" {
			continue
		}
		runes := []rune(word)
		code += strings.ToUpper(string(runes[0]))
		if len(code) >= 3 {
			break
		}
	}
	if len(code) < 3 {
		var b strings.Builder
		for _, r := range name {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				b.WriteRune(unicode.ToUpper(r))
			}
		}
		alpha := b.String()
		if alpha == "" {
			alpha = "PRJ"
		}
		code = code + alpha
		if len(code) < 3 {
			code = code + "PRJ"
		}
		code = code[:3]
	}
	base := code
	suffix := 1
	for {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE company_id=$1 AND short_code=$2)`, companyID, code).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
		if len(base) >= 2 {
			code = base[:2] + strconv.Itoa(suffix)
		} else {
			code = base + strconv.Itoa(suffix)
		}
		suffix++
	}
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func (h *Handler) IssueTypes(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	if err := h.ensureIssueTypes(r.Context(), member.CompanyID); err != nil {
		response.Fail(w, http.StatusInternalServerError, "issue types seed failed", err.Error())
		return
	}
	rows, err := h.pool.Query(r.Context(), `SELECT id, company_id, name, icon FROM issue_types WHERE company_id=$1 ORDER BY name`, member.CompanyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list issue types failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, companyID, name string
		var icon *string
		if err := rows.Scan(&id, &companyID, &name, &icon); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		out = append(out, map[string]interface{}{"id": id, "company_id": companyID, "name": name, "icon": icon})
	}
	response.OK(w, "", out)
}

func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	q := `SELECT id, company_id, name, status, code, short_code, emoji, summary, description,
		project_lead_id, completed, started_at, planned_finished_at, actually_finished_at
		FROM projects WHERE company_id=$1`
	args := []interface{}{member.CompanyID}
	if !hasRole(member, "administrator", "hr") && !member.HasPermission("projects.view") {
		q += ` AND EXISTS(SELECT 1 FROM employee_project ep WHERE ep.project_id=projects.id AND ep.employee_id=$2)`
		args = append(args, member.EmployeeID)
	}
	q += ` ORDER BY name`
	rows, err := h.pool.Query(r.Context(), q, args...)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list projects failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var p projectRow
		if err := rows.Scan(&p.ID, &p.CompanyID, &p.Name, &p.Status, &p.Code, &p.ShortCode, &p.Emoji, &p.Summary, &p.Description,
			&p.ProjectLeadID, &p.Completed, &p.StartedAt, &p.PlannedFinished, &p.ActuallyFinished); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		payload, err := h.projectPayload(r.Context(), p)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "payload failed", err.Error())
			return
		}
		out = append(out, payload)
	}
	response.OK(w, "", out)
}

type projectInput struct {
	Name              string  `json:"name"`
	Code              *string `json:"code"`
	ShortCode         *string `json:"short_code"`
	Emoji             *string `json:"emoji"`
	Summary           *string `json:"summary"`
	Description       *string `json:"description"`
	Status            *string `json:"status"`
	ProjectLeadID     *string `json:"project_lead_id"`
	StartedAt         *string `json:"started_at"`
	PlannedFinishedAt *string `json:"planned_finished_at"`
}

func parseOptionalTime(raw *string) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *raw)
	if err != nil {
		t, err = time.Parse("2006-01-02", *raw)
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	var req projectInput
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.Name) == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "name is required", nil)
		return
	}
	shortCode := ""
	if req.ShortCode != nil && *req.ShortCode != "" {
		shortCode = *req.ShortCode
	} else {
		var err error
		shortCode, err = generateShortCode(r.Context(), h.pool, member.CompanyID, req.Name)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "short code generation failed", err.Error())
			return
		}
	}
	status := "created"
	if req.Status != nil && *req.Status != "" {
		status = *req.Status
	}
	startedAt, _ := parseOptionalTime(req.StartedAt)
	plannedAt, _ := parseOptionalTime(req.PlannedFinishedAt)
	projectID := uuidv7.New()
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())
	_, err = tx.Exec(r.Context(), `
		INSERT INTO projects(id,company_id,project_lead_id,name,code,short_code,emoji,summary,description,status,started_at,planned_finished_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		projectID, member.CompanyID, req.ProjectLeadID, req.Name, req.Code, shortCode, req.Emoji,
		req.Summary, req.Description, status, startedAt, plannedAt)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create project failed", err.Error())
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO employee_project(employee_id,project_id) VALUES($1,$2)`, member.EmployeeID, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "add member failed", err.Error())
		return
	}
	if req.ProjectLeadID != nil && *req.ProjectLeadID != "" {
		_, _ = tx.Exec(r.Context(), `INSERT INTO employee_project(employee_id,project_id,role) VALUES($1,$2,'lead') ON CONFLICT(employee_id,project_id) DO UPDATE SET role='lead'`, *req.ProjectLeadID, projectID)
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}
	p, err := h.findProject(r.Context(), member.CompanyID, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "load project failed", err.Error())
		return
	}
	payload, err := h.projectPayload(r.Context(), p)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "payload failed", err.Error())
		return
	}
	response.Created(w, "Project created", payload)
}

func (h *Handler) ShowProject(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	p, err := h.findProject(r.Context(), member.CompanyID, projectID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Project not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "lookup failed", err.Error())
		return
	}
	if !h.canAccessCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	payload, err := h.projectPayload(r.Context(), p)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "payload failed", err.Error())
		return
	}
	response.OK(w, "", payload)
}

func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	p, err := h.findProject(r.Context(), member.CompanyID, projectID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Project not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "lookup failed", err.Error())
		return
	}
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req projectInput
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	name := p.Name
	if req.Name != "" {
		name = req.Name
	}
	status := p.Status
	if req.Status != nil {
		status = *req.Status
	}
	completed := status == "closed" || status == "cancelled"
	startedAt := p.StartedAt
	if req.StartedAt != nil {
		startedAt, _ = parseOptionalTime(req.StartedAt)
	}
	plannedAt := p.PlannedFinished
	if req.PlannedFinishedAt != nil {
		plannedAt, _ = parseOptionalTime(req.PlannedFinishedAt)
	}
	code := p.Code
	if req.Code != nil {
		code = req.Code
	}
	shortCode := p.ShortCode
	if req.ShortCode != nil {
		shortCode = req.ShortCode
	}
	emoji := p.Emoji
	if req.Emoji != nil {
		emoji = req.Emoji
	}
	summary := p.Summary
	if req.Summary != nil {
		summary = req.Summary
	}
	description := p.Description
	if req.Description != nil {
		description = req.Description
	}
	leadID := p.ProjectLeadID
	if req.ProjectLeadID != nil {
		leadID = req.ProjectLeadID
	}
	_, err = h.pool.Exec(r.Context(), `
		UPDATE projects SET name=$2,code=$3,short_code=$4,emoji=$5,summary=$6,description=$7,
			status=$8,completed=$9,project_lead_id=$10,started_at=$11,planned_finished_at=$12,updated_at=now()
		WHERE id=$1`,
		projectID, name, code, shortCode, emoji, summary, description, status, completed, leadID, startedAt, plannedAt)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update failed", err.Error())
		return
	}
	p, _ = h.findProject(r.Context(), member.CompanyID, projectID)
	payload, err := h.projectPayload(r.Context(), p)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "payload failed", err.Error())
		return
	}
	response.OK(w, "Project updated", payload)
}

func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if _, err := h.findProject(r.Context(), member.CompanyID, projectID); err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Project not found", nil)
		return
	} else if err != nil {
		response.Fail(w, http.StatusInternalServerError, "lookup failed", err.Error())
		return
	}
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	if _, err := h.pool.Exec(r.Context(), `DELETE FROM projects WHERE id=$1`, projectID); err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete failed", err.Error())
		return
	}
	response.OK(w, "Project deleted", nil)
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	employeeID := chi.URLParam(r, "employeeId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND company_id=$2)`, employeeID, member.CompanyID).Scan(&exists)
	if !exists {
		response.Fail(w, http.StatusUnprocessableEntity, "Employee does not belong to this company", nil)
		return
	}
	var req struct{ Role *string `json:"role"` }
	_ = json.NewDecoder(r.Body).Decode(&req)
	_, err := h.pool.Exec(r.Context(), `INSERT INTO employee_project(employee_id,project_id,role) VALUES($1,$2,$3) ON CONFLICT(employee_id,project_id) DO UPDATE SET role=EXCLUDED.role`, employeeID, projectID, req.Role)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "add member failed", err.Error())
		return
	}
	h.writeProject(w, r, projectID)
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	employeeID := chi.URLParam(r, "employeeId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	_, _ = h.pool.Exec(r.Context(), `DELETE FROM employee_project WHERE project_id=$1 AND employee_id=$2`, projectID, employeeID)
	_, _ = h.pool.Exec(r.Context(), `UPDATE projects SET project_lead_id=NULL,updated_at=now() WHERE id=$1 AND project_lead_id=$2`, projectID, employeeID)
	h.writeProject(w, r, projectID)
}

func (h *Handler) SetLead(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct {
		EmployeeID *string `json:"employee_id"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	if req.EmployeeID != nil && *req.EmployeeID != "" {
		var exists bool
		_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND company_id=$2)`, *req.EmployeeID, member.CompanyID).Scan(&exists)
		if !exists {
			response.Fail(w, http.StatusUnprocessableEntity, "Employee does not belong to this company", nil)
			return
		}
		_, _ = h.pool.Exec(r.Context(), `INSERT INTO employee_project(employee_id,project_id,role) VALUES($1,$2,'lead') ON CONFLICT(employee_id,project_id) DO UPDATE SET role='lead'`, *req.EmployeeID, projectID)
	}
	_, err := h.pool.Exec(r.Context(), `UPDATE projects SET project_lead_id=$2,updated_at=now() WHERE id=$1`, projectID, req.EmployeeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "set lead failed", err.Error())
		return
	}
	h.writeProject(w, r, projectID)
}

func (h *Handler) AttachTeam(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	teamID := chi.URLParam(r, "teamId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM teams WHERE id=$1 AND company_id=$2)`, teamID, member.CompanyID).Scan(&exists)
	if !exists {
		response.Fail(w, http.StatusUnprocessableEntity, "Team does not belong to this company", nil)
		return
	}
	_, err := h.pool.Exec(r.Context(), `INSERT INTO project_team(project_id,team_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, projectID, teamID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "attach team failed", err.Error())
		return
	}
	h.writeProject(w, r, projectID)
}

func (h *Handler) DetachTeam(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	teamID := chi.URLParam(r, "teamId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	_, _ = h.pool.Exec(r.Context(), `DELETE FROM project_team WHERE project_id=$1 AND team_id=$2`, projectID, teamID)
	h.writeProject(w, r, projectID)
}

func (h *Handler) writeProject(w http.ResponseWriter, r *http.Request, projectID string) {
	member, _ := companyauth.MemberFromContext(r.Context())
	p, err := h.findProject(r.Context(), member.CompanyID, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "load project failed", err.Error())
		return
	}
	payload, err := h.projectPayload(r.Context(), p)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "payload failed", err.Error())
		return
	}
	response.OK(w, "", payload)
}

func (h *Handler) ListLinks(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canAccessCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `SELECT id, project_id, type, label, url FROM project_links WHERE project_id=$1 ORDER BY created_at`, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list links failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, pid, ltype, url string
		var label *string
		if err := rows.Scan(&id, &pid, &ltype, &label, &url); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		out = append(out, map[string]interface{}{"id": id, "project_id": pid, "type": ltype, "label": label, "url": url})
	}
	response.OK(w, "", out)
}

func (h *Handler) CreateLink(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct {
		Type  string  `json:"type"`
		Label *string `json:"label"`
		URL   string  `json:"url"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Type == "" || req.URL == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "type and url are required", nil)
		return
	}
	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `INSERT INTO project_links(id,project_id,type,label,url) VALUES($1,$2,$3,$4,$5)`, id, projectID, req.Type, req.Label, req.URL)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create link failed", err.Error())
		return
	}
	response.Created(w, "Link created", map[string]interface{}{"id": id, "project_id": projectID, "type": req.Type, "label": req.Label, "url": req.URL})
}

func (h *Handler) UpdateLink(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	linkID := chi.URLParam(r, "linkId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct {
		Type  *string `json:"type"`
		Label *string `json:"label"`
		URL   *string `json:"url"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `
		UPDATE project_links SET
			type=COALESCE($3,type), label=COALESCE($4,label), url=COALESCE($5,url), updated_at=now()
		WHERE id=$1 AND project_id=$2`, linkID, projectID, req.Type, req.Label, req.URL)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update link failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Link not found", nil)
		return
	}
	var id, pid, ltype, url string
	var label *string
	_ = h.pool.QueryRow(r.Context(), `SELECT id,project_id,type,label,url FROM project_links WHERE id=$1`, linkID).Scan(&id, &pid, &ltype, &label, &url)
	response.OK(w, "Link updated", map[string]interface{}{"id": id, "project_id": pid, "type": ltype, "label": label, "url": url})
}

func (h *Handler) DeleteLink(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	linkID := chi.URLParam(r, "linkId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `DELETE FROM project_links WHERE id=$1 AND project_id=$2`, linkID, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete link failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Link not found", nil)
		return
	}
	response.OK(w, "Link deleted", nil)
}

func (h *Handler) ListStatuses(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canAccessCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT ps.id, ps.project_id, ps.author_id, ps.title, ps.status, ps.description, ps.created_at,
			COALESCE(e.first_name||' '||e.last_name, '')
		FROM project_statuses ps
		LEFT JOIN employees e ON e.id=ps.author_id
		WHERE ps.project_id=$1 ORDER BY ps.created_at DESC`, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list statuses failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, pid, title, status, description string
		var authorID *string
		var createdAt time.Time
		var authorName string
		if err := rows.Scan(&id, &pid, &authorID, &title, &status, &description, &createdAt, &authorName); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		var an interface{}
		if authorName != "" {
			an = strings.TrimSpace(authorName)
		}
		out = append(out, map[string]interface{}{
			"id": id, "project_id": pid, "author_id": authorID, "author_name": an,
			"title": title, "status": status, "description": description,
			"created_at": createdAt.UTC().Format(time.RFC3339),
		})
	}
	response.OK(w, "", out)
}

func (h *Handler) CreateStatus(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct {
		Title       string `json:"title"`
		Status      string `json:"status"`
		Description string `json:"description"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Title == "" || req.Status == "" || req.Description == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "title, status, and description are required", nil)
		return
	}
	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO project_statuses(id,project_id,author_id,title,status,description) VALUES($1,$2,$3,$4,$5,$6)`,
		id, projectID, member.EmployeeID, req.Title, req.Status, req.Description)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create status failed", err.Error())
		return
	}
	name, _ := h.employeeFullName(r.Context(), member.EmployeeID)
	response.Created(w, "Status created", map[string]interface{}{
		"id": id, "project_id": projectID, "author_id": member.EmployeeID, "author_name": name,
		"title": req.Title, "status": req.Status, "description": req.Description,
		"created_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canAccessCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, file_name, mime_type, size FROM media
		WHERE model_type=$1 AND model_id=$2 AND collection_name='files' ORDER BY id`, modelProject, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list files failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id int64
		var fileName string
		var mimeType *string
		var size int64
		if err := rows.Scan(&id, &fileName, &mimeType, &size); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		out = append(out, map[string]interface{}{
			"id": id, "file_name": fileName, "mime_type": mimeType, "size": size,
			"url": fmt.Sprintf("/api/v1/media/%d/file", id),
		})
	}
	response.OK(w, "", out)
}

type attachFileRequest struct {
	TemporaryUploadID int64 `json:"temporary_upload_id"`
	MediaID           int64 `json:"media_id"`
}

func (h *Handler) AttachFile(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req attachFileRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())
	var fileName string
	err = tx.QueryRow(r.Context(), `
		SELECT file_name FROM media
		WHERE id=$1 AND model_type=$2 AND model_id=$3`,
		req.MediaID, modelTemporaryUpload, strconv.FormatInt(req.TemporaryUploadID, 10),
	).Scan(&fileName)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Media not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "media lookup failed", err.Error())
		return
	}
	tag, err := tx.Exec(r.Context(), `
		UPDATE media SET model_type=$2, model_id=$3, collection_name='files', updated_at=now()
		WHERE id=$1 AND model_type=$4 AND model_id=$5`,
		req.MediaID, modelProject, projectID, modelTemporaryUpload, strconv.FormatInt(req.TemporaryUploadID, 10))
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "attach file failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Media not found", nil)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}
	var mimeType *string
	var size int64
	_ = h.pool.QueryRow(r.Context(), `SELECT mime_type, size FROM media WHERE id=$1`, req.MediaID).Scan(&mimeType, &size)
	response.Created(w, "File attached", map[string]interface{}{
		"id": req.MediaID, "file_name": fileName, "mime_type": mimeType, "size": size,
		"url": fmt.Sprintf("/api/v1/media/%d/file", req.MediaID),
	})
}

func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	mediaID := chi.URLParam(r, "mediaId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `
		DELETE FROM media WHERE id=$1 AND model_type=$2 AND model_id=$3 AND collection_name='files'`,
		mediaID, modelProject, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete file failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "File not found", nil)
		return
	}
	response.OK(w, "File deleted", nil)
}
