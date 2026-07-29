package mediaurl

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	modelEmployee = "employee"
	modelCompany  = "company"
)

// AvatarURL returns the public URL for an employee avatar if set.
func AvatarURL(ctx context.Context, pool *pgxpool.Pool, employeeID string) *string {
	return collectionURL(ctx, pool, modelEmployee, employeeID, "avatar")
}

// LogoURL returns the public URL for a company logo if set.
func LogoURL(ctx context.Context, pool *pgxpool.Pool, companyID string) *string {
	return collectionURL(ctx, pool, modelCompany, companyID, "logo")
}

func collectionURL(ctx context.Context, pool *pgxpool.Pool, modelType, modelID, collection string) *string {
	var mediaID int64
	err := pool.QueryRow(ctx, `
		SELECT id FROM media
		WHERE model_type = $1 AND model_id = $2 AND collection_name = $3
		ORDER BY id DESC LIMIT 1`,
		modelType, modelID, collection,
	).Scan(&mediaID)
	if err != nil {
		return nil
	}
	url := fmt.Sprintf("/api/v1/media/%d/file", mediaID)
	return &url
}
