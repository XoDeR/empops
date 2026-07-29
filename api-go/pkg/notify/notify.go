// Package notify provides a shared helper for inserting company-scoped
// notification rows, used by any vertical module that needs to alert an
// employee (e.g. the team module when an employee is attached to a ship).
package notify

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

// Create inserts an unread notification row for employeeID within companyID.
func Create(ctx context.Context, pool *pgxpool.Pool, companyID, employeeID, action string, objects map[string]interface{}) error {
	if objects == nil {
		objects = map[string]interface{}{}
	}
	raw, err := json.Marshal(objects)
	if err != nil {
		return err
	}

	id := uuidv7.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO notifications (id, company_id, employee_id, action, objects)
		VALUES ($1, $2, $3, $4, $5::jsonb)`,
		id, companyID, employeeID, action, string(raw),
	)
	return err
}
