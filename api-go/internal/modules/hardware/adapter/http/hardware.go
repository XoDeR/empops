package http

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

func (h *Handler) ListHardware(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "all"
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	sql := `SELECT id, company_id, employee_id, name, serial_number, created_at, updated_at
		FROM hardware WHERE company_id=$1`
	args := []interface{}{member.CompanyID}
	if status == "available" {
		sql += ` AND employee_id IS NULL`
	} else if status == "lent" {
		sql += ` AND employee_id IS NOT NULL`
	}
	if q != "" {
		sql += ` AND (name ILIKE $2 OR COALESCE(serial_number,'') ILIKE $2)`
		args = append(args, "%"+q+"%")
	}
	sql += ` ORDER BY name`

	rows, err := h.pool.Query(r.Context(), sql, args...)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	items := []map[string]interface{}{}
	for rows.Next() {
		item, err := h.scanHardwareRow(r.Context(), rows)
		if err != nil {
			response.Fail(w, 500, "scan failed", err.Error())
			return
		}
		items = append(items, item)
	}
	response.OK(w, "", items)
}

func (h *Handler) CreateHardware(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	var req struct {
		Name         string  `json:"name"`
		SerialNumber *string `json:"serial_number"`
		EmployeeID   *string `json:"employee_id"`
	}
	if err := decodeJSON(r, &req); err != nil || strings.TrimSpace(req.Name) == "" {
		response.Fail(w, 422, "invalid body", nil)
		return
	}
	if req.EmployeeID != nil && !h.employeeInCompany(r.Context(), member.CompanyID, *req.EmployeeID) {
		response.Fail(w, 404, "Employee not found", nil)
		return
	}
	id := uuidv7.New()
	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO hardware (id, company_id, employee_id, name, serial_number)
		VALUES ($1,$2,$3,$4,$5)`, id, member.CompanyID, req.EmployeeID, req.Name, req.SerialNumber)
	if err != nil {
		response.Fail(w, 500, "create failed", err.Error())
		return
	}
	item, err := h.loadHardware(r.Context(), member.CompanyID, id)
	if err != nil {
		response.Fail(w, 500, "load failed", err.Error())
		return
	}
	response.Created(w, "Hardware created", item)
}

func (h *Handler) ShowHardware(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	item, err := h.loadHardware(r.Context(), member.CompanyID, chi.URLParam(r, "hardwareId"))
	if err == pgx.ErrNoRows {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return
	}
	response.OK(w, "", item)
}

func (h *Handler) UpdateHardware(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	hardwareID := chi.URLParam(r, "hardwareId")
	var req map[string]interface{}
	if err := decodeJSON(r, &req); err != nil {
		response.Fail(w, 422, "invalid body", nil)
		return
	}
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM hardware WHERE id=$1 AND company_id=$2)`, hardwareID, member.CompanyID).Scan(&exists)
	if !exists {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if name, ok := req["name"].(string); ok && strings.TrimSpace(name) != "" {
		_, _ = h.pool.Exec(r.Context(), `UPDATE hardware SET name=$1, updated_at=now() WHERE id=$2`, name, hardwareID)
	}
	if _, ok := req["serial_number"]; ok {
		var serial *string
		if req["serial_number"] != nil {
			s, _ := req["serial_number"].(string)
			serial = &s
		}
		_, _ = h.pool.Exec(r.Context(), `UPDATE hardware SET serial_number=$1, updated_at=now() WHERE id=$2`, serial, hardwareID)
	}
	item, err := h.loadHardware(r.Context(), member.CompanyID, hardwareID)
	if err != nil {
		response.Fail(w, 500, "load failed", err.Error())
		return
	}
	response.OK(w, "Hardware updated", item)
}

func (h *Handler) DestroyHardware(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	tag, err := h.pool.Exec(r.Context(), `DELETE FROM hardware WHERE id=$1 AND company_id=$2`,
		chi.URLParam(r, "hardwareId"), member.CompanyID)
	if err != nil {
		response.Fail(w, 500, "delete failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	response.OK(w, "Hardware deleted", nil)
}

func (h *Handler) LendHardware(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	hardwareID := chi.URLParam(r, "hardwareId")
	var req struct {
		EmployeeID string `json:"employee_id"`
	}
	if err := decodeJSON(r, &req); err != nil || req.EmployeeID == "" {
		response.Fail(w, 422, "invalid body", nil)
		return
	}
	if !h.employeeInCompany(r.Context(), member.CompanyID, req.EmployeeID) {
		response.Fail(w, 404, "Employee not found", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `
		UPDATE hardware SET employee_id=$1, updated_at=now() WHERE id=$2 AND company_id=$3`,
		req.EmployeeID, hardwareID, member.CompanyID)
	if err != nil {
		response.Fail(w, 500, "lend failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	item, _ := h.loadHardware(r.Context(), member.CompanyID, hardwareID)
	response.OK(w, "Hardware lent", item)
}

func (h *Handler) RegainHardware(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	hardwareID := chi.URLParam(r, "hardwareId")
	tag, err := h.pool.Exec(r.Context(), `
		UPDATE hardware SET employee_id=NULL, updated_at=now() WHERE id=$1 AND company_id=$2`,
		hardwareID, member.CompanyID)
	if err != nil {
		response.Fail(w, 500, "regain failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	item, _ := h.loadHardware(r.Context(), member.CompanyID, hardwareID)
	response.OK(w, "Hardware regained", item)
}

func (h *Handler) EmployeeHardware(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	employeeID := chi.URLParam(r, "employeeId")
	if !h.canViewEmployeeAssets(member, employeeID, "hardware.view") {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, company_id, employee_id, name, serial_number, created_at, updated_at
		FROM hardware WHERE company_id=$1 AND employee_id=$2 ORDER BY name`,
		member.CompanyID, employeeID)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	items := []map[string]interface{}{}
	for rows.Next() {
		item, err := h.scanHardwareRow(r.Context(), rows)
		if err != nil {
			response.Fail(w, 500, "scan failed", err.Error())
			return
		}
		items = append(items, item)
	}
	response.OK(w, "", items)
}
