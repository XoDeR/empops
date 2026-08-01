package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/XoDeR/empops/api-go/pkg/companyauth"
	"github.com/XoDeR/empops/api-go/pkg/response"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

func (h *Handler) seatsUsed(ctx context.Context, softwareID string) (int, error) {
	var n int
	err := h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM employee_software WHERE software_id=$1`, softwareID).Scan(&n)
	return n, err
}

func (h *Handler) softwareEmployees(ctx context.Context, softwareID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT e.id, e.first_name, e.last_name, e.email
		FROM employee_software es
		JOIN employees e ON e.id = es.employee_id
		WHERE es.software_id=$1
		ORDER BY e.first_name, e.last_name`, softwareID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, first, last, email string
		if err := rows.Scan(&id, &first, &last, &email); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"id": id, "first_name": first, "last_name": last, "email": email,
		})
	}
	return out, nil
}

func (h *Handler) softwareFiles(ctx context.Context, softwareID string) ([]map[string]interface{}, error) {
	rows, err := h.pool.Query(ctx, `
		SELECT id, file_name, COALESCE(mime_type,''), size FROM media
		WHERE model_type=$1 AND model_id=$2 AND collection_name=$3 ORDER BY id`,
		modelSoftware, softwareID, collectionSoftware)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, size int64
		var fileName, mimeType string
		if err := rows.Scan(&id, &fileName, &mimeType, &size); err != nil {
			return nil, err
		}
		out = append(out, filePayload(id, fileName, mimeType, size))
	}
	return out, nil
}

func (h *Handler) softwarePayload(ctx context.Context, id, companyID, name string, productKey *string, seats int,
	website, licensedToName, licensedToEmail, orderNumber *string,
	purchaseAmount *int64, currency *string, convertedAmount *int64, convertedCurrency *string,
	convertedAt *time.Time, rate *float64, purchasedAt *time.Time, createdAt, updatedAt time.Time,
) (map[string]interface{}, error) {
	used, err := h.seatsUsed(ctx, id)
	if err != nil {
		return nil, err
	}
	employees, err := h.softwareEmployees(ctx, id)
	if err != nil {
		return nil, err
	}
	files, err := h.softwareFiles(ctx, id)
	if err != nil {
		return nil, err
	}
	remaining := seats - used
	if remaining < 0 {
		remaining = 0
	}
	return map[string]interface{}{
		"id": id, "company_id": companyID, "name": name, "product_key": productKey,
		"seats": seats, "seats_used": used, "remaining_seats": remaining,
		"website": website, "licensed_to_name": licensedToName,
		"licensed_to_email_address": licensedToEmail, "order_number": orderNumber,
		"purchase_amount": purchaseAmount, "currency": currency,
		"converted_purchase_amount": convertedAmount, "converted_to_currency": convertedCurrency,
		"converted_at": isoTime(convertedAt), "exchange_rate": rate,
		"purchased_at": isoDate(purchasedAt),
		"employees": employees, "files": files,
		"created_at": isoTime(&createdAt), "updated_at": isoTime(&updatedAt),
	}, nil
}

