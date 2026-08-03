package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

type processStageRequest struct {
	Accepted bool `json:"accepted"`
}

type noteRequest struct {
	Note string `json:"note"`
}

type participantRequest struct {
	EmployeeID string `json:"employee_id"`
}

type hireRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	HiredAt   string `json:"hired_at"`
}

type attachFileRequest struct {
	TemporaryUploadID int64 `json:"temporary_upload_id"`
	MediaID           int64 `json:"media_id"`
}

func (h *Handler) ListCandidates(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")

	if !h.openingBelongsToCompany(r.Context(), companyID, openingID) {
		response.Fail(w, http.StatusNotFound, "Job opening not found", nil)
		return
	}
	if _, ok := h.requireAccess(w, r, openingID); !ok {
		return
	}

	bucket := r.URL.Query().Get("bucket")
	if bucket == "" {
		bucket = "to_sort"
	}

	q := `
		SELECT c.id FROM candidates c
		WHERE c.job_opening_id = $1 AND c.application_completed = true`
	args := []interface{}{openingID}

	switch bucket {
	case "rejected":
		q += ` AND c.rejected = true`
	case "selected":
		q += ` AND c.rejected = false AND EXISTS(
			SELECT 1 FROM candidate_stages cs
			WHERE cs.candidate_id = c.id AND cs.status != 'pending')`
	default:
		q += ` AND c.rejected = false AND NOT EXISTS(
			SELECT 1 FROM candidate_stages cs
			WHERE cs.candidate_id = c.id AND cs.status != 'pending')`
	}
	q += ` ORDER BY c.created_at DESC`

	rows, err := h.pool.Query(r.Context(), q, args...)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list candidates failed", err.Error())
		return
	}
	defer rows.Close()

	out := []map[string]interface{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		payload, err := h.candidatePayload(r.Context(), id, false)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "candidate payload failed", err.Error())
			return
		}
		out = append(out, payload)
	}
	response.OK(w, "", out)
}

func (h *Handler) ShowCandidate(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")
	candidateID := chi.URLParam(r, "candidateId")

	if !h.findOpeningIDForCandidate(r.Context(), companyID, openingID, candidateID) {
		response.Fail(w, http.StatusNotFound, "Candidate not found", nil)
		return
	}
	if _, ok := h.requireAccess(w, r, openingID); !ok {
		return
	}

	payload, err := h.candidatePayload(r.Context(), candidateID, true)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "candidate payload failed", err.Error())
		return
	}
	response.OK(w, "", payload)
}

func (h *Handler) ProcessStage(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")
	candidateID := chi.URLParam(r, "candidateId")
	stageID := chi.URLParam(r, "stageId")

	if !h.findOpeningIDForCandidate(r.Context(), companyID, openingID, candidateID) {
		response.Fail(w, http.StatusNotFound, "Candidate not found", nil)
		return
	}
	member, ok := h.requireAccess(w, r, openingID)
	if !ok {
		return
	}

	var req processStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	var rejected bool
	var status string
	err := h.pool.QueryRow(r.Context(), `
		SELECT c.rejected, cs.status FROM candidates c
		JOIN candidate_stages cs ON cs.candidate_id = c.id
		WHERE c.id = $1 AND cs.id = $2 AND c.job_opening_id = $3`,
		candidateID, stageID, openingID,
	).Scan(&rejected, &status)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Stage not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "stage lookup failed", err.Error())
		return
	}
	if rejected {
		response.Fail(w, http.StatusConflict, "Candidate already rejected", nil)
		return
	}
	if status != "pending" {
		response.Fail(w, http.StatusConflict, "Stage already processed", nil)
		return
	}

	name := deciderName(r.Context(), h.pool, member.EmployeeID)
	newStatus := "rejected"
	if req.Accepted {
		newStatus = "passed"
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())

	now := time.Now()
	_, err = tx.Exec(r.Context(), `
		UPDATE candidate_stages SET status = $1, decider_id = $2, decider_name = $3, decided_at = $4, updated_at = now()
		WHERE id = $5`, newStatus, member.EmployeeID, name, now, stageID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "process stage failed", err.Error())
		return
	}

	if !req.Accepted {
		_, err = tx.Exec(r.Context(), `UPDATE candidates SET rejected = true, updated_at = now() WHERE id = $1`, candidateID)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "reject candidate failed", err.Error())
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}

	payload, err := h.candidatePayload(r.Context(), candidateID, true)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "candidate payload failed", err.Error())
		return
	}
	response.OK(w, "Stage processed", payload)
}

