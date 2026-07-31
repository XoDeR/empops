package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/XoDeR/empops/api-go/internal/infrastructure/config"
	"github.com/XoDeR/empops/api-go/internal/infrastructure/database"
	"github.com/XoDeR/empops/api-go/pkg/notify"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "rate-manager-start: fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := envOrDefault("EMPOPS_CONFIG", "config/app.dev.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	pool, err := database.Connect(ctx, database.Config{DSN: cfg.DB.DSN, ConnectTimeout: 10 * time.Second})
	if err != nil {
		return err
	}
	defer pool.Close()

	validUntil := time.Now().UTC().AddDate(0, 0, 5) // approx 3 weekdays buffer
	rows, err := pool.Query(ctx, `SELECT DISTINCT manager_id, company_id FROM direct_reports`)
	if err != nil {
		return err
	}
	defer rows.Close()
	created := 0
	for rows.Next() {
		var managerID, companyID string
		if rows.Scan(&managerID, &companyID) != nil {
			continue
		}
		var locked bool
		_ = pool.QueryRow(ctx, `SELECT locked FROM employees WHERE id=$1`, managerID).Scan(&locked)
		if locked {
			continue
		}
		surveyID := uuidv7.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO rate_your_manager_surveys (id, company_id, manager_id, active, valid_until_at)
			VALUES ($1,$2,$3,true,$4)`, surveyID, companyID, managerID, validUntil)
		if err != nil {
			continue
		}
		reports, _ := pool.Query(ctx, `
			SELECT employee_id FROM direct_reports dr
			JOIN employees e ON e.id=dr.employee_id
			WHERE dr.company_id=$1 AND dr.manager_id=$2 AND e.locked=false`, companyID, managerID)
		if reports != nil {
			for reports.Next() {
				var empID string
				if reports.Scan(&empID) == nil {
					_, _ = pool.Exec(ctx, `
						INSERT INTO rate_your_manager_answers (id, rate_your_manager_survey_id, employee_id, active)
						VALUES ($1,$2,$3,true)`, uuidv7.New(), surveyID, empID)
					_ = notify.Create(ctx, pool, companyID, empID, "rate_manager.pending", map[string]interface{}{
						"survey_id": surveyID, "manager_id": managerID,
					})
				}
			}
			reports.Close()
		}
		created++
	}
	fmt.Printf("rate-manager-start: started %d survey(s)\n", created)
	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
