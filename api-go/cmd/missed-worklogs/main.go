// Command missed-worklogs increments employees.consecutive_worklog_missed for
// every unlocked employee who has not logged a worklog for a given date
// (defaults to today, UTC). Intended to run once daily near end-of-day via
// cron/scheduler, e.g.:
//
//	# crontab (run at 23:55 UTC every day)
//	55 23 * * * cd /path/to/api-go && EMPOPS_DB_DSN=postgres://... ./bin/missed-worklogs.exe
//
// or, during development:
//
//	EMPOPS_DB_DSN=postgres://... go run ./cmd/missed-worklogs
//	go run ./cmd/missed-worklogs 2026-07-29   # backfill a specific date
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/XoDeR/empops/api-go/internal/infrastructure/config"
	"github.com/XoDeR/empops/api-go/internal/infrastructure/database"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "missed-worklogs: fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := envOrDefault("EMPOPS_CONFIG", "config/app.dev.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.DB.DSN == "" {
		return fmt.Errorf("no database DSN configured (set EMPOPS_DB_DSN or db.dsn in %s)", configPath)
	}

	date := time.Now().UTC().Format("2006-01-02")
	if len(os.Args) > 1 {
		date = os.Args[1]
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid date %q (expected YYYY-MM-DD): %w", date, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, database.Config{DSN: cfg.DB.DSN, ConnectTimeout: 10 * time.Second})
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	tag, err := pool.Exec(ctx, `
		UPDATE employees e
		SET consecutive_worklog_missed = consecutive_worklog_missed + 1,
			updated_at = now()
		WHERE e.locked = false
			AND NOT EXISTS (
				SELECT 1 FROM worklogs w
				WHERE w.employee_id = e.id AND w.logged_on = $1
			)`, date,
	)
	if err != nil {
		return fmt.Errorf("update missed worklogs: %w", err)
	}

	fmt.Printf("missed-worklogs: incremented consecutive_worklog_missed for %d employee(s) missing a worklog on %s\n", tag.RowsAffected(), date)
	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