func (h *Handler) ListNotes(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")
	candidateID := chi.URLParam(r, "candidateId")
	stageID := chi.URLParam(r, "stageId")

	if !h.stageBelongsToCandidate(r.Context(), companyID, openingID, candidateID, stageID) {
		response.Fail(w, http.StatusNotFound, "Stage not found", nil)
		return
	}
	if _, ok := h.requireAccess(w, r, openingID); !ok {
		return
	}

	notes, err := h.listNotes(r.Context(), stageID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list notes failed", err.Error())
		return
	}
	response.OK(w, "", notes)
}

func (h *Handler) CreateNote(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")
	candidateID := chi.URLParam(r, "candidateId")
	stageID := chi.URLParam(r, "stageId")

	if !h.stageBelongsToCandidate(r.Context(), companyID, openingID, candidateID, stageID) {
		response.Fail(w, http.StatusNotFound, "Stage not found", nil)
		return
	}
	member, ok := h.requireAccess(w, r, openingID)
	if !ok {
		return
	}

	var req noteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Note = strings.TrimSpace(req.Note)
	if req.Note == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "note is required", nil)
		return
	}

	noteID := uuidv7.New()
	authorName := deciderName(r.Context(), h.pool, member.EmployeeID)
	var createdAt time.Time
	err := h.pool.QueryRow(r.Context(), `
		INSERT INTO candidate_stage_notes (id, candidate_stage_id, author_id, author_name, note, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, now(), now()) RETURNING created_at`,
		noteID, stageID, member.EmployeeID, authorName, req.Note,
	).Scan(&createdAt)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create note failed", err.Error())
		return
	}

	response.Created(w, "Note created", notePayload(noteID, member.EmployeeID, authorName, req.Note, createdAt))
}

func (h *Handler) UpdateNote(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")
	candidateID := chi.URLParam(r, "candidateId")
	stageID := chi.URLParam(r, "stageId")
	noteID := chi.URLParam(r, "noteId")

	if !h.stageBelongsToCandidate(r.Context(), companyID, openingID, candidateID, stageID) {
		response.Fail(w, http.StatusNotFound, "Stage not found", nil)
		return
	}
	if _, ok := h.requireAccess(w, r, openingID); !ok {
		return
	}

	var req noteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Note = strings.TrimSpace(req.Note)
	if req.Note == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "note is required", nil)
		return
	}

	var authorID, authorName string
	var createdAt time.Time
	err := h.pool.QueryRow(r.Context(), `
		UPDATE candidate_stage_notes SET note = $1, updated_at = now()
		WHERE id = $2 AND candidate_stage_id = $3
		RETURNING author_id, author_name, created_at`, req.Note, noteID, stageID,
	).Scan(&authorID, &authorName, &createdAt)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Note not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update note failed", err.Error())
		return
	}

	response.OK(w, "Note updated", notePayload(noteID, authorID, authorName, req.Note, createdAt))
}

func (h *Handler) DeleteNote(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")
	candidateID := chi.URLParam(r, "candidateId")
	stageID := chi.URLParam(r, "stageId")
	noteID := chi.URLParam(r, "noteId")

	if !h.stageBelongsToCandidate(r.Context(), companyID, openingID, candidateID, stageID) {
		response.Fail(w, http.StatusNotFound, "Stage not found", nil)
		return
	}
	if _, ok := h.requireAccess(w, r, openingID); !ok {
		return
	}

	tag, err := h.pool.Exec(r.Context(), `
		DELETE FROM candidate_stage_notes WHERE id = $1 AND candidate_stage_id = $2`, noteID, stageID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete note failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Note not found", nil)
		return
	}
	response.OK(w, "Note deleted", nil)
}