func (h *Handler) loadSoftware(ctx context.Context, companyID, softwareID string) (map[string]interface{}, error) {
	var id, cid, name string
	var productKey, website, licensedToName, licensedToEmail, orderNumber, currency, convertedCurrency *string
	var seats int
	var purchaseAmount, convertedAmount *int64
	var rate *float64
	var purchasedAt, convertedAt *time.Time
	var createdAt, updatedAt time.Time
	err := h.pool.QueryRow(ctx, `
		SELECT id, company_id, name, product_key, seats, website, licensed_to_name, licensed_to_email_address,
			order_number, purchase_amount, currency, converted_purchase_amount, converted_to_currency,
			converted_at, exchange_rate::float8, purchased_at, created_at, updated_at
		FROM softwares WHERE id=$1 AND company_id=$2`, softwareID, companyID,
	).Scan(&id, &cid, &name, &productKey, &seats, &website, &licensedToName, &licensedToEmail,
		&orderNumber, &purchaseAmount, &currency, &convertedAmount, &convertedCurrency,
		&convertedAt, &rate, &purchasedAt, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return h.softwarePayload(ctx, id, cid, name, productKey, seats, website, licensedToName, licensedToEmail,
		orderNumber, purchaseAmount, currency, convertedAmount, convertedCurrency, convertedAt, rate,
		purchasedAt, createdAt, updatedAt)
}

func (h *Handler) ListSoftwares(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	rows, err := h.pool.Query(r.Context(), `
		SELECT id FROM softwares WHERE company_id=$1 ORDER BY name`, member.CompanyID)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	items := []map[string]interface{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			response.Fail(w, 500, "scan failed", err.Error())
			return
		}
		item, err := h.loadSoftware(r.Context(), member.CompanyID, id)
		if err != nil {
			response.Fail(w, 500, "load failed", err.Error())
			return
		}
		items = append(items, item)
	}
	response.OK(w, "", items)
}

func (h *Handler) CreateSoftware(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	var req struct {
		Name                    string  `json:"name"`
		ProductKey              string  `json:"product_key"`
		Seats                   int     `json:"seats"`
		Website                 *string `json:"website"`
		LicensedToName          *string `json:"licensed_to_name"`
		LicensedToEmailAddress  *string `json:"licensed_to_email_address"`
		OrderNumber             *string `json:"order_number"`
		PurchaseAmount          *int64  `json:"purchase_amount"`
		Currency                *string `json:"currency"`
		PurchasedAt             *string `json:"purchased_at"`
	}
	if err := decodeJSON(r, &req); err != nil || strings.TrimSpace(req.Name) == "" || req.ProductKey == "" || req.Seats < 1 {
		response.Fail(w, 422, "invalid body", nil)
		return
	}
	if req.Currency != nil {
		c := strings.ToUpper(*req.Currency)
		req.Currency = &c
	}
	companyCurrency, err := h.companyCurrency(r.Context(), member.CompanyID)
	if err != nil {
		response.Fail(w, 500, "company lookup failed", err.Error())
		return
	}
	convertedAmount, convertedCurrency, convertedAt, rate := h.convertPurchase(r.Context(), companyCurrency, req.PurchaseAmount, req.Currency, req.PurchasedAt)
	id := uuidv7.New()
	_, err = h.pool.Exec(r.Context(), `
		INSERT INTO softwares (
			id, company_id, name, product_key, seats, website, licensed_to_name, licensed_to_email_address,
			order_number, purchase_amount, currency, converted_purchase_amount, converted_to_currency,
			converted_at, exchange_rate, purchased_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::date)`,
		id, member.CompanyID, req.Name, req.ProductKey, req.Seats, req.Website, req.LicensedToName,
		req.LicensedToEmailAddress, req.OrderNumber, req.PurchaseAmount, req.Currency,
		convertedAmount, convertedCurrency, convertedAt, rate, req.PurchasedAt)
	if err != nil {
		response.Fail(w, 500, "create failed", err.Error())
		return
	}
	item, err := h.loadSoftware(r.Context(), member.CompanyID, id)
	if err != nil {
		response.Fail(w, 500, "load failed", err.Error())
		return
	}
	response.Created(w, "Software created", item)
}

