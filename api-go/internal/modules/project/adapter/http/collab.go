package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

func (h *Handler) loadComments(ctx context.Context, commentableType, commentableID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT id, company_id, author_id, author_name, content, created_at
		FROM comments WHERE commentable_type=$1 AND commentable_id=$2 ORDER BY created_at`, commentableType, commentableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, companyID, authorName, content string
		var authorID *string
		var createdAt time.Time
		if err := rows.Scan(&id, &companyID, &authorID, &authorName, &content, &createdAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"id": id, "company_id": companyID, "author_id": authorID,
			"author_name": authorName, "content": content,
			"created_at": createdAt.UTC().Format(time.RFC3339),
		})
	}
	return out, rows.Err()
}

func (h *Handler) commentPayload(ctx context.Context, id string) (map[string]interface{}, error) {
	var companyID, authorName, content string
	var authorID *string
	var createdAt time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT company_id, author_id, author_name, content, created_at FROM comments WHERE id=$1`, id,
	).Scan(&companyID, &authorID, &authorName, &content, &createdAt)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id": id, "company_id": companyID, "author_id": authorID,
		"author_name": authorName, "content": content,
		"created_at": createdAt.UTC().Format(time.RFC3339),
	}, nil
}

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canAccessCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT pm.id, pm.project_id, pm.author_id, pm.title, pm.content, pm.created_at,
			COALESCE(e.first_name||' '||e.last_name, '')
		FROM project_messages pm
		LEFT JOIN employees e ON e.id=pm.author_id
		WHERE pm.project_id=$1 ORDER BY pm.created_at DESC`, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list messages failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, pid, title, content string
		var authorID *string
		var createdAt time.Time
		var authorName string
		if err := rows.Scan(&id, &pid, &authorID, &title, &content, &createdAt, &authorName); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		comments, _ := h.loadComments(r.Context(), commentTypeMessage, id)
		var an interface{}
		if authorName != "" {
			an = authorName
		}
		out = append(out, map[string]interface{}{
			"id": id, "project_id": pid, "author_id": authorID, "author_name": an,
			"title": title, "content": content, "comments": comments,
			"created_at": createdAt.UTC().Format(time.RFC3339),
		})
	}
	response.OK(w, "", out)
}

func (h *Handler) ShowMessage(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	messageID := chi.URLParam(r, "messageId")
	if !h.canAccessCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var id, pid, title, content string
	var authorID *string
	var createdAt time.Time
	err := h.pool.QueryRow(r.Context(), `
		SELECT id, project_id, author_id, title, content, created_at
		FROM project_messages WHERE id=$1 AND project_id=$2`, messageID, projectID,
	).Scan(&id, &pid, &authorID, &title, &content, &createdAt)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Message not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "lookup failed", err.Error())
		return
	}
	comments, _ := h.loadComments(r.Context(), commentTypeMessage, id)
	var authorName interface{}
	if authorID != nil {
		authorName, _ = h.employeeFullName(r.Context(), *authorID)
	}
	response.OK(w, "", map[string]interface{}{
		"id": id, "project_id": pid, "author_id": authorID, "author_name": authorName,
		"title": title, "content": content, "comments": comments,
		"created_at": createdAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Title == "" || req.Content == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "title and content are required", nil)
		return
	}
	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO project_messages(id,project_id,author_id,title,content) VALUES($1,$2,$3,$4,$5)`,
		id, projectID, member.EmployeeID, req.Title, req.Content)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create message failed", err.Error())
		return
	}
	name, _ := h.employeeFullName(r.Context(), member.EmployeeID)
	response.Created(w, "Message created", map[string]interface{}{
		"id": id, "project_id": projectID, "author_id": member.EmployeeID, "author_name": name,
		"title": req.Title, "content": req.Content, "comments": []map[string]interface{}{},
		"created_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) UpdateMessage(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	messageID := chi.URLParam(r, "messageId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct {
		Title   *string `json:"title"`
		Content *string `json:"content"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `
		UPDATE project_messages SET title=COALESCE($3,title), content=COALESCE($4,content), updated_at=now()
		WHERE id=$1 AND project_id=$2`, messageID, projectID, req.Title, req.Content)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update message failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Message not found", nil)
		return
	}
	h.ShowMessage(w, r)
}

func (h *Handler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	messageID := chi.URLParam(r, "messageId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `DELETE FROM project_messages WHERE id=$1 AND project_id=$2`, messageID, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete message failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Message not found", nil)
		return
	}
	response.OK(w, "Message deleted", nil)
}

func (h *Handler) CreateMessageComment(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	messageID := chi.URLParam(r, "messageId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM project_messages WHERE id=$1 AND project_id=$2)`, messageID, projectID).Scan(&exists)
	if !exists {
		response.Fail(w, http.StatusNotFound, "Message not found", nil)
		return
	}
	var req struct{ Content string `json:"content"` }
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Content == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "content is required", nil)
		return
	}
	name, _ := h.employeeFullName(r.Context(), member.EmployeeID)
	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO comments(id,company_id,author_id,author_name,content,commentable_id,commentable_type)
		VALUES($1,$2,$3,$4,$5,$6,$7)`,
		id, member.CompanyID, member.EmployeeID, name, req.Content, messageID, commentTypeMessage)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create comment failed", err.Error())
		return
	}
	payload, _ := h.commentPayload(r.Context(), id)
	response.Created(w, "Comment created", payload)
}

func (h *Handler) UpdateMessageComment(w http.ResponseWriter, r *http.Request) {
	h.updateComment(w, r, commentTypeMessage)
}

func (h *Handler) DeleteMessageComment(w http.ResponseWriter, r *http.Request) {
	h.deleteComment(w, r, commentTypeMessage)
}

func (h *Handler) updateComment(w http.ResponseWriter, r *http.Request, commentableType string) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	commentID := chi.URLParam(r, "commentId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct{ Content string `json:"content"` }
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Content == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "content is required", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `
		UPDATE comments SET content=$3, updated_at=now()
		WHERE id=$1 AND commentable_type=$2 AND EXISTS(
			SELECT 1 FROM project_messages pm WHERE pm.id=comments.commentable_id AND pm.project_id=$4
			UNION SELECT 1 FROM project_tasks pt WHERE pt.id=comments.commentable_id AND pt.project_id=$4
		)`, commentID, commentableType, req.Content, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update comment failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Comment not found", nil)
		return
	}
	payload, _ := h.commentPayload(r.Context(), commentID)
	response.OK(w, "Comment updated", payload)
}

func (h *Handler) deleteComment(w http.ResponseWriter, r *http.Request, commentableType string) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	commentID := chi.URLParam(r, "commentId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `
		DELETE FROM comments WHERE id=$1 AND commentable_type=$2 AND EXISTS(
			SELECT 1 FROM project_messages pm WHERE pm.id=comments.commentable_id AND pm.project_id=$3
			UNION SELECT 1 FROM project_tasks pt WHERE pt.id=comments.commentable_id AND pt.project_id=$3
		)`, commentID, commentableType, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete comment failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Comment not found", nil)
		return
	}
	response.OK(w, "Comment deleted", nil)
}

func (h *Handler) ListDecisions(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canAccessCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT pd.id, pd.project_id, pd.author_id, pd.title, pd.decided_at,
			COALESCE(e.first_name||' '||e.last_name, '')
		FROM project_decisions pd
		LEFT JOIN employees e ON e.id=pd.author_id
		WHERE pd.project_id=$1 ORDER BY pd.created_at DESC`, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list decisions failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, pid, title string
		var authorID *string
		var decidedAt *time.Time
		var authorName string
		if err := rows.Scan(&id, &pid, &authorID, &title, &decidedAt, &authorName); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		deciders, _ := h.loadDeciders(r.Context(), id)
		var an interface{}
		if authorName != "" {
			an = authorName
		}
		out = append(out, map[string]interface{}{
			"id": id, "project_id": pid, "author_id": authorID, "author_name": an,
			"title": title, "decided_at": isoDate(decidedAt), "deciders": deciders,
		})
	}
	response.OK(w, "", out)
}

