package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

type boardRow struct {
	ID, ProjectID, Name string
}

type sprintRow struct {
	ID, ProjectID, Name string
	BoardID             *string
	Active              bool
	Position            *int
	StartedAt           *time.Time
	CompletedAt         *time.Time
}

func (h *Handler) findBoard(ctx context.Context, projectID, boardID string) (boardRow, error) {
	var b boardRow
	err := h.pool.QueryRow(ctx, `SELECT id, project_id, name FROM project_boards WHERE id=$1 AND project_id=$2`, boardID, projectID).Scan(&b.ID, &b.ProjectID, &b.Name)
	return b, err
}

func (h *Handler) findSprint(ctx context.Context, projectID, boardID, sprintID string) (sprintRow, error) {
	var s sprintRow
	err := h.pool.QueryRow(ctx, `
		SELECT id, project_id, project_board_id, name, active, position, started_at, completed_at
		FROM project_sprints WHERE id=$1 AND project_id=$2 AND project_board_id=$3`,
		sprintID, projectID, boardID,
	).Scan(&s.ID, &s.ProjectID, &s.BoardID, &s.Name, &s.Active, &s.Position, &s.StartedAt, &s.CompletedAt)
	return s, err
}

func (h *Handler) loadIssueAssignees(ctx context.Context, issueID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT e.id FROM employees e
		JOIN project_issue_assignees pia ON pia.employee_id=e.id
		WHERE pia.project_issue_id=$1`, issueID)
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

func (h *Handler) issuePayload(ctx context.Context, issueID string, sprintID *string) (map[string]interface{}, error) {
	var id, pid, title, key, slug string
	var boardID, reporterID, issueTypeID *string
	var isSeparator bool
	var idInProject int
	var description *string
	var storyPoints *int
	err := h.pool.QueryRow(ctx, `
		SELECT id, project_id, project_board_id, reporter_id, issue_type_id, is_separator,
			id_in_project, key, slug, title, description, story_points
		FROM project_issues WHERE id=$1`, issueID,
	).Scan(&id, &pid, &boardID, &reporterID, &issueTypeID, &isSeparator, &idInProject, &key, &slug, &title, &description, &storyPoints)
	if err != nil {
		return nil, err
	}
	var issueType interface{}
	if issueTypeID != nil {
		var typeID, companyID, name string
		var icon *string
		if h.pool.QueryRow(ctx, `SELECT id, company_id, name, icon FROM issue_types WHERE id=$1`, *issueTypeID).Scan(&typeID, &companyID, &name, &icon) == nil {
			issueType = map[string]interface{}{"id": typeID, "company_id": companyID, "name": name, "icon": icon}
		}
	}
	assignees, _ := h.loadIssueAssignees(ctx, id)
	var position interface{}
	if sprintID != nil {
		var pos int
		if h.pool.QueryRow(ctx, `SELECT position FROM project_issue_project_sprint WHERE project_issue_id=$1 AND project_sprint_id=$2`, id, *sprintID).Scan(&pos) == nil {
			position = pos
		}
	}
	return map[string]interface{}{
		"id": id, "project_id": pid, "project_board_id": boardID, "reporter_id": reporterID,
		"issue_type_id": issueTypeID, "issue_type": issueType, "is_separator": isSeparator,
		"id_in_project": idInProject, "key": key, "slug": slug, "title": title,
		"description": description, "story_points": storyPoints, "position": position,
		"assignees": assignees,
	}, nil
}

func (h *Handler) loadSprintIssues(ctx context.Context, sprintID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT pi.id FROM project_issues pi
		JOIN project_issue_project_sprint pips ON pips.project_issue_id=pi.id
		WHERE pips.project_sprint_id=$1 ORDER BY pips.position`, sprintID)
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
		p, err := h.issuePayload(ctx, id, &sprintID)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (h *Handler) sprintPayload(ctx context.Context, s sprintRow, withIssues bool) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"id": s.ID, "project_id": s.ProjectID, "project_board_id": s.BoardID,
		"name": s.Name, "active": s.Active, "position": s.Position,
		"started_at": isoTime(s.StartedAt), "completed_at": isoTime(s.CompletedAt),
	}
	if withIssues {
		issues, err := h.loadSprintIssues(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		payload["issues"] = issues
	}
	return payload, nil
}

