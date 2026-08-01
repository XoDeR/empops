package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/internal/modules/finance/adapter/frankfurter"
	"github.com/XoDeR/empops/api-go/pkg/companyauth"
)

const (
	modelTemporaryUpload = "temporary_upload"
	modelSoftware        = "software"
	collectionSoftware   = "software"
)

type RateProvider interface {
	Rate(context.Context, string, string, string) (float64, error)
}

type Handler struct {
	pool *pgxpool.Pool
	fx   RateProvider
}

func NewHandler(pool *pgxpool.Pool, fx RateProvider) *Handler {
	if fx == nil {
		fx = frankfurter.New()
	}
	return &Handler{pool: pool, fx: fx}
}

func decodeJSON(r *http.Request, dst interface{}) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (h *Handler) employeeSummary(ctx context.Context, id string) map[string]interface{} {
	var first, last, email string
	_ = h.pool.QueryRow(ctx, `SELECT first_name, last_name, email FROM employees WHERE id=$1`, id).
		Scan(&first, &last, &email)
	return map[string]interface{}{"id": id, "first_name": first, "last_name": last, "email": email}
}

func (h *Handler) canViewEmployeeAssets(m companyauth.Member, employeeID, viewPerm string) bool {
	if m.EmployeeID == employeeID {
		return true
	}
	return m.HasPermission(viewPerm)
}

func (h *Handler) employeeInCompany(ctx context.Context, companyID, employeeID string) bool {
	var ok bool
	_ = h.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM employees WHERE id=$1 AND company_id=$2)`, employeeID, companyID).Scan(&ok)
	return ok
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

func filePayload(id int64, fileName, mimeType string, size int64) map[string]interface{} {
	return map[string]interface{}{
		"id": id, "file_name": fileName, "mime_type": mimeType, "size": size,
		"url": fmt.Sprintf("/api/v1/media/%d/file", id),
	}
}

func (h *Handler) convertPurchase(ctx context.Context, companyCurrency string, amount *int64, currency *string, purchasedAt *string) (
	convertedAmount *int64, convertedCurrency *string, convertedAt *time.Time, rate *float64,
) {
	if amount == nil || currency == nil || *currency == "" {
		return nil, nil, nil, nil
	}
	from := strings.ToUpper(*currency)
	to := strings.ToUpper(companyCurrency)
	if from == to {
		return nil, nil, nil, nil
	}
	date := time.Now().UTC().Format("2006-01-02")
	if purchasedAt != nil && *purchasedAt != "" {
		date = *purchasedAt
	}
	v, err := h.fx.Rate(ctx, date, from, to)
	if err != nil {
		return nil, nil, nil, nil
	}
	c := int64(math.Round(float64(*amount) * v))
	now := time.Now().UTC()
	return &c, &to, &now, &v
}

func (h *Handler) companyCurrency(ctx context.Context, companyID string) (string, error) {
	var currency string
	err := h.pool.QueryRow(ctx, `SELECT currency FROM companies WHERE id=$1`, companyID).Scan(&currency)
	return currency, err
}

func (h *Handler) scanHardwareRow(ctx context.Context, rows pgx.Row) (map[string]interface{}, error) {
	var id, companyID, name string
	var employeeID, serial *string
	var createdAt, updatedAt time.Time
	if err := rows.Scan(&id, &companyID, &employeeID, &name, &serial, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	var employee interface{}
	if employeeID != nil {
		employee = h.employeeSummary(ctx, *employeeID)
	}
	return map[string]interface{}{
		"id": id, "company_id": companyID, "name": name, "serial_number": serial,
		"employee_id": employeeID, "employee": employee,
		"created_at": isoTime(&createdAt), "updated_at": isoTime(&updatedAt),
	}, nil
}

func (h *Handler) loadHardware(ctx context.Context, companyID, hardwareID string) (map[string]interface{}, error) {
	row := h.pool.QueryRow(ctx, `
		SELECT id, company_id, employee_id, name, serial_number, created_at, updated_at
		FROM hardware WHERE id=$1 AND company_id=$2`, hardwareID, companyID)
	return h.scanHardwareRow(ctx, row)
}