func (h *Handler) loadDeciders(ctx context.Context, decisionID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT e.id FROM employees e
		JOIN project_decision_deciders pdd ON pdd.employee_id=e.id
		WHERE pdd.project_decision_id=$1`, decisionID)
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

func (h *Handler) CreateDecision(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct {
		Title      string   `json:"title"`
		DecidedAt  *string  `json:"decided_at"`
		DeciderIDs []string `json:"decider_ids"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Title == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "title is required", nil)
		return
	}
	id := uuidv7.New()
	var decidedAt *time.Time
	if req.DecidedAt != nil && *req.DecidedAt != "" {
		t, err := time.Parse("2006-01-02", *req.DecidedAt)
		if err != nil {
			response.Fail(w, http.StatusUnprocessableEntity, "invalid decided_at", nil)
			return
		}
		decidedAt = &t
	}
	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())
	_, err = tx.Exec(r.Context(), `
		INSERT INTO project_decisions(id,project_id,author_id,title,decided_at) VALUES($1,$2,$3,$4,$5)`,
		id, projectID, member.EmployeeID, req.Title, decidedAt)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create decision failed", err.Error())
		return
	}
	for _, deciderID := range req.DeciderIDs {
		_, err = tx.Exec(r.Context(), `INSERT INTO project_decision_deciders(project_decision_id,employee_id) VALUES($1,$2)`, id, deciderID)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "add decider failed", err.Error())
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}
	deciders, _ := h.loadDeciders(r.Context(), id)
	name, _ := h.employeeFullName(r.Context(), member.EmployeeID)
	response.Created(w, "Decision created", map[string]interface{}{
		"id": id, "project_id": projectID, "author_id": member.EmployeeID, "author_name": name,
		"title": req.Title, "decided_at": isoDate(decidedAt), "deciders": deciders,
	})
}