func (h *Handler) boardPayload(ctx context.Context, b boardRow, detailed bool) (map[string]interface{}, error) {
	payload := map[string]interface{}{"id": b.ID, "project_id": b.ProjectID, "name": b.Name}
	if !detailed {
		return payload, nil
	}
	rows, err := h.pool.Query(ctx, `
		SELECT id, project_id, project_board_id, name, active, position, started_at, completed_at
		FROM project_sprints WHERE project_board_id=$1 ORDER BY position`, b.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sprints := []map[string]interface{}{}
	for rows.Next() {
		var s sprintRow
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.BoardID, &s.Name, &s.Active, &s.Position, &s.StartedAt, &s.CompletedAt); err != nil {
			return nil, err
		}
		sp, err := h.sprintPayload(ctx, s, true)
		if err != nil {
			return nil, err
		}
		sprints = append(sprints, sp)
	}
	payload["sprints"] = sprints
	return payload, nil
}

func (h *Handler) ListBoards(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canAccessCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `SELECT id, project_id, name FROM project_boards WHERE project_id=$1 ORDER BY name`, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list boards failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var b boardRow
		if err := rows.Scan(&b.ID, &b.ProjectID, &b.Name); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		p, _ := h.boardPayload(r.Context(), b, false)
		out = append(out, p)
	}
	response.OK(w, "", out)
}

func (h *Handler) CreateBoard(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct{ Name string `json:"name"` }
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Name == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "name is required", nil)
		return
	}
	boardID := uuidv7.New()
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())
	_, err = tx.Exec(r.Context(), `INSERT INTO project_boards(id,project_id,name) VALUES($1,$2,$3)`, boardID, projectID, req.Name)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create board failed", err.Error())
		return
	}
	pos0, pos1 := 0, 1
	_, err = tx.Exec(r.Context(), `
		INSERT INTO project_sprints(id,project_id,project_board_id,name,active,position) VALUES($1,$2,$3,'Backlog',false,$4)`,
		uuidv7.New(), projectID, boardID, pos0)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create backlog failed", err.Error())
		return
	}
	_, err = tx.Exec(r.Context(), `
		INSERT INTO project_sprints(id,project_id,project_board_id,name,active,position) VALUES($1,$2,$3,'Sprint 1',false,$4)`,
		uuidv7.New(), projectID, boardID, pos1)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create sprint failed", err.Error())
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}
	response.Created(w, "Board created", map[string]interface{}{"id": boardID, "project_id": projectID, "name": req.Name})
}

func (h *Handler) ShowBoard(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	boardID := chi.URLParam(r, "boardId")
	if !h.canAccessCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	b, err := h.findBoard(r.Context(), projectID, boardID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Board not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "lookup failed", err.Error())
		return
	}
	payload, err := h.boardPayload(r.Context(), b, true)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "payload failed", err.Error())
		return
	}
	response.OK(w, "", payload)
}

func (h *Handler) UpdateBoard(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	boardID := chi.URLParam(r, "boardId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct{ Name string `json:"name"` }
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Name == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "name is required", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `UPDATE project_boards SET name=$3, updated_at=now() WHERE id=$1 AND project_id=$2`, boardID, projectID, req.Name)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update board failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Board not found", nil)
		return
	}
	response.OK(w, "Board updated", map[string]interface{}{"id": boardID, "project_id": projectID, "name": req.Name})
}

func (h *Handler) DeleteBoard(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	boardID := chi.URLParam(r, "boardId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `DELETE FROM project_boards WHERE id=$1 AND project_id=$2`, boardID, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete board failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Board not found", nil)
		return
	}
	response.OK(w, "Board deleted", nil)
}

func (h *Handler) findOrCreateBacklog(ctx context.Context, board boardRow) (sprintRow, error) {
	var s sprintRow
	err := h.pool.QueryRow(ctx, `
		SELECT id, project_id, project_board_id, name, active, position, started_at, completed_at
		FROM project_sprints WHERE project_board_id=$1 AND name='Backlog'`, board.ID,
	).Scan(&s.ID, &s.ProjectID, &s.BoardID, &s.Name, &s.Active, &s.Position, &s.StartedAt, &s.CompletedAt)
	if err == nil {
		return s, nil
	}
	if err != pgx.ErrNoRows {
		return s, err
	}
	pos := 0
	s.ID = uuidv7.New()
	s.ProjectID = board.ProjectID
	s.BoardID = &board.ID
	s.Name = "Backlog"
	s.Active = false
	s.Position = &pos
	_, err = h.pool.Exec(ctx, `
		INSERT INTO project_sprints(id,project_id,project_board_id,name,active,position) VALUES($1,$2,$3,$4,$5,$6)`,
		s.ID, s.ProjectID, board.ID, s.Name, s.Active, pos)
	return s, err
}