func (h *Handler) AddParticipant(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")
	candidateID := chi.URLParam(r, "candidateId")
	stageID := chi.URLParam(r, "stageId")

	if !h.stageBelongsToCandidate(r.Context(), companyID, openingID, candidateID, stageID) {
		response.Fail(w, http.StatusNotFound, "Stage not found", nil)
		return
	}
	if _, ok := h.requireAccess(w, r, openingID); !ok {
		return
	}

	var req participantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.EmployeeID = strings.TrimSpace(req.EmployeeID)
	if req.EmployeeID == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "employee_id is required", nil)
		return
	}

	var first, last string
	err := h.pool.QueryRow(r.Context(), `
		SELECT first_name, last_name FROM employees WHERE id = $1 AND company_id = $2`,
		req.EmployeeID, companyID,
	).Scan(&first, &last)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Employee not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "employee lookup failed", err.Error())
		return
	}

	participantName := strings.TrimSpace(first + " " + last)
	participantID := uuidv7.New()
	_, err = h.pool.Exec(r.Context(), `
		INSERT INTO candidate_stage_participants (
			id, candidate_stage_id, participant_id, participant_name, participated, created_at, updated_at
		) VALUES ($1, $2, $3, $4, false, now(), now())
		ON CONFLICT (candidate_stage_id, participant_id) DO UPDATE SET
			participant_name = EXCLUDED.participant_name, updated_at = now()`,
		participantID, stageID, req.EmployeeID, participantName)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "add participant failed", err.Error())
		return
	}

	var rowID string
	_ = h.pool.QueryRow(r.Context(), `
		SELECT id FROM candidate_stage_participants
		WHERE candidate_stage_id = $1 AND participant_id = $2`, stageID, req.EmployeeID,
	).Scan(&rowID)

	response.Created(w, "Participant added", map[string]interface{}{
		"id":               rowID,
		"participant_id":   req.EmployeeID,
		"participant_name": participantName,
		"participated":     false,
	})
}

func (h *Handler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")
	candidateID := chi.URLParam(r, "candidateId")
	stageID := chi.URLParam(r, "stageId")
	participantID := chi.URLParam(r, "participantId")

	if !h.stageBelongsToCandidate(r.Context(), companyID, openingID, candidateID, stageID) {
		response.Fail(w, http.StatusNotFound, "Stage not found", nil)
		return
	}
	if _, ok := h.requireAccess(w, r, openingID); !ok {
		return
	}

	tag, err := h.pool.Exec(r.Context(), `
		DELETE FROM candidate_stage_participants WHERE id = $1 AND candidate_stage_id = $2`,
		participantID, stageID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "remove participant failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, http.StatusNotFound, "Participant not found", nil)
		return
	}
	response.OK(w, "Participant removed", nil)
}