func (h *Handler) DeleteDecision(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	decisionID := chi.URLParam(r, "decisionId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `DELETE FROM project_decisions WHERE id=$1 AND project_id=$2`, decisionID, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete decision failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Decision not found", nil)
		return
	}
	response.OK(w, "Decision deleted", nil)
}

func (h *Handler) taskPayload(ctx context.Context, taskID string) (map[string]interface{}, error) {
	var id, pid, title string
	var listID, authorID, assigneeID *string
	var description *string
	var completed bool
	var completedAt *time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT id, project_id, project_task_list_id, author_id, assignee_id, title, description, completed, completed_at
		FROM project_tasks WHERE id=$1`, taskID,
	).Scan(&id, &pid, &listID, &authorID, &assigneeID, &title, &description, &completed, &completedAt)
	if err != nil {
		return nil, err
	}
	var assignee interface{}
	if assigneeID != nil {
		assignee, _ = h.employeeSummary(ctx, *assigneeID)
	}
	comments, _ := h.loadComments(ctx, commentTypeTask, id)
	return map[string]interface{}{
		"id": id, "project_id": pid, "project_task_list_id": listID,
		"author_id": authorID, "assignee_id": assigneeID, "assignee": assignee,
		"title": title, "description": description, "completed": completed,
		"completed_at": isoTime(completedAt), "comments": comments,
	}, nil
}

func (h *Handler) ListTaskLists(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canAccessCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, project_id, author_id, title, description FROM project_task_lists
		WHERE project_id=$1 ORDER BY created_at`, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list task lists failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, pid, title string
		var authorID *string
		var description *string
		if err := rows.Scan(&id, &pid, &authorID, &title, &description); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		tasks, _ := h.loadTasksForList(r.Context(), id)
		out = append(out, map[string]interface{}{
			"id": id, "project_id": pid, "author_id": authorID,
			"title": title, "description": description, "tasks": tasks,
		})
	}
	response.OK(w, "", out)
}

func (h *Handler) loadTasksForList(ctx context.Context, listID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `SELECT id FROM project_tasks WHERE project_task_list_id=$1 ORDER BY created_at`, listID)
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
		p, err := h.taskPayload(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (h *Handler) CreateTaskList(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct {
		Title       string  `json:"title"`
		Description *string `json:"description"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Title == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "title is required", nil)
		return
	}
	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO project_task_lists(id,project_id,author_id,title,description) VALUES($1,$2,$3,$4,$5)`,
		id, projectID, member.EmployeeID, req.Title, req.Description)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create task list failed", err.Error())
		return
	}
	response.Created(w, "Task list created", map[string]interface{}{
		"id": id, "project_id": projectID, "author_id": member.EmployeeID,
		"title": req.Title, "description": req.Description, "tasks": []map[string]interface{}{},
	})
}

