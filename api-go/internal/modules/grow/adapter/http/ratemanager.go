package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
)

func (h *Handler) buildRateAnswer(ctx context.Context, answerID string) (map[string]interface{}, error) {
	var surveyID, employeeID string
	var active bool
	var rating, comment *string
	var reveal bool
	var surveyActive bool
	var managerID *string
	var validUntil *time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT a.rate_your_manager_survey_id, a.employee_id, a.active, a.rating, a.comment,
			a.reveal_identity_to_manager, s.active, s.manager_id, s.valid_until_at
		FROM rate_your_manager_answers a
		JOIN rate_your_manager_surveys s ON s.id=a.rate_your_manager_survey_id
		WHERE a.id=$1`, answerID,
	).Scan(&surveyID, &employeeID, &active, &rating, &comment, &reveal, &surveyActive, &managerID, &validUntil)
	if err != nil {
		return nil, err
	}
	var manager interface{}
	if managerID != nil {
		manager = h.employeeSummary(ctx, *managerID)
	}
	return map[string]interface{}{
		"id":                         answerID,
		"survey_id":                  surveyID,
		"employee":                   h.employeeSummary(ctx, employeeID),
		"manager":                    manager,
		"active":                     active,
		"rating":                     rating,
		"comment":                    comment,
		"reveal_identity_to_manager": reveal,
		"valid_until_at":             isoTime(validUntil),
		"survey_active":              surveyActive,
	}, nil
}

func (h *Handler) PendingRateAnswers(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	rows, err := h.pool.Query(r.Context(), `
		SELECT a.id FROM rate_your_manager_answers a
		JOIN rate_your_manager_surveys s ON s.id=a.rate_your_manager_survey_id
		WHERE a.employee_id=$1 AND a.active=true AND s.active=true`, member.EmployeeID)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			if p, err := h.buildRateAnswer(r.Context(), id); err == nil {
				out = append(out, p)
			}
		}
	}
	response.OK(w, "", out)
}

func (h *Handler) SubmitRating(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	answerID := chi.URLParam(r, "answerId")
	var body struct {
		Rating string `json:"rating"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.Fail(w, 422, "invalid body", nil)
		return
	}
	if body.Rating != "bad" && body.Rating != "average" && body.Rating != "good" {
		response.Fail(w, 422, "Rating must be bad, average, or good", nil)
		return
	}
	var employeeID string
	var answerActive, surveyActive bool
	err := h.pool.QueryRow(r.Context(), `
		SELECT a.employee_id, a.active, s.active
		FROM rate_your_manager_answers a
		JOIN rate_your_manager_surveys s ON s.id=a.rate_your_manager_survey_id
		WHERE a.id=$1 AND s.company_id=$2`, answerID, member.CompanyID,
	).Scan(&employeeID, &answerActive, &surveyActive)
	if err == pgx.ErrNoRows {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return
	}
	if employeeID != member.EmployeeID {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	if !answerActive || !surveyActive {
		response.Fail(w, 409, "Survey is not active", nil)
		return
	}
	_, err = h.pool.Exec(r.Context(), `
		UPDATE rate_your_manager_answers SET rating=$2, active=false, updated_at=now() WHERE id=$1`,
		answerID, body.Rating)
	if err != nil {
		response.Fail(w, 500, "update failed", err.Error())
		return
	}
	p, _ := h.buildRateAnswer(r.Context(), answerID)
	response.OK(w, "", p)
}

func (h *Handler) CommentOnRating(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	answerID := chi.URLParam(r, "answerId")
	var body struct {
		Comment                   *string `json:"comment"`
		RevealIdentityToManager   bool    `json:"reveal_identity_to_manager"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.Fail(w, 422, "invalid body", nil)
		return
	}
	var employeeID string
	var surveyActive bool
	err := h.pool.QueryRow(r.Context(), `
		SELECT a.employee_id, s.active
		FROM rate_your_manager_answers a
		JOIN rate_your_manager_surveys s ON s.id=a.rate_your_manager_survey_id
		WHERE a.id=$1 AND s.company_id=$2`, answerID, member.CompanyID,
	).Scan(&employeeID, &surveyActive)
	if err == pgx.ErrNoRows {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if employeeID != member.EmployeeID {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	if !surveyActive {
		response.Fail(w, 409, "Survey is not active", nil)
		return
	}
	_, err = h.pool.Exec(r.Context(), `
		UPDATE rate_your_manager_answers
		SET comment=$2, reveal_identity_to_manager=$3, updated_at=now() WHERE id=$1`,
		answerID, body.Comment, body.RevealIdentityToManager)
	if err != nil {
		response.Fail(w, 500, "update failed", err.Error())
		return
	}
	p, _ := h.buildRateAnswer(r.Context(), answerID)
	response.OK(w, "", p)
}

func (h *Handler) ManagerSurveys(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	managerID := chi.URLParam(r, "employeeId")
	if managerID != member.EmployeeID && !h.isHr(member) {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, active, valid_until_at, created_at FROM rate_your_manager_surveys
		WHERE company_id=$1 AND manager_id=$2 ORDER BY created_at DESC`,
		member.CompanyID, managerID)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id string
		var active bool
		var validUntil *time.Time
		var createdAt time.Time
		if rows.Scan(&id, &active, &validUntil, &createdAt) != nil {
			continue
		}
		answers := []map[string]interface{}{}
		arows, _ := h.pool.Query(r.Context(), `
			SELECT id, employee_id, rating, comment, reveal_identity_to_manager
			FROM rate_your_manager_answers WHERE rate_your_manager_survey_id=$1`, id)
		if arows != nil {
			for arows.Next() {
				var aid, eid string
				var rating, comment *string
				var reveal bool
				if arows.Scan(&aid, &eid, &rating, &comment, &reveal) == nil {
					var emp interface{}
					if managerID != member.EmployeeID || reveal || h.isHr(member) {
						emp = h.employeeSummary(r.Context(), eid)
					}
					answers = append(answers, map[string]interface{}{
						"id": aid, "rating": rating, "comment": comment,
						"reveal_identity_to_manager": reveal, "employee": emp,
					})
				}
			}
			arows.Close()
		}
		out = append(out, map[string]interface{}{
			"id": id, "active": active, "valid_until_at": isoTime(validUntil),
			"created_at": createdAt.UTC().Format(time.RFC3339), "answers": answers,
		})
	}
	response.OK(w, "", out)
}