func (h *Handler) Hire(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")
	candidateID := chi.URLParam(r, "candidateId")

	if !h.findOpeningIDForCandidate(r.Context(), companyID, openingID, candidateID) {
		response.Fail(w, http.StatusNotFound, "Candidate not found", nil)
		return
	}
	member, ok := h.requireAccess(w, r, openingID)
	if !ok {
		return
	}
	if !hasRecruitingPerm(member, "recruiting.hire") && !hasRole(member, "administrator", "hr") {
		response.Fail(w, http.StatusForbidden, "Forbidden", nil)
		return
	}

	var req hireRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.HiredAt = strings.TrimSpace(req.HiredAt)
	if req.Email == "" || req.FirstName == "" || req.LastName == "" || req.HiredAt == "" {
		response.Fail(w, http.StatusUnprocessableEntity, "email, first_name, last_name and hired_at are required", nil)
		return
	}
	hiredAt, err := time.Parse("2006-01-02", req.HiredAt)
	if err != nil {
		response.Fail(w, http.StatusBadRequest, "invalid hired_at", nil)
		return
	}

	var active, fulfilled bool
	var positionID string
	var teamID *string
	var employeeID *string
	err = h.pool.QueryRow(r.Context(), `
		SELECT jo.active, jo.fulfilled, jo.position_id, jo.team_id, c.employee_id
		FROM job_openings jo
		JOIN candidates c ON c.job_opening_id = jo.id
		WHERE jo.id = $1 AND c.id = $2 AND jo.company_id = $3`,
		openingID, candidateID, companyID,
	).Scan(&active, &fulfilled, &positionID, &teamID, &employeeID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Candidate not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "lookup failed", err.Error())
		return
	}
	if !active || fulfilled {
		response.Fail(w, http.StatusConflict, "Opening is not open for hiring", nil)
		return
	}
	if employeeID != nil {
		response.Fail(w, http.StatusConflict, "Candidate already hired", nil)
		return
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())

	newEmployeeID := uuidv7.New()
	err = tx.QueryRow(r.Context(), `
		INSERT INTO employees (
			id, company_id, user_id, email, first_name, last_name, hired_at,
			position_id, employee_status_id, locked,
			amount_of_allowed_holidays, amount_of_sick_days, amount_of_pto_days,
			created_at, updated_at
		) VALUES ($1, $2, NULL, $3, $4, $5, $6, $7, NULL, false,
			(SELECT default_amount_of_allowed_holidays FROM company_pto_policies WHERE company_id=$2 AND year=EXTRACT(YEAR FROM CURRENT_DATE)),
			(SELECT default_amount_of_sick_days FROM company_pto_policies WHERE company_id=$2 AND year=EXTRACT(YEAR FROM CURRENT_DATE)),
			(SELECT default_amount_of_pto_days FROM company_pto_policies WHERE company_id=$2 AND year=EXTRACT(YEAR FROM CURRENT_DATE)),
			now(), now())
		RETURNING id`,
		newEmployeeID, companyID, req.Email, req.FirstName, req.LastName, hiredAt, positionID,
	).Scan(&newEmployeeID)
	if err != nil {
		if isUniqueViolation(err) {
			response.Fail(w, http.StatusConflict, "Email already exists", nil)
			return
		}
		response.Fail(w, http.StatusInternalServerError, "create employee failed", err.Error())
		return
	}

	if err := assignEmployeeRole(r.Context(), tx, newEmployeeID, "employee"); err != nil {
		response.Fail(w, http.StatusInternalServerError, "assign role failed", err.Error())
		return
	}

	if teamID != nil && *teamID != "" {
		_, _ = tx.Exec(r.Context(), `
			INSERT INTO employee_team (employee_id, team_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING`, newEmployeeID, *teamID)
	}

	now := time.Now()
	empName := strings.TrimSpace(req.FirstName + " " + req.LastName)
	_, err = tx.Exec(r.Context(), `
		UPDATE job_openings SET active = false, fulfilled = true, fulfilled_at = $1,
			fulfilled_by_candidate_id = $2, updated_at = now()
		WHERE id = $3`, now, candidateID, openingID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update opening failed", err.Error())
		return
	}

	_, err = tx.Exec(r.Context(), `
		UPDATE candidates SET employee_id = $1, employee_name = $2, updated_at = now()
		WHERE id = $3`, newEmployeeID, empName, candidateID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update candidate failed", err.Error())
		return
	}

	_, err = tx.Exec(r.Context(), `INSERT INTO flow_action_runs(id,company_id,flow_action_id,employee_id,due_on)
		SELECT gen_random_uuid(),f.company_id,a.id,$1,
			$2::date + (CASE WHEN s.modifier='before' THEN -s.real_number_of_days ELSE s.real_number_of_days END)
		FROM flows f JOIN flow_steps s ON s.flow_id=f.id JOIN flow_actions a ON a.step_id=s.id
		WHERE f.company_id=$3 AND f.type='join' ON CONFLICT DO NOTHING`, newEmployeeID, hiredAt, companyID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "schedule join flow failed", err.Error())
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}

	candidatePayload, _ := h.candidatePayload(r.Context(), candidateID, true)
	openingPayload, _ := h.openingPayload(r.Context(), openingID)
	employeePayload, _ := hireEmployeePayload(r.Context(), h.pool, newEmployeeID, companyID)

	response.Created(w, "Candidate hired", map[string]interface{}{
		"candidate": candidatePayload,
		"employee":  employeePayload,
		"opening":   openingPayload,
	})
}

