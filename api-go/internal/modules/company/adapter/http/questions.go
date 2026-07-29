package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/audit"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

type createQuestionRequest struct {
	Title string `json:"title"`
}

type updateQuestionRequest struct {
	Title *string `json:"title"`
}

type answerRequest struct {
	Body string `json:"body"`
}

// ListQuestions handles GET /companies/{companyId}/questions.
func (h *Handler) ListQuestions(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")

	rows, err := h.pool.Query(r.Context(), `
		SELECT id FROM questions WHERE company_id = $1 ORDER BY created_at DESC`, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list questions failed", err.Error())
		return
	}
	defer rows.Close()

	list := []map[string]interface{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		payload, err := h.questionPayload(r.Context(), id, companyID, false)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "question payload failed", err.Error())
			return
		}
		list = append(list, payload)
	}
	response.OK(w, "", list)
}

// ActiveQuestion handles GET /companies/{companyId}/questions/active.
func (h *Handler) ActiveQuestion(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")

	var id string
	err := h.pool.QueryRow(r.Context(), `
		SELECT id FROM questions WHERE company_id = $1 AND active = true LIMIT 1`, companyID,
	).Scan(&id)
	if err == pgx.ErrNoRows {
		response.OK(w, "", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "active question lookup failed", err.Error())
		return
	}

	payload, err := h.questionPayload(r.Context(), id, companyID, true)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "question payload failed", err.Error())
		return
	}
	myAnswer, _ := h.myAnswer(r.Context(), id, member.EmployeeID)
	payload["my_answer"] = myAnswer
	response.OK(w, "", payload)
}

// ShowQuestion handles GET /companies/{companyId}/questions/{questionId}.
func (h *Handler) ShowQuestion(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	questionID := chi.URLParam(r, "questionId")

	payload, err := h.questionPayload(r.Context(), questionID, companyID, true)
	if err != nil {
		if err == pgx.ErrNoRows {
			response.Fail(w, http.StatusNotFound, "Question not found", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "question payload failed", err.Error())
		return
	}
	myAnswer, _ := h.myAnswer(r.Context(), questionID, member.EmployeeID)
	payload["my_answer"] = myAnswer
	response.OK(w, "", payload)
}

// CreateQuestion handles POST /companies/{companyId}/questions.
func (h *Handler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")

	var req createQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		response.Fail(w, http.StatusBadRequest, "title is required", nil)
		return
	}

	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO questions (id, company_id, title) VALUES ($1, $2, $3)`, id, companyID, req.Title)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create question failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "question.created", member.EmployeeID, strPtr("question"), &id, nil)

	payload, err := h.questionPayload(r.Context(), id, companyID, false)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "question payload failed", err.Error())
		return
	}
	response.Created(w, "Question created", payload)
}

// UpdateQuestion handles PATCH /companies/{companyId}/questions/{questionId}.
func (h *Handler) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	questionID := chi.URLParam(r, "questionId")

	var req updateQuestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	var title string
	err := h.pool.QueryRow(r.Context(), `SELECT title FROM questions WHERE id = $1 AND company_id = $2`, questionID, companyID).Scan(&title)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Question not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "question lookup failed", err.Error())
		return
	}
	if req.Title != nil {
		title = strings.TrimSpace(*req.Title)
		if title == "" {
			response.Fail(w, http.StatusBadRequest, "title cannot be empty", nil)
			return
		}
	}

	_, err = h.pool.Exec(r.Context(), `UPDATE questions SET title = $3, updated_at = now() WHERE id = $1 AND company_id = $2`, questionID, companyID, title)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update question failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "question.updated", member.EmployeeID, strPtr("question"), &questionID, nil)

	payload, err := h.questionPayload(r.Context(), questionID, companyID, false)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "question payload failed", err.Error())
		return
	}
	response.OK(w, "Question updated", payload)
}

// DeleteQuestion handles DELETE /companies/{companyId}/questions/{questionId}.
func (h *Handler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	questionID := chi.URLParam(r, "questionId")

	tag, err := h.pool.Exec(r.Context(), `DELETE FROM questions WHERE id = $1 AND company_id = $2`, questionID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete question failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Question not found", nil)
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "question.deleted", member.EmployeeID, strPtr("question"), &questionID, nil)
	response.OK(w, "Question deleted", nil)
}

// ActivateQuestion handles PUT /companies/{companyId}/questions/{questionId}/activate.
// Deactivates any other active question in the company first, so at most one
// question is active per company at a time.
func (h *Handler) ActivateQuestion(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	questionID := chi.URLParam(r, "questionId")

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())

	var exists bool
	_ = tx.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM questions WHERE id = $1 AND company_id = $2)`, questionID, companyID).Scan(&exists)
	if !exists {
		response.Fail(w, http.StatusNotFound, "Question not found", nil)
		return
	}

	_, err = tx.Exec(r.Context(), `
		UPDATE questions SET active = false, deactivated_at = now(), updated_at = now()
		WHERE company_id = $1 AND active = true AND id != $2`, companyID, questionID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "deactivate other questions failed", err.Error())
		return
	}
	_, err = tx.Exec(r.Context(), `
		UPDATE questions SET active = true, activated_at = now(), deactivated_at = NULL, updated_at = now()
		WHERE id = $1 AND company_id = $2`, questionID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "activate question failed", err.Error())
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "question.activated", member.EmployeeID, strPtr("question"), &questionID, nil)

	payload, err := h.questionPayload(r.Context(), questionID, companyID, false)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "question payload failed", err.Error())
		return
	}
	response.OK(w, "Question activated", payload)
}