func (h *Handler) ShowSoftware(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	item, err := h.loadSoftware(r.Context(), member.CompanyID, chi.URLParam(r, "softwareId"))
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

func (h *Handler) UpdateSoftware(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	softwareID := chi.URLParam(r, "softwareId")
	var req map[string]interface{}
	if err := decodeJSON(r, &req); err != nil {
		response.Fail(w, 422, "invalid body", nil)
		return
	}

	var name string
	var productKey string
	var seats int
	var website, licensedToName, licensedToEmail, orderNumber, currency *string
	var purchaseAmount *int64
	var purchasedAt *time.Time
	err := h.pool.QueryRow(r.Context(), `
		SELECT name, COALESCE(product_key,''), seats, website, licensed_to_name, licensed_to_email_address,
			order_number, purchase_amount, currency, purchased_at
		FROM softwares WHERE id=$1 AND company_id=$2`, softwareID, member.CompanyID,
	).Scan(&name, &productKey, &seats, &website, &licensedToName, &licensedToEmail,
		&orderNumber, &purchaseAmount, &currency, &purchasedAt)
	if err == pgx.ErrNoRows {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return
	}

	var purchasedAtStr *string
	if purchasedAt != nil {
		s := purchasedAt.Format("2006-01-02")
		purchasedAtStr = &s
	}

	if v, ok := req["name"].(string); ok && strings.TrimSpace(v) != "" {
		name = v
	}
	if v, ok := req["product_key"].(string); ok {
		productKey = v
	}
	if v, ok := req["seats"].(float64); ok && int(v) >= 1 {
		seats = int(v)
	}
	setStr := func(key string, dest **string) {
		if raw, ok := req[key]; ok {
			if raw == nil {
				*dest = nil
			} else if s, ok := raw.(string); ok {
				*dest = &s
			}
		}
	}
	setStr("website", &website)
	setStr("licensed_to_name", &licensedToName)
	setStr("licensed_to_email_address", &licensedToEmail)
	setStr("order_number", &orderNumber)
	if raw, ok := req["currency"]; ok {
		if raw == nil {
			currency = nil
		} else if s, ok := raw.(string); ok {
			c := strings.ToUpper(s)
			currency = &c
		}
	}
	if raw, ok := req["purchase_amount"]; ok {
		if raw == nil {
			purchaseAmount = nil
		} else if f, ok := raw.(float64); ok {
			n := int64(f)
			purchaseAmount = &n
		}
	}
	if raw, ok := req["purchased_at"]; ok {
		if raw == nil {
			purchasedAtStr = nil
		} else if s, ok := raw.(string); ok {
			purchasedAtStr = &s
		}
	}

	companyCurrency, err := h.companyCurrency(r.Context(), member.CompanyID)
	if err != nil {
		response.Fail(w, 500, "company lookup failed", err.Error())
		return
	}
	convertedAmount, convertedCurrency, convertedAt, rate := h.convertPurchase(r.Context(), companyCurrency, purchaseAmount, currency, purchasedAtStr)

	_, err = h.pool.Exec(r.Context(), `
		UPDATE softwares SET
			name=$1, product_key=$2, seats=$3, website=$4, licensed_to_name=$5, licensed_to_email_address=$6,
			order_number=$7, purchase_amount=$8, currency=$9, converted_purchase_amount=$10,
			converted_to_currency=$11, converted_at=$12, exchange_rate=$13, purchased_at=$14::date, updated_at=now()
		WHERE id=$15 AND company_id=$16`,
		name, productKey, seats, website, licensedToName, licensedToEmail, orderNumber,
		purchaseAmount, currency, convertedAmount, convertedCurrency, convertedAt, rate, purchasedAtStr,
		softwareID, member.CompanyID)
	if err != nil {
		response.Fail(w, 500, "update failed", err.Error())
		return
	}
	out, err := h.loadSoftware(r.Context(), member.CompanyID, softwareID)
	if err != nil {
		response.Fail(w, 500, "load failed", err.Error())
		return
	}
	response.OK(w, "Software updated", out)
}

func (h *Handler) DestroySoftware(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	softwareID := chi.URLParam(r, "softwareId")
	_, _ = h.pool.Exec(r.Context(), `DELETE FROM media WHERE model_type=$1 AND model_id=$2 AND collection_name=$3`,
		modelSoftware, softwareID, collectionSoftware)
	_, _ = h.pool.Exec(r.Context(), `DELETE FROM employee_software WHERE software_id=$1`, softwareID)
	tag, err := h.pool.Exec(r.Context(), `DELETE FROM softwares WHERE id=$1 AND company_id=$2`, softwareID, member.CompanyID)
	if err != nil {
		response.Fail(w, 500, "delete failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	response.OK(w, "Software deleted", nil)
}

func (h *Handler) GiveSeat(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	softwareID := chi.URLParam(r, "softwareId")
	var req struct {
		EmployeeID string `json:"employee_id"`
	}
	if err := decodeJSON(r, &req); err != nil || req.EmployeeID == "" {
		response.Fail(w, 422, "invalid body", nil)
		return
	}
	item, err := h.loadSoftware(r.Context(), member.CompanyID, softwareID)
	if err == pgx.ErrNoRows {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return
	}
	if !h.employeeInCompany(r.Context(), member.CompanyID, req.EmployeeID) {
		response.Fail(w, 404, "Employee not found", nil)
		return
	}
	used, _ := item["seats_used"].(int)
	seats, _ := item["seats"].(int)
	if used >= seats {
		response.Fail(w, 422, "No seats remaining", nil)
		return
	}
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM employee_software WHERE software_id=$1 AND employee_id=$2)`,
		softwareID, req.EmployeeID).Scan(&exists)
	if !exists {
		_, err = h.pool.Exec(r.Context(), `
			INSERT INTO employee_software (employee_id, software_id) VALUES ($1,$2)`,
			req.EmployeeID, softwareID)
		if err != nil {
			response.Fail(w, 500, "attach failed", err.Error())
			return
		}
	}
	out, _ := h.loadSoftware(r.Context(), member.CompanyID, softwareID)
	response.OK(w, "Seat given", out)
}

func (h *Handler) RevokeSeat(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	softwareID := chi.URLParam(r, "softwareId")
	employeeID := chi.URLParam(r, "employeeId")
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM softwares WHERE id=$1 AND company_id=$2)`, softwareID, member.CompanyID).Scan(&exists)
	if !exists {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	_, _ = h.pool.Exec(r.Context(), `DELETE FROM employee_software WHERE software_id=$1 AND employee_id=$2`, softwareID, employeeID)
	out, _ := h.loadSoftware(r.Context(), member.CompanyID, softwareID)
	response.OK(w, "Seat revoked", out)
}

func (h *Handler) GiveSeatsToAll(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	softwareID := chi.URLParam(r, "softwareId")
	item, err := h.loadSoftware(r.Context(), member.CompanyID, softwareID)
	if err == pgx.ErrNoRows {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return
	}
	remaining, _ := item["remaining_seats"].(int)
	if remaining == 0 {
		response.OK(w, "Seats assigned", map[string]interface{}{"assigned": 0, "software": item})
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT e.id FROM employees e
		WHERE e.company_id=$1 AND e.locked=false
			AND NOT EXISTS (SELECT 1 FROM employee_software es WHERE es.software_id=$2 AND es.employee_id=e.id)
		ORDER BY e.first_name
		LIMIT $3`, member.CompanyID, softwareID, remaining)
	if err != nil {
		response.Fail(w, 500, "query failed", err.Error())
		return
	}
	defer rows.Close()
	assigned := 0
	for rows.Next() {
		var eid string
		if err := rows.Scan(&eid); err != nil {
			response.Fail(w, 500, "scan failed", err.Error())
			return
		}
		_, err = h.pool.Exec(r.Context(), `
			INSERT INTO employee_software (employee_id, software_id) VALUES ($1,$2)
			ON CONFLICT DO NOTHING`, eid, softwareID)
		if err != nil {
			response.Fail(w, 500, "attach failed", err.Error())
			return
		}
		assigned++
	}
	out, _ := h.loadSoftware(r.Context(), member.CompanyID, softwareID)
	response.OK(w, "Seats assigned", map[string]interface{}{"assigned": assigned, "software": out})
}

func (h *Handler) EmployeesWithout(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	softwareID := chi.URLParam(r, "softwareId")
	item, err := h.loadSoftware(r.Context(), member.CompanyID, softwareID)
	if err == pgx.ErrNoRows {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return
	}
	var without int
	_ = h.pool.QueryRow(r.Context(), `
		SELECT COUNT(*) FROM employees e
		WHERE e.company_id=$1 AND e.locked=false
			AND NOT EXISTS (SELECT 1 FROM employee_software es WHERE es.software_id=$2 AND es.employee_id=e.id)`,
		member.CompanyID, softwareID).Scan(&without)
	response.OK(w, "", map[string]interface{}{
		"employees_without": without,
		"remaining_seats":   item["remaining_seats"],
		"seats":             item["seats"],
	})
}

func (h *Handler) AttachSoftwareFile(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	softwareID := chi.URLParam(r, "softwareId")
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM softwares WHERE id=$1 AND company_id=$2)`, softwareID, member.CompanyID).Scan(&exists)
	if !exists {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	var body struct {
		TemporaryUploadID int64 `json:"temporary_upload_id"`
		MediaID           int64 `json:"media_id"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.Fail(w, 422, "invalid body", nil)
		return
	}
	var fileName string
	err := h.pool.QueryRow(r.Context(), `
		SELECT file_name FROM media
		WHERE id=$1 AND model_type=$2 AND model_id=$3`,
		body.MediaID, modelTemporaryUpload, fmt.Sprintf("%d", body.TemporaryUploadID),
	).Scan(&fileName)
	if err == pgx.ErrNoRows {
		response.Fail(w, 404, "Media not found on temporary upload", nil)
		return
	}
	if err != nil {
		response.Fail(w, 500, "lookup failed", err.Error())
		return
	}
	_, err = h.pool.Exec(r.Context(), `
		UPDATE media SET model_type=$2, model_id=$3, collection_name=$4, updated_at=now()
		WHERE id=$1 AND model_type=$5 AND model_id=$6`,
		body.MediaID, modelSoftware, softwareID, collectionSoftware,
		modelTemporaryUpload, fmt.Sprintf("%d", body.TemporaryUploadID))
	if err != nil {
		response.Fail(w, 500, "attach failed", err.Error())
		return
	}
	var mimeType string
	var size int64
	_ = h.pool.QueryRow(r.Context(), `SELECT mime_type, size FROM media WHERE id=$1`, body.MediaID).Scan(&mimeType, &size)
	response.OK(w, "File attached", filePayload(body.MediaID, fileName, mimeType, size))
}

func (h *Handler) DetachSoftwareFile(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	softwareID := chi.URLParam(r, "softwareId")
	mediaID := chi.URLParam(r, "mediaId")
	var exists bool
	_ = h.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM softwares WHERE id=$1 AND company_id=$2)`, softwareID, member.CompanyID).Scan(&exists)
	if !exists {
		response.Fail(w, 404, "Not found", nil)
		return
	}
	tag, err := h.pool.Exec(r.Context(), `
		DELETE FROM media WHERE id=$1 AND model_type=$2 AND model_id=$3 AND collection_name=$4`,
		mediaID, modelSoftware, softwareID, collectionSoftware)
	if err != nil {
		response.Fail(w, 500, "delete failed", err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		response.Fail(w, 404, "File not found", nil)
		return
	}
	response.OK(w, "File detached", nil)
}

func (h *Handler) EmployeeSoftwares(w http.ResponseWriter, r *http.Request) {
	member, _ := companyauth.MemberFromContext(r.Context())
	employeeID := chi.URLParam(r, "employeeId")
	if !h.canViewEmployeeAssets(member, employeeID, "software.view") {
		response.Fail(w, 403, "Forbidden", nil)
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT s.id FROM softwares s
		JOIN employee_software es ON es.software_id = s.id
		WHERE s.company_id=$1 AND es.employee_id=$2
		ORDER BY s.name`, member.CompanyID, employeeID)
	if err != nil {
		response.Fail(w, 500, "list failed", err.Error())
		return
	}
	defer rows.Close()
	items := []map[string]interface{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			response.Fail(w, 500, "scan failed", err.Error())
			return
		}
		item, err := h.loadSoftware(r.Context(), member.CompanyID, id)
		if err != nil {
			response.Fail(w, 500, "load failed", err.Error())
			return
		}
		items = append(items, item)
	}
	response.OK(w, "", items)
}