func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")
	candidateID := chi.URLParam(r, "candidateId")

	if !h.findOpeningIDForCandidate(r.Context(), companyID, openingID, candidateID) {
		response.Fail(w, http.StatusNotFound, "Candidate not found", nil)
		return
	}
	if _, ok := h.requireAccess(w, r, openingID); !ok {
		return
	}

	files, err := h.listCandidateFiles(r.Context(), candidateID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list files failed", err.Error())
		return
	}
	response.OK(w, "", files)
}

func (h *Handler) AttachFile(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")
	candidateID := chi.URLParam(r, "candidateId")

	if !h.findOpeningIDForCandidate(r.Context(), companyID, openingID, candidateID) {
		response.Fail(w, http.StatusNotFound, "Candidate not found", nil)
		return
	}
	if _, ok := h.requireAccess(w, r, openingID); !ok {
		return
	}

	var req attachFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	payload, err := h.attachCandidateFile(r.Context(), candidateID, req.TemporaryUploadID, req.MediaID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Media not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "attach file failed", err.Error())
		return
	}
	response.Created(w, "File attached", payload)
}

func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")
	openingID := chi.URLParam(r, "jobOpeningId")
	candidateID := chi.URLParam(r, "candidateId")
	mediaID := chi.URLParam(r, "mediaId")

	if !h.findOpeningIDForCandidate(r.Context(), companyID, openingID, candidateID) {
		response.Fail(w, http.StatusNotFound, "Candidate not found", nil)
		return
	}
	if _, ok := h.requireAccess(w, r, openingID); !ok {
		return
	}

	err := h.deleteCandidateFile(r.Context(), candidateID, mediaID)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "File not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "delete file failed", err.Error())
		return
	}
	response.OK(w, "File deleted", nil)
}

func (h *Handler) stageBelongsToCandidate(ctx context.Context, companyID, openingID, candidateID, stageID string) bool {
	var exists bool
	_ = h.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM candidate_stages cs
			JOIN candidates c ON c.id = cs.candidate_id
			JOIN job_openings jo ON jo.id = c.job_opening_id
			WHERE cs.id = $1 AND c.id = $2 AND c.job_opening_id = $3 AND jo.company_id = $4
		)`, stageID, candidateID, openingID, companyID,
	).Scan(&exists)
	return exists
}

func (h *Handler) candidatePayload(ctx context.Context, candidateID string, detailed bool) (map[string]interface{}, error) {
	var (
		id, jobOpeningID, name, email, candidateUUID string
		url, desiredSalary, notes, employeeName      *string
		employeeID                                   *string
		appCompleted, rejected                       bool
		createdAt                                    time.Time
	)
	err := h.pool.QueryRow(ctx, `
		SELECT id, job_opening_id, name, email, uuid, url, desired_salary, notes,
			application_completed, rejected, employee_id, employee_name, created_at
		FROM candidates WHERE id = $1`, candidateID,
	).Scan(&id, &jobOpeningID, &name, &email, &candidateUUID, &url, &desiredSalary, &notes,
		&appCompleted, &rejected, &employeeID, &employeeName, &createdAt)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"id":                    id,
		"job_opening_id":        jobOpeningID,
		"name":                  name,
		"email":                 email,
		"uuid":                  candidateUUID,
		"url":                   url,
		"desired_salary":        desiredSalary,
		"notes":                 notes,
		"application_completed": appCompleted,
		"rejected":              rejected,
		"employee_id":           employeeID,
		"employee_name":         employeeName,
		"created_at":            createdAt.UTC().Format(time.RFC3339),
	}

	if detailed {
		stages, err := h.listCandidateStages(ctx, candidateID)
		if err != nil {
			return nil, err
		}
		payload["stages"] = stages
		files, err := h.listCandidateFiles(ctx, candidateID)
		if err != nil {
			return nil, err
		}
		payload["files"] = files
	}

	return payload, nil
}

func (h *Handler) listCandidateStages(ctx context.Context, candidateID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT id, stage_name, stage_position, status, decider_id, decider_name, decided_at
		FROM candidate_stages WHERE candidate_id = $1 ORDER BY stage_position`, candidateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []map[string]interface{}{}
	for rows.Next() {
		var id, stageName, status string
		var stagePos int
		var deciderID, deciderName *string
		var decidedAt *time.Time
		if err := rows.Scan(&id, &stageName, &stagePos, &status, &deciderID, &deciderName, &decidedAt); err != nil {
			return nil, err
		}

		notes, err := h.listNotes(ctx, id)
		if err != nil {
			return nil, err
		}
		participants, err := h.listParticipants(ctx, id)
		if err != nil {
			return nil, err
		}

		out = append(out, map[string]interface{}{
			"id":             id,
			"stage_name":     stageName,
			"stage_position": stagePos,
			"status":         status,
			"decider_id":     deciderID,
			"decider_name":   deciderName,
			"decided_at":     formatTimePtr(decidedAt),
			"notes":          notes,
			"participants":   participants,
		})
	}
	return out, nil
}

