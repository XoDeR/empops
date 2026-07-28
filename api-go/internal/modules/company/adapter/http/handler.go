// Package http contains the company module's Chi handlers.
package http

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/httpauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

const defaultCurrency = "EUR"

const joinCodeAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Handler serves the company module's HTTP routes.
type Handler struct {
	pool *pgxpool.Pool
}

// NewHandler builds a company Handler backed by pool.
func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

type companyRow struct {
	ID                string
	Name              string
	Slug              string
	Currency          string
	CodeToJoinCompany string
}

type employeeRow struct {
	ID               string
	CompanyID        string
	UserID           *string
	Email            string
	FirstName        string
	LastName         string
	HiredAt          *time.Time
	PositionID       *string
	EmployeeStatusID *string
	InvitationLink   *string
	InvitationUsedAt *time.Time
	Locked           bool
}

type createCompanyRequest struct {
	Name     string  `json:"name"`
	Currency *string `json:"currency"`
}

type joinCompanyRequest struct {
	Code string `json:"code"`
}

type updateCompanyRequest struct {
	Name     *string `json:"name"`
	Currency *string `json:"currency"`
}

type db interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

// ListCompanies handles GET /companies.
func (h *Handler) ListCompanies(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpauth.UserIDFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusUnauthorized, "Unauthenticated", nil)
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT c.id, c.name, c.slug, c.currency, c.code_to_join_company, e.id
		FROM employees e
		JOIN companies c ON c.id = e.company_id
		WHERE e.user_id = $1 AND e.locked = false
		ORDER BY c.name`, userID)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "list companies failed", err.Error())
		return
	}
	defer rows.Close()

	var list []map[string]interface{}
	for rows.Next() {
		var c companyRow
		var employeeID string
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Currency, &c.CodeToJoinCompany, &employeeID); err != nil {
			response.Fail(w, http.StatusInternalServerError, "scan failed", err.Error())
			return
		}
		roles, err := listEmployeeRoles(r.Context(), h.pool, employeeID)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "roles lookup failed", err.Error())
			return
		}
		item := companyPayload(c, false)
		item["employee_id"] = employeeID
		item["roles"] = roles
		list = append(list, item)
	}
	if list == nil {
		list = []map[string]interface{}{}
	}

	response.OK(w, "", list)
}

// CreateCompany handles POST /companies.
func (h *Handler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpauth.UserIDFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusUnauthorized, "Unauthenticated", nil)
		return
	}

	var req createCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		response.Fail(w, http.StatusBadRequest, "name is required", nil)
		return
	}

	currency := defaultCurrency
	if req.Currency != nil {
		c := strings.TrimSpace(*req.Currency)
		if len(c) != 3 {
			response.Fail(w, http.StatusBadRequest, "currency must be 3 characters", nil)
			return
		}
		currency = strings.ToUpper(c)
	}

	var userEmail, userName string
	err := h.pool.QueryRow(r.Context(), `SELECT email, name FROM users WHERE id = $1`, userID).
		Scan(&userEmail, &userName)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "User not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "user lookup failed", err.Error())
		return
	}

	firstName, lastName := splitName(userName)
	today := time.Now().UTC().Truncate(24 * time.Hour)

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "transaction failed", err.Error())
		return
	}
	defer tx.Rollback(r.Context())

	companyID := uuidv7.New()
	slug, err := uniqueSlug(r.Context(), tx, req.Name, "")
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "slug generation failed", err.Error())
		return
	}
	joinCode, err := uniqueJoinCode(r.Context(), tx)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "join code generation failed", err.Error())
		return
	}

	var company companyRow
	err = tx.QueryRow(r.Context(), `
		INSERT INTO companies (id, name, slug, currency, code_to_join_company, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, now(), now())
		RETURNING id, name, slug, currency, code_to_join_company`,
		companyID, req.Name, slug, currency, joinCode,
	).Scan(&company.ID, &company.Name, &company.Slug, &company.Currency, &company.CodeToJoinCompany)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create company failed", err.Error())
		return
	}

	employeeID := uuidv7.New()
	var employee employeeRow
	err = tx.QueryRow(r.Context(), `
		INSERT INTO employees (
			id, company_id, user_id, email, first_name, last_name, hired_at, locked, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, false, now(), now())
		RETURNING id, company_id, user_id, email, first_name, last_name, hired_at,
			position_id, employee_status_id, invitation_link, invitation_used_at, locked`,
		employeeID, company.ID, userID, userEmail, firstName, lastName, today,
	).Scan(
		&employee.ID, &employee.CompanyID, &employee.UserID, &employee.Email,
		&employee.FirstName, &employee.LastName, &employee.HiredAt,
		&employee.PositionID, &employee.EmployeeStatusID, &employee.InvitationLink,
		&employee.InvitationUsedAt, &employee.Locked,
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create employee failed", err.Error())
		return
	}

	if err := assignRole(r.Context(), tx, employee.ID, "administrator"); err != nil {
		response.Fail(w, http.StatusInternalServerError, "assign role failed", err.Error())
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		response.Fail(w, http.StatusInternalServerError, "commit failed", err.Error())
		return
	}

	payload, err := employeePayload(r.Context(), h.pool, employee.ID, employee.CompanyID, false)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "employee payload failed", err.Error())
		return
	}

	response.Created(w, "Company created", map[string]interface{}{
		"company":  companyPayload(company, true),
		"employee": payload,
	})
}

// JoinCompany handles POST /companies/join.
func (h *Handler) JoinCompany(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpauth.UserIDFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusUnauthorized, "Unauthenticated", nil)
		return
	}

	var req joinCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	if req.Code == "" {
		response.Fail(w, http.StatusBadRequest, "code is required", nil)
		return
	}

	var company companyRow
	err := h.pool.QueryRow(r.Context(), `
		SELECT id, name, slug, currency, code_to_join_company
		FROM companies WHERE code_to_join_company = $1`, req.Code,
	).Scan(&company.ID, &company.Name, &company.Slug, &company.Currency, &company.CodeToJoinCompany)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Invalid join code", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "company lookup failed", err.Error())
		return
	}

	var existingID string
	err = h.pool.QueryRow(r.Context(), `
		SELECT id FROM employees WHERE company_id = $1 AND user_id = $2`,
		company.ID, userID,
	).Scan(&existingID)
	if err == nil {
		response.Fail(w, http.StatusConflict, "Already a member of this company", nil)
		return
	}
	if err != pgx.ErrNoRows {
		response.Fail(w, http.StatusInternalServerError, "membership check failed", err.Error())
		return
	}

	var userEmail, userName string
	err = h.pool.QueryRow(r.Context(), `SELECT email, name FROM users WHERE id = $1`, userID).
		Scan(&userEmail, &userName)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "user lookup failed", err.Error())
		return
	}

	var unclaimedID string
	err = h.pool.QueryRow(r.Context(), `
		SELECT id FROM employees
		WHERE company_id = $1 AND email = $2 AND user_id IS NULL`,
		company.ID, userEmail,
	).Scan(&unclaimedID)
	if err == nil {
		response.Fail(w, http.StatusConflict, "An invitation exists for this email; accept the invite instead", nil)
		return
	}
	if err != pgx.ErrNoRows {
		response.Fail(w, http.StatusInternalServerError, "invitation check failed", err.Error())
		return
	}

	firstName, lastName := splitName(userName)
	today := time.Now().UTC().Truncate(24 * time.Hour)

	employeeID := uuidv7.New()
	var employee employeeRow
	err = h.pool.QueryRow(r.Context(), `
		INSERT INTO employees (
			id, company_id, user_id, email, first_name, last_name, hired_at, locked, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, false, now(), now())
		RETURNING id, company_id, user_id, email, first_name, last_name, hired_at,
			position_id, employee_status_id, invitation_link, invitation_used_at, locked`,
		employeeID, company.ID, userID, userEmail, firstName, lastName, today,
	).Scan(
		&employee.ID, &employee.CompanyID, &employee.UserID, &employee.Email,
		&employee.FirstName, &employee.LastName, &employee.HiredAt,
		&employee.PositionID, &employee.EmployeeStatusID, &employee.InvitationLink,
		&employee.InvitationUsedAt, &employee.Locked,
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "create employee failed", err.Error())
		return
	}

	if err := assignRole(r.Context(), h.pool, employee.ID, "employee"); err != nil {
		response.Fail(w, http.StatusInternalServerError, "assign role failed", err.Error())
		return
	}

	payload, err := employeePayload(r.Context(), h.pool, employee.ID, employee.CompanyID, false)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "employee payload failed", err.Error())
		return
	}

	response.OK(w, "Joined company", map[string]interface{}{
		"company":  companyPayload(company, false),
		"employee": payload,
	})
}

// AcceptInvitation handles POST /invitations/{link}/accept.
func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpauth.UserIDFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusUnauthorized, "Unauthenticated", nil)
		return
	}

	link := chi.URLParam(r, "link")
	if link == "" {
		response.Fail(w, http.StatusBadRequest, "link required", nil)
		return
	}

	var employee employeeRow
	err := h.pool.QueryRow(r.Context(), `
		SELECT id, company_id, user_id, email, first_name, last_name, hired_at,
			position_id, employee_status_id, invitation_link, invitation_used_at, locked
		FROM employees WHERE invitation_link = $1`, link,
	).Scan(
		&employee.ID, &employee.CompanyID, &employee.UserID, &employee.Email,
		&employee.FirstName, &employee.LastName, &employee.HiredAt,
		&employee.PositionID, &employee.EmployeeStatusID, &employee.InvitationLink,
		&employee.InvitationUsedAt, &employee.Locked,
	)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Invitation not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "invitation lookup failed", err.Error())
		return
	}

	if employee.InvitationUsedAt != nil || employee.UserID != nil {
		response.Fail(w, http.StatusConflict, "Invitation already used", nil)
		return
	}
	if employee.Locked {
		response.Fail(w, http.StatusForbidden, "Employee is locked", nil)
		return
	}

	var otherID string
	err = h.pool.QueryRow(r.Context(), `
		SELECT id FROM employees
		WHERE company_id = $1 AND user_id = $2 AND id != $3`,
		employee.CompanyID, userID, employee.ID,
	).Scan(&otherID)
	if err == nil {
		response.Fail(w, http.StatusConflict, "Already a member of this company", nil)
		return
	}
	if err != pgx.ErrNoRows {
		response.Fail(w, http.StatusInternalServerError, "membership check failed", err.Error())
		return
	}

	err = h.pool.QueryRow(r.Context(), `
		UPDATE employees SET user_id = $2, invitation_used_at = now(), updated_at = now()
		WHERE id = $1
		RETURNING id, company_id, user_id, email, first_name, last_name, hired_at,
			position_id, employee_status_id, invitation_link, invitation_used_at, locked`,
		employee.ID, userID,
	).Scan(
		&employee.ID, &employee.CompanyID, &employee.UserID, &employee.Email,
		&employee.FirstName, &employee.LastName, &employee.HiredAt,
		&employee.PositionID, &employee.EmployeeStatusID, &employee.InvitationLink,
		&employee.InvitationUsedAt, &employee.Locked,
	)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "claim invitation failed", err.Error())
		return
	}

	var company companyRow
	err = h.pool.QueryRow(r.Context(), `
		SELECT id, name, slug, currency, code_to_join_company FROM companies WHERE id = $1`,
		employee.CompanyID,
	).Scan(&company.ID, &company.Name, &company.Slug, &company.Currency, &company.CodeToJoinCompany)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "company lookup failed", err.Error())
		return
	}

	payload, err := employeePayload(r.Context(), h.pool, employee.ID, employee.CompanyID, false)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "employee payload failed", err.Error())
		return
	}

	response.OK(w, "Invitation accepted", map[string]interface{}{
		"company":  companyPayload(company, false),
		"employee": payload,
	})
}

// ShowCompany handles GET /companies/{companyId}.
func (h *Handler) ShowCompany(w http.ResponseWriter, r *http.Request) {
	member, ok := companyauth.MemberFromContext(r.Context())
	if !ok {
		response.Fail(w, http.StatusForbidden, "Company membership required", nil)
		return
	}

	companyID := chi.URLParam(r, "companyId")
	var company companyRow
	err := h.pool.QueryRow(r.Context(), `
		SELECT id, name, slug, currency, code_to_join_company FROM companies WHERE id = $1`, companyID,
	).Scan(&company.ID, &company.Name, &company.Slug, &company.Currency, &company.CodeToJoinCompany)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Company not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "company lookup failed", err.Error())
		return
	}

	includeJoinCode := member.HasPermission("company.update")
	data := companyPayload(company, includeJoinCode)
	data["employee_id"] = member.EmployeeID
	data["roles"] = member.Roles

	response.OK(w, "", data)
}

// UpdateCompany handles PATCH /companies/{companyId}.
func (h *Handler) UpdateCompany(w http.ResponseWriter, r *http.Request) {
	companyID := chi.URLParam(r, "companyId")

	var req updateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, http.StatusBadRequest, "Invalid JSON body", nil)
		return
	}

	var company companyRow
	err := h.pool.QueryRow(r.Context(), `
		SELECT id, name, slug, currency, code_to_join_company FROM companies WHERE id = $1`, companyID,
	).Scan(&company.ID, &company.Name, &company.Slug, &company.Currency, &company.CodeToJoinCompany)
	if err == pgx.ErrNoRows {
		response.Fail(w, http.StatusNotFound, "Company not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "company lookup failed", err.Error())
		return
	}

	name := company.Name
	slug := company.Slug
	currency := company.Currency

	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
		if name == "" {
			response.Fail(w, http.StatusBadRequest, "name cannot be empty", nil)
			return
		}
		slug, err = uniqueSlug(r.Context(), h.pool, name, company.ID)
		if err != nil {
			response.Fail(w, http.StatusInternalServerError, "slug generation failed", err.Error())
			return
		}
	}
	if req.Currency != nil {
		c := strings.TrimSpace(*req.Currency)
		if len(c) != 3 {
			response.Fail(w, http.StatusBadRequest, "currency must be 3 characters", nil)
			return
		}
		currency = strings.ToUpper(c)
	}

	err = h.pool.QueryRow(r.Context(), `
		UPDATE companies SET name = $2, slug = $3, currency = $4, updated_at = now()
		WHERE id = $1
		RETURNING id, name, slug, currency, code_to_join_company`,
		company.ID, name, slug, currency,
	).Scan(&company.ID, &company.Name, &company.Slug, &company.Currency, &company.CodeToJoinCompany)
	if err != nil {
		response.Fail(w, http.StatusInternalServerError, "update company failed", err.Error())
		return
	}

	response.OK(w, "Company updated", companyPayload(company, true))
}

func companyPayload(c companyRow, includeJoinCode bool) map[string]interface{} {
	payload := map[string]interface{}{
		"id":       c.ID,
		"name":     c.Name,
		"slug":     c.Slug,
		"currency": c.Currency,
	}
	if includeJoinCode {
		payload["code_to_join_company"] = c.CodeToJoinCompany
	}
	return payload
}

func uniqueSlug(ctx context.Context, q db, name, exceptID string) (string, error) {
	base := slugify(name)
	slug := base
	for i := 1; ; i++ {
		var exists bool
		var err error
		if exceptID == "" {
			err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM companies WHERE slug = $1)`, slug).Scan(&exists)
		} else {
			err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM companies WHERE slug = $1 AND id != $2)`, slug, exceptID).Scan(&exists)
		}
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		slug = base + "-" + itoa(i)
	}
}

func uniqueJoinCode(ctx context.Context, q db) (string, error) {
	for {
		code, err := randomJoinCode()
		if err != nil {
			return "", err
		}
		var exists bool
		if err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM companies WHERE code_to_join_company = $1)`, code).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
}

