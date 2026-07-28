package audit

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

// Log writes a company-scoped activity log row.
func Log(ctx context.Context, pool *pgxpool.Pool, companyID, event string, causerID *string, subjectType *string, subjectID *string, props map[string]interface{}) error {
	if props == nil {
		props = map[string]interface{}{}
	}
	props["company_id"] = companyID
	props["event"] = event
	raw, err := json.Marshal(props)
	if err != nil {
		return err
	}

	var causerType *string
	if causerID != nil && *causerID != "" {
		t := "employee"
		causerType = &t
	} else {
		causerID = nil
	}

	id := uuidv7.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO activity_logs (id, company_id, event, description, subject_type, subject_id, causer_type, causer_id, properties)
		VALUES ($1, $2, $3, $3, $4, $5, $6, $7, $8::jsonb)`,
		id, companyID, event, subjectType, subjectID, causerType, causerID, string(raw),
	)
	return err
}

// LogEmployee logs with an employee causer ID.
func LogEmployee(ctx context.Context, pool *pgxpool.Pool, companyID, event, causerEmployeeID string, subjectType, subjectID *string, props map[string]interface{}) error {
	var causer *string
	if causerEmployeeID != "" {
		causer = &causerEmployeeID
	}
	return Log(ctx, pool, companyID, event, causer, subjectType, subjectID, props)
}
