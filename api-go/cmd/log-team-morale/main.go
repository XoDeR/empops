package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/XoDeR/empops/api-go/internal/infrastructure/config"
	"github.com/XoDeR/empops/api-go/internal/infrastructure/database"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "log-team-morale: fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := envOrDefault("EMPOPS_CONFIG", "config/app.dev.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	date := time.Now().UTC().Format("2006-01-02")
	if len(os.Args) > 1 {
		date = os.Args[1]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := database.Connect(ctx, database.Config{DSN: cfg.DB.DSN, ConnectTimeout: 10 * time.Second})
	if err != nil {
		return err
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, `SELECT id FROM teams`)
	if err != nil {
		return err
	}
	defer rows.Close()
	created := 0
	for rows.Next() {
		var teamID string
		if rows.Scan(&teamID) != nil {
			continue
		}
		var exists bool
		_ = pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM morale_team_histories
				WHERE team_id=$1 AND created_at::date=$2::date)`, teamID, date).Scan(&exists)
		if exists {
			continue
		}
		var avg *float64
		var count int
		_ = pool.QueryRow(ctx, `
			SELECT AVG(m.emotion)::float8, COUNT(*)::int FROM morales m
			JOIN employee_team te ON te.employee_id=m.employee_id
			WHERE te.team_id=$1 AND m.created_at::date=$2::date`, teamID, date).Scan(&avg, &count)
		average := 0.0
		if avg != nil {
			average = *avg
		}
		_, err := pool.Exec(ctx, `
			INSERT INTO morale_team_histories (id, team_id, average, number_of_team_members, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5::date,$5::date)`, uuidv7.New(), teamID, average, count, date)
		if err == nil {
			created++
		}
	}
	fmt.Printf("log-team-morale: created %d row(s) for %s\n", created, date)
	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