func slugify(name string) string {
	s := strings.TrimSpace(strings.ToLower(name))
	var b strings.Builder
	prevHyphen := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevHyphen = false
			continue
		}
		if !prevHyphen && b.Len() > 0 {
			b.WriteByte('-')
			prevHyphen = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "company"
	}
	return out
}

func randomJoinCode() (string, error) {
	b := make([]byte, 8)
	max := big.NewInt(int64(len(joinCodeAlphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = joinCodeAlphabet[n.Int64()]
	}
	return string(b), nil
}

func splitName(name string) (firstName, lastName string) {
	parts := strings.Fields(strings.TrimSpace(name))
	if len(parts) == 0 {
		return "User", ""
	}
	firstName = parts[0]
	if len(parts) > 1 {
		lastName = parts[1]
	}
	return firstName, lastName
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func assignRole(ctx context.Context, q db, employeeID, roleName string) error {
	var roleID string
	if err := q.QueryRow(ctx, `SELECT id FROM roles WHERE name = $1`, roleName).Scan(&roleID); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `
		INSERT INTO employee_roles (employee_id, role_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, employeeID, roleID)
	return err
}

func listEmployeeRoles(ctx context.Context, pool *pgxpool.Pool, employeeID string) ([]string, error) {
	rows, err := pool.Query(ctx, `
		SELECT r.name FROM employee_roles er
		JOIN roles r ON r.id = er.role_id
		WHERE er.employee_id = $1 ORDER BY r.name`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if roles == nil {
		roles = []string{}
	}
	return roles, nil
}

func employeePayload(ctx context.Context, pool *pgxpool.Pool, employeeID, companyID string, includeInvite bool) (map[string]interface{}, error) {
	var (
		id, cid, email, firstName, lastName string
		userID, inviteLink                    *string
		hiredAt, inviteUsedAt                 *time.Time
		locked                                bool
		posID, posTitle                       *string
		stID, stName, stType                  *string
	)

	err := pool.QueryRow(ctx, `
		SELECT e.id, e.company_id, e.user_id, e.email, e.first_name, e.last_name, e.hired_at,
			e.locked, e.invitation_link, e.invitation_used_at,
			p.id, p.title,
			s.id, s.name, s.type
		FROM employees e
		LEFT JOIN positions p ON p.id = e.position_id
		LEFT JOIN employee_statuses s ON s.id = e.employee_status_id
		WHERE e.id = $1 AND e.company_id = $2`,
		employeeID, companyID,
	).Scan(
		&id, &cid, &userID, &email, &firstName, &lastName, &hiredAt,
		&locked, &inviteLink, &inviteUsedAt,
		&posID, &posTitle,
		&stID, &stName, &stType,
	)
	if err != nil {
		return nil, err
	}

	roles, err := listEmployeeRoles(ctx, pool, id)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"id":         id,
		"company_id": cid,
		"user_id":    userID,
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
		"hired_at":   formatDate(hiredAt),
		"locked":     locked,
		"roles":      roles,
	}

	if posID != nil && posTitle != nil {
		payload["position"] = map[string]interface{}{
			"id":    *posID,
			"title": *posTitle,
		}
	} else {
		payload["position"] = nil
	}

	if stID != nil && stName != nil && stType != nil {
		payload["status"] = map[string]interface{}{
			"id":   *stID,
			"name": *stName,
			"type": *stType,
		}
	} else {
		payload["status"] = nil
	}

	if includeInvite {
		payload["invitation_link"] = inviteLink
		if inviteLink != nil {
			payload["invitation_url"] = "/invitations/" + *inviteLink + "/accept"
		} else {
			payload["invitation_url"] = nil
		}
	}

	return payload, nil
}

func formatDate(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}