func (h *Handler) Backlog(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	boardID := chi.URLParam(r, "boardId")
	if !h.canAccessCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	b, err := h.findBoard(r.Context(), projectID, boardID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Board not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "lookup failed", err.Error())
		return
	}
	s, err := h.findOrCreateBacklog(r.Context(), b)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "backlog failed", err.Error())
		return
	}
	payload, err := h.sprintPayload(r.Context(), s, true)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "payload failed", err.Error())
		return
	}
	response.OK(w, "", payload)
}

func (h *Handler) StartSprint(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	boardID := chi.URLParam(r, "boardId")
	sprintID := chi.URLParam(r, "sprintId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	s, err := h.findSprint(r.Context(), projectID, boardID, sprintID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Sprint not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "lookup failed", err.Error())
		return
	}
	if s.Name == "Backlog" {
		response.Fail(w, http.StatusConflict, "Backlog sprint cannot be started", nil)
		return
	}
	now := time.Now().UTC()
	_, err = h.pool.Exec(r.Context(), `
		UPDATE project_sprints SET active=true, started_at=$2, completed_at=NULL, updated_at=now() WHERE id=$1`,
		sprintID, now)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "start sprint failed", err.Error())
		return
	}
	s.Active = true
	s.StartedAt = &now
	s.CompletedAt = nil
	payload, _ := h.sprintPayload(r.Context(), s, true)
	response.OK(w, "Sprint started", payload)
}

func (h *Handler) ToggleSprint(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	boardID := chi.URLParam(r, "boardId")
	sprintID := chi.URLParam(r, "sprintId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	s, err := h.findSprint(r.Context(), projectID, boardID, sprintID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Sprint not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "lookup failed", err.Error())
		return
	}
	if s.Name == "Backlog" {
		response.Fail(w, http.StatusConflict, "Backlog sprint cannot be toggled", nil)
		return
	}
	if s.CompletedAt != nil {
		_, err = h.pool.Exec(r.Context(), `UPDATE project_sprints SET active=false, completed_at=NULL, updated_at=now() WHERE id=$1`, sprintID)
		s.Active = false
		s.CompletedAt = nil
	} else {
		now := time.Now().UTC()
		_, err = h.pool.Exec(r.Context(), `UPDATE project_sprints SET active=false, completed_at=$2, updated_at=now() WHERE id=$1`, sprintID, now)
		s.Active = false
		s.CompletedAt = &now
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "toggle sprint failed", err.Error())
		return
	}
	payload, _ := h.sprintPayload(r.Context(), s, true)
	response.OK(w, "Sprint toggled", payload)
}

