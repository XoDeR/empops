package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if os.Getenv("ENABLE_PAID_PLAN") != "true" && os.Getenv("ENABLE_PAID_PLAN") != "1" {
		fmt.Println("paid plan disabled; skipping")
		return
	}
	dsn := os.Getenv("EMPOPS_DB_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "EMPOPS_DB_DSN required")
		os.Exit(1)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer pool.Close()

	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, -1)

	rows, err := pool.Query(ctx, `
		SELECT DISTINCT ON (company_id) id, company_id
		FROM company_daily_usage_history
		WHERE logged_on >= $1::date AND logged_on <= $2::date
		ORDER BY company_id, number_of_active_employees DESC, logged_on DESC`,
		start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer rows.Close()

	created := 0
	for rows.Next() {
		var usageID, companyID string
		if err := rows.Scan(&usageID, &companyID); err != nil {
			continue
		}
		var exists bool
		_ = pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM company_invoices
				WHERE company_id=$1 AND usage_history_id=$2
			)`, companyID, usageID).Scan(&exists)
		if exists {
			continue
		}
		_, err := pool.Exec(ctx, `
			INSERT INTO company_invoices (id, company_id, usage_history_id, sent_to_customer, customer_has_paid, created_at, updated_at)
			VALUES ($1,$2,$3,false,false,now(),now())`,
			uuid.NewString(), companyID, usageID)
		if err == nil {
			created++
		}
	}
	fmt.Printf("created %d invoice(s)\n", created)
}