func (h *Handler) UpdateTaskList(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	listID := chi.URLParam(r, "listId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `
		UPDATE project_task_lists SET title=COALESCE($3,title), description=COALESCE($4,description), updated_at=now()
		WHERE id=$1 AND project_id=$2`, listID, projectID, req.Title, req.Description)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update task list failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Task list not found", nil)
		return
	}
	tasks, _ := h.loadTasksForList(r.Context(), listID)
	var title string
	var authorID *string
	var description *string
	_ = h.pool.QueryRow(r.Context(), `SELECT title, author_id, description FROM project_task_lists WHERE id=$1`, listID).Scan(&title, &authorID, &description)
	response.OK(w, "Task list updated", map[string]interface{}{
		"id": listID, "project_id": projectID, "author_id": authorID,
		"title": title, "description": description, "tasks": tasks,
	})
}

func (h *Handler) DeleteTaskList(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	listID := chi.URLParam(r, "listId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `DELETE FROM project_task_lists WHERE id=$1 AND project_id=$2`, listID, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete task list failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Task list not found", nil)
		return
	}
	response.OK(w, "Task list deleted", nil)
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct {
		Title             string  `json:"title"`
		Description       *string `json:"description"`
		ProjectTaskListID *string `json:"project_task_list_id"`
		AssigneeID        *string `json:"assignee_id"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Title == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "title is required", nil)
		return
	}
	if req.ProjectTaskListID != nil && *req.ProjectTaskListID != "" {
		var exists bool
		_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM project_task_lists WHERE id=$1 AND project_id=$2)`, *req.ProjectTaskListID, projectID).Scan(&exists)
		if !exists {
			response.Fail(w, http.StatusNotFound, "Task list not found", nil)
			return
		}
	}
	if req.AssigneeID != nil && *req.AssigneeID != "" {
		var exists bool
		_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND company_id=$2)`, *req.AssigneeID, member.CompanyID).Scan(&exists)
		if !exists {
			response.Fail(w, http.StatusNotFound, "Employee not found", nil)
			return
		}
	}
	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO project_tasks(id,project_id,project_task_list_id,author_id,assignee_id,title,description)
		VALUES($1,$2,$3,$4,$5,$6,$7)`,
		id, projectID, req.ProjectTaskListID, member.EmployeeID, req.AssigneeID, req.Title, req.Description)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create task failed", err.Error())
		return
	}
	payload, _ := h.taskPayload(r.Context(), id)
	response.Created(w, "Task created", payload)
}

func (h *Handler) ShowTask(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	taskID := chi.URLParam(r, "taskId")
	if !h.canAccessCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM project_tasks WHERE id=$1 AND project_id=$2)`, taskID, projectID).Scan(&exists)
	if !exists {
		response.Fail(w, http.StatusNotFound, "Task not found", nil)
		return
	}
	payload, err := h.taskPayload(r.Context(), taskID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "payload failed", err.Error())
		return
	}
	response.OK(w, "", payload)
}

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	taskID := chi.URLParam(r, "taskId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var req struct {
		Title             *string `json:"title"`
		Description       *string `json:"description"`
		ProjectTaskListID *string `json:"project_task_list_id"`
		AssigneeID        *string `json:"assignee_id"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	if req.ProjectTaskListID != nil && *req.ProjectTaskListID != "" {
		var exists bool
		_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM project_task_lists WHERE id=$1 AND project_id=$2)`, *req.ProjectTaskListID, projectID).Scan(&exists)
		if !exists {
			response.Fail(w, http.StatusNotFound, "Task list not found", nil)
			return
		}
	}
	if req.AssigneeID != nil && *req.AssigneeID != "" {
		var exists bool
		_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND company_id=$2)`, *req.AssigneeID, member.CompanyID).Scan(&exists)
		if !exists {
			response.Fail(w, http.StatusNotFound, "Employee not found", nil)
			return
		}
	}
	tag, err := h.pool.Exec(r.Context(), `
		UPDATE project_tasks SET
			title=COALESCE($3,title), description=COALESCE($4,description),
			project_task_list_id=COALESCE($5,project_task_list_id), assignee_id=COALESCE($6,assignee_id), updated_at=now()
		WHERE id=$1 AND project_id=$2`, taskID, projectID, req.Title, req.Description, req.ProjectTaskListID, req.AssigneeID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update task failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Task not found", nil)
		return
	}
	payload, _ := h.taskPayload(r.Context(), taskID)
	response.OK(w, "Task updated", payload)
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	taskID := chi.URLParam(r, "taskId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `DELETE FROM project_tasks WHERE id=$1 AND project_id=$2`, taskID, projectID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete task failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Task not found", nil)
		return
	}
	response.OK(w, "Task deleted", nil)
}

func (h *Handler) ToggleTask(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	taskID := chi.URLParam(r, "taskId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var completed bool
	err := h.pool.QueryRow(r.Context(), `SELECT completed FROM project_tasks WHERE id=$1 AND project_id=$2`, taskID, projectID).Scan(&completed)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Task not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "lookup failed", err.Error())
		return
	}
	newCompleted := !completed
	var completedAt *time.Time
	if newCompleted {
		now := time.Now().UTC()
		completedAt = &now
	}
	_, err = h.pool.Exec(r.Context(), `
		UPDATE project_tasks SET completed=$3, completed_at=$4, updated_at=now() WHERE id=$1 AND project_id=$2`,
		taskID, projectID, newCompleted, completedAt)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "toggle task failed", err.Error())
		return
	}
	payload, _ := h.taskPayload(r.Context(), taskID)
	response.OK(w, "Task toggled", payload)
}

func (h *Handler) TaskTimeEntries(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	taskID := chi.URLParam(r, "taskId")
	if !h.canAccessCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM project_tasks WHERE id=$1 AND project_id=$2)`, taskID, projectID).Scan(&exists)
	if !exists {
		response.Fail(w, http.StatusNotFound, "Task not found", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, timesheet_id, employee_id, duration, happened_at, description
		FROM time_tracking_entries WHERE project_task_id=$1 ORDER BY happened_at DESC`, taskID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list time entries failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, tsid, eid string
		var duration int
		var day time.Time
		var description *string
		if err := rows.Scan(&id, &tsid, &eid, &duration, &day, &description); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		out = append(out, map[string]interface{}{
			"id": id, "timesheet_id": tsid, "employee_id": eid,
			"duration": duration, "happened_at": day.Format("2006-01-02"), "description": description,
		})
	}
	response.OK(w, "", out)
}

func (h *Handler) CreateTaskComment(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	projectID := chi.URLParam(r, "projectId")
	taskID := chi.URLParam(r, "taskId")
	if !h.canManageCtx(r.Context(), member, projectID) {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM project_tasks WHERE id=$1 AND project_id=$2)`, taskID, projectID).Scan(&exists)
	if !exists {
		response.Fail(w, http.StatusNotFound, "Task not found", nil)
		return
	}
	var req struct{ Content string `json:"content"` }
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Content == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "content is required", nil)
		return
	}
	name, _ := h.employeeFullName(r.Context(), member.EmployeeID)
	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO comments(id,company_id,author_id,author_name,content,commentable_id,commentable_type)
		VALUES($1,$2,$3,$4,$5,$6,$7)`,
		id, member.CompanyID, member.EmployeeID, name, req.Content, taskID, commentTypeTask)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create comment failed", err.Error())
		return
	}
	payload, _ := h.commentPayload(r.Context(), id)
	response.Created(w, "Comment created", payload)
}

func (h *Handler) UpdateTaskComment(w http.ResponseWriter, r *http.Request) {
	h.updateComment(w, r, commentTypeTask)
}

func (h *Handler) DeleteTaskComment(w http.ResponseWriter, r *http.Request) {
	h.deleteComment(w, r, commentTypeTask)
}