// DeactivateQuestion handles PUT /companies/{companyId}/questions/{questionId}/deactivate.
func (h *Handler) DeactivateQuestion(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	questionID := chi.URLParam(r, "questionId")

	tag, err := h.pool.Exec(r.Context(), `
		UPDATE questions SET active = false, deactivated_at = now(), updated_at = now()
		WHERE id = $1 AND company_id = $2`, questionID, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "deactivate question failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Question not found", nil)
		return
	}

	_ = audit.LogEmployee(r.Context(), h.pool, companyID, "question.deactivated", member.EmployeeID, strPtr("question"), &questionID, nil)

	payload, err := h.questionPayload(r.Context(), questionID, companyID, false)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "question payload failed", err.Error())
		return
	}
	response.OK(w, "Question deactivated", payload)
}

// CreateAnswer handles POST /companies/{companyId}/questions/{questionId}/answers.
// Upserts on (question_id, employee_id) so re-submitting simply edits the answer.
func (h *Handler) CreateAnswer(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	questionID := chi.URLParam(r, "questionId")

	var exists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM questions WHERE id = $1 AND company_id = $2)`, questionID, companyID).Scan(&exists)
	if !exists {
		response.Fail(w, http.StatusNotFound, "Question not found", nil)
		return
	}

	var req answerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		response.Fail(w, http.StatusBadRequest, "body is required", nil)
		return
	}

	id := uuidv7.New()
	var answerID string
	err := h.pool.QueryRow(r.Context(), `
		INSERT INTO answers (id, company_id, question_id, employee_id, body)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (question_id, employee_id)
		DO UPDATE SET body = EXCLUDED.body, updated_at = now()
		RETURNING id`,
		id, companyID, questionID, member.EmployeeID, req.Body,
	).Scan(&answerID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create answer failed", err.Error())
		return
	}

	payload, err := h.answerPayload(r.Context(), answerID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "answer payload failed", err.Error())
		return
	}
	response.Created(w, "Answer saved", payload)
}

// UpdateAnswer handles PATCH /companies/{companyId}/questions/{questionId}/answers/{answerId}.
func (h *Handler) UpdateAnswer(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	questionID := chi.URLParam(r, "questionId")
	answerID := chi.URLParam(r, "answerId")

	var employeeID, body string
	err := h.pool.QueryRow(r.Context(), `
		SELECT employee_id, body FROM answers
		WHERE id = $1 AND question_id = $2 AND company_id = $3`, answerID, questionID, companyID,
	).Scan(&employeeID, &body)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Answer not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "answer lookup failed", err.Error())
		return
	}
	if employeeID != member.EmployeeID && !member.HasPermission("questions.update") {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	var req answerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	if strings.TrimSpace(req.Body) != "" {
		body = strings.TrimSpace(req.Body)
	}

	_, err = h.pool.Exec(r.Context(), `UPDATE answers SET body = $2, updated_at = now() WHERE id = $1`, answerID, body)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update answer failed", err.Error())
		return
	}

	payload, err := h.answerPayload(r.Context(), answerID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "answer payload failed", err.Error())
		return
	}
	response.OK(w, "Answer updated", payload)
}

// DeleteAnswer handles DELETE /companies/{companyId}/questions/{questionId}/answers/{answerId}.
func (h *Handler) DeleteAnswer(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}
	companyID := chi.URLParam(r, "companyId")
	questionID := chi.URLParam(r, "questionId")
	answerID := chi.URLParam(r, "answerId")

	var employeeID string
	err := h.pool.QueryRow(r.Context(), `
		SELECT employee_id FROM answers WHERE id = $1 AND question_id = $2 AND company_id = $3`,
		answerID, questionID, companyID,
	).Scan(&employeeID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Answer not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "answer lookup failed", err.Error())
		return
	}
	if employeeID != member.EmployeeID && !member.HasPermission("questions.update") {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	_, err = h.pool.Exec(r.Context(), `DELETE FROM answers WHERE id = $1`, answerID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete answer failed", err.Error())
		return
	}
	response.OK(w, "Answer deleted", nil)
}

func (h *Handler) questionPayload(ctx context.Context, questionID, companyID string, withAnswers bool) (map[string]interface{}, error) {
	var id, cid, title string
	var active bool
	var activatedAt, deactivatedAt *time.Time
	var createdAt, updatedAt time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT id, company_id, title, active, activated_at, deactivated_at, created_at, updated_at
		FROM questions WHERE id = $1 AND company_id = $2`, questionID, companyID,
	).Scan(&id, &cid, &title, &active, &activatedAt, &deactivatedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	var answerCount int
	_ = h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM answers WHERE question_id = $1`, id).Scan(&answerCount)

	payload := map[string]interface{}{
		"id":             id,
		"company_id":     cid,
		"title":          title,
		"active":         active,
		"activated_at":   formatTimePtr(activatedAt),
		"deactivated_at": formatTimePtr(deactivatedAt),
		"answer_count":   answerCount,
		"created_at":     createdAt.UTC().Format(time.RFC3339),
		"updated_at":     updatedAt.UTC().Format(time.RFC3339),
	}

	if withAnswers {
		answers, err := h.listAnswers(ctx, id)
		if err != nil {
			return nil, err
		}
		payload["answers"] = answers
	}

	return payload, nil
}

func (h *Handler) listAnswers(ctx context.Context, questionID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT a.id, a.employee_id, e.first_name, e.last_name, a.body, a.created_at, a.updated_at
		FROM answers a
		JOIN employees e ON e.id = a.employee_id
		WHERE a.question_id = $1
		ORDER BY a.created_at`, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []map[string]interface{}{}
	for rows.Next() {
		var id, employeeID, first, last, body string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &employeeID, &first, &last, &body, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		list = append(list, map[string]interface{}{
			"id":            id,
			"employee_id":   employeeID,
			"employee_name": strings.TrimSpace(first + " " + last),
			"body":          body,
			"created_at":    createdAt.UTC().Format(time.RFC3339),
			"updated_at":    updatedAt.UTC().Format(time.RFC3339),
		})
	}
	return list, nil
}

func (h *Handler) myAnswer(ctx context.Context, questionID, employeeID string) (map[string]interface{}, error) {
	var id, body string
	var createdAt, updatedAt time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT id, body, created_at, updated_at FROM answers
		WHERE question_id = $1 AND employee_id = $2`, questionID, employeeID,
	).Scan(&id, &body, &createdAt, &updatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":         id,
		"body":       body,
		"created_at": createdAt.UTC().Format(time.RFC3339),
		"updated_at": updatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (h *Handler) answerPayload(ctx context.Context, answerID string) (map[string]interface{}, error) {
	var id, questionID, employeeID, body string
	var createdAt, updatedAt time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT id, question_id, employee_id, body, created_at, updated_at FROM answers WHERE id = $1`, answerID,
	).Scan(&id, &questionID, &employeeID, &body, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":          id,
		"question_id": questionID,
		"employee_id": employeeID,
		"body":        body,
		"created_at":  createdAt.UTC().Format(time.RFC3339),
		"updated_at":  updatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func formatTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