func (h *Handler) listNotes(ctx context.Context, stageID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT id, author_id, author_name, note, created_at
		FROM candidate_stage_notes WHERE candidate_stage_id = $1 ORDER BY created_at`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, authorID, authorName, note string
		var createdAt time.Time
		if err := rows.Scan(&id, &authorID, &authorName, &note, &createdAt); err != nil {
			return nil, err
		}
		out = append(out, notePayload(id, authorID, authorName, note, createdAt))
	}
	return out, nil
}

func (h *Handler) listParticipants(ctx context.Context, stageID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT id, participant_id, participant_name, participated
		FROM candidate_stage_participants WHERE candidate_stage_id = $1`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, participantID, participantName string
		var participated bool
		if err := rows.Scan(&id, &participantID, &participantName, &participated); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"id":               id,
			"participant_id":   participantID,
			"participant_name": participantName,
			"participated":     participated,
		})
	}
	return out, nil
}

func notePayload(id, authorID, authorName, note string, createdAt time.Time) map[string]interface{} {
	return map[string]interface{}{
		"id":          id,
		"author_id":   authorID,
		"author_name": authorName,
		"note":        note,
		"created_at":  createdAt.UTC().Format(time.RFC3339),
	}
}

type hireDB interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

func assignEmployeeRole(ctx context.Context, q hireDB, employeeID, roleName string) error {
	var roleID string
	if err := q.QueryRow(ctx, `SELECT id FROM roles WHERE name = $1`, roleName).Scan(&roleID); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `
		INSERT INTO employee_roles (employee_id, role_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, employeeID, roleID)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func hireEmployeePayload(ctx context.Context, pool *pgxpool.Pool, employeeID, companyID string) (map[string]interface{}, error) {
	var id, cid, email, firstName, lastName string
	var hiredAt *time.Time
	var positionID, positionTitle *string
	err := pool.QueryRow(ctx, `
		SELECT e.id, e.company_id, e.email, e.first_name, e.last_name, e.hired_at,
			p.id, p.title
		FROM employees e
		LEFT JOIN positions p ON p.id = e.position_id
		WHERE e.id = $1 AND e.company_id = $2`, employeeID, companyID,
	).Scan(&id, &cid, &email, &firstName, &lastName, &hiredAt, &positionID, &positionTitle)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"id":         id,
		"company_id": cid,
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
		"hired_at":   formatTimePtr(hiredAt),
	}
	if positionID != nil && positionTitle != nil {
		payload["position"] = map[string]interface{}{"id": *positionID, "title": *positionTitle}
	} else {
		payload["position"] = nil
	}
	return payload, nil
}
