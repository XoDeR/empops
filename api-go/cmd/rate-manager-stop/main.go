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
		fmt.Fprintln(os.Stderr, "rate-manager-stop: fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := envOrDefault("EMPOPS_CONFIG", "config/app.dev.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	force := false
	for _, a := range os.Args[1:] {
		if a == "--force" {
			force = true
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := database.Connect(ctx, database.Config{DSN: cfg.DB.DSN, ConnectTimeout: 10 * time.Second})
	if err != nil {
		return err
	}
	defer pool.Close()

	q := `SELECT id FROM rate_your_manager_surveys WHERE active=true`
	if !force {
		q += ` AND valid_until_at <= now()`
	}
	rows, err := pool.Query(ctx, q)
	if err != nil {
		return err
	}
	defer rows.Close()
	stopped := 0
	for rows.Next() {
		var id string
		if rows.Scan(&id) != nil {
			continue
		}
		_, _ = pool.Exec(ctx, `UPDATE rate_your_manager_surveys SET active=false, updated_at=now() WHERE id=$1`, id)
		_, _ = pool.Exec(ctx, `UPDATE rate_your_manager_answers SET active=false, updated_at=now() WHERE rate_your_manager_survey_id=$1`, id)
		stopped++
	}
	fmt.Printf("rate-manager-stop: stopped %d survey(s)\n", stopped)
	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