func (h *Handler) CreateIssue(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	boardID := chi.URLParam(r, "boardId")
	sprintID := chi.URLParam(r, "sprintId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	if _, err := h.findBoard(r.Context(), projectID, boardID); err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Board not found", nil)
		return
	}
	if _, err := h.findSprint(r.Context(), projectID, boardID, sprintID); err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Sprint not found", nil)
		return
	}
	var req struct {
		Title        string  `json:"title"`
		Description  *string `json:"description"`
		IssueTypeID  *string `json:"issue_type_id"`
		IsSeparator  *bool   `json:"is_separator"`
		StoryPoints  *int    `json:"story_points"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Title == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "title is required", nil)
		return
	}
	if req.IssueTypeID != nil && *req.IssueTypeID != "" {
		var exists bool
		_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM issue_types WHERE id=$1 AND company_id=$2)`, *req.IssueTypeID, member.CompanyID).Scan(&exists)
		if !exists {
			response.Fail(w, http.StatusNotFound, "Issue type not found", nil)
			return
		}
	}
	var shortCode string
	_ = h.pool.QueryRow(r.Context(), `SELECT COALESCE(short_code,'PRJ') FROM projects WHERE id=$1`, projectID).Scan(&shortCode)
	var maxID int
	_ = h.pool.QueryRow(r.Context(), `SELECT COALESCE(MAX(id_in_project),0) FROM project_issues WHERE project_id=$1`, projectID).Scan(&maxID)
	idInProject := maxID + 1
	key := shortCode + "-" + strconv.Itoa(idInProject)
	isSep := false
	if req.IsSeparator != nil {
		isSep = *req.IsSeparator
	}
	issueID := uuidv7.New()
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())
	_, err = tx.Exec(r.Context(), `
		INSERT INTO project_issues(id,project_id,project_board_id,reporter_id,issue_type_id,is_separator,
			id_in_project,key,slug,title,description,story_points)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		issueID, projectID, boardID, member.EmployeeID, req.IssueTypeID, isSep,
		idInProject, key, slugify(req.Title), req.Title, req.Description, req.StoryPoints)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create issue failed", err.Error())
		return
	}
	var maxPos int
	_ = tx.QueryRow(r.Context(), `SELECT COALESCE(MAX(position),0) FROM project_issue_project_sprint WHERE project_sprint_id=$1`, sprintID).Scan(&maxPos)
	_, err = tx.Exec(r.Context(), `
		INSERT INTO project_issue_project_sprint(project_issue_id,project_sprint_id,position) VALUES($1,$2,$3)`,
		issueID, sprintID, maxPos+1)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "attach issue failed", err.Error())
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}
	payload, _ := h.issuePayload(r.Context(), issueID, &sprintID)
	response.Created(w, "Issue created", payload)
}

func (h *Handler) ReorderIssue(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	sprintID := chi.URLParam(r, "sprintId")
	issueID := chi.URLParam(r, "issueId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct{ Position int `json:"position"` }
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	var pastPosition int
	err := h.pool.QueryRow(r.Context(), `
		SELECT position FROM project_issue_project_sprint WHERE project_sprint_id=$1 AND project_issue_id=$2`,
		sprintID, issueID).Scan(&pastPosition)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Issue is not in this sprint", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "lookup failed", err.Error())
		return
	}
	position := req.Position
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())
	if position > pastPosition {
		_, err = tx.Exec(r.Context(), `
			UPDATE project_issue_project_sprint SET position=position-1
			WHERE project_sprint_id=$1 AND position>$2 AND position<=$3`,
			sprintID, pastPosition, position)
	} else {
		_, err = tx.Exec(r.Context(), `
			UPDATE project_issue_project_sprint SET position=position+1
			WHERE project_sprint_id=$1 AND position>=$2 AND position<$3`,
			sprintID, position, pastPosition)
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "reorder failed", err.Error())
		return
	}
	_, err = tx.Exec(r.Context(), `
		UPDATE project_issue_project_sprint SET position=$3
		WHERE project_sprint_id=$1 AND project_issue_id=$2`, sprintID, issueID, position)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "reorder failed", err.Error())
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}
	payload, _ := h.issuePayload(r.Context(), issueID, &sprintID)
	response.OK(w, "Issue reordered", payload)
}

func (h *Handler) DeleteIssue(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	issueID := chi.URLParam(r, "issueId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `DELETE FROM project_issues WHERE id=$1 AND project_id=$2`, issueID, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete issue failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Issue not found", nil)
		return
	}
	response.OK(w, "Issue deleted", nil)
}

func (h *Handler) AddIssueAssignee(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	issueID := chi.URLParam(r, "issueId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct{ EmployeeID string `json:"employee_id"` }
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.EmployeeID == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "employee_id is required", nil)
		return
	}
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND company_id=$2)`, req.EmployeeID, member.CompanyID).Scan(&exists)
	if !exists {
		response.Fail(w, http.StatusUnprocessableEntity, "Employee does not belong to this company", nil)
		return
	}
	var issueExists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM project_issues WHERE id=$1 AND project_id=$2)`, issueID, projectID).Scan(&issueExists)
	if !issueExists {
		response.Fail(w, http.StatusNotFound, "Issue not found", nil)
		return
	}
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO project_issue_assignees(project_issue_id,employee_id) VALUES($1,$2) ON CONFLICT DO NOTHING`,
		issueID, req.EmployeeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "add assignee failed", err.Error())
		return
	}
	payload, _ := h.issuePayload(r.Context(), issueID, nil)
	response.OK(w, "Assignee added", payload)
}

func (h *Handler) RemoveIssueAssignee(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	issueID := chi.URLParam(r, "issueId")
	assigneeID := chi.URLParam(r, "assigneeId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var issueExists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM project_issues WHERE id=$1 AND project_id=$2)`, issueID, projectID).Scan(&issueExists)
	if !issueExists {
		response.Fail(w, http.StatusNotFound, "Issue not found", nil)
		return
	}
	_, _ = h.pool.Exec(r.Context(), `DELETE FROM project_issue_assignees WHERE project_issue_id=$1 AND employee_id=$2`, issueID, assigneeID)
	payload, _ := h.issuePayload(r.Context(), issueID, nil)
	response.OK(w, "Assignee removed", payload)
}

func (h *Handler) SetIssuePoints(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	issueID := chi.URLParam(r, "issueId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct{ StoryPoints *int `json:"story_points"` }
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `
		UPDATE project_issues SET story_points=$3, updated_at=now() WHERE id=$1 AND project_id=$2`,
		issueID, projectID, req.StoryPoints)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "set points failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Issue not found", nil)
		return
	}
	payload, _ := h.issuePayload(r.Context(), issueID, nil)
	response.OK(w, "Story points updated", payload)
}
