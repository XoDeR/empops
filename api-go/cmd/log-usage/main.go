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

	day := time.Now().UTC().Format("2006-01-02")
	if len(os.Args) > 1 {
		day = os.Args[1]
	}

	rows, err := pool.Query(ctx, `SELECT id FROM companies`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var companyID string
		if err := rows.Scan(&companyID); err != nil {
			continue
		}
		var n int
		_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM employees WHERE company_id=$1 AND locked=false`, companyID).Scan(&n)
		id := uuid.NewString()
		_, err := pool.Exec(ctx, `
			INSERT INTO company_daily_usage_history (id, company_id, number_of_active_employees, logged_on, created_at, updated_at)
			VALUES ($1,$2,$3,$4::date,now(),now())
			ON CONFLICT (company_id, logged_on) DO UPDATE SET number_of_active_employees=EXCLUDED.number_of_active_employees, updated_at=now()`,
			id, companyID, n, day)
		if err == nil {
			count++
		}
	}
	fmt.Printf("logged usage for %d company(ies) on %s\n", count, day)
}
