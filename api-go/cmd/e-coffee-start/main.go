package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/XoDeR/empops/api-go/internal/infrastructure/config"
	"github.com/XoDeR/empops/api-go/internal/infrastructure/database"
	"github.com/XoDeR/empops/api-go/pkg/uuidv7"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "e-coffee-start: fatal:", err)
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

	rows, err := pool.Query(ctx, `SELECT id FROM companies WHERE e_coffee_enabled=true`)
	if err != nil {
		return err
	}
	defer rows.Close()
	started := 0
	for rows.Next() {
		var companyID string
		if rows.Scan(&companyID) != nil {
			continue
		}
		empRows, err := pool.Query(ctx, `SELECT id FROM employees WHERE company_id=$1 AND locked=false`, companyID)
		if err != nil {
			continue
		}
		ids := []string{}
		for empRows.Next() {
			var id string
			if empRows.Scan(&id) == nil {
				ids = append(ids, id)
			}
		}
		empRows.Close()
		rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

		var batch int
		_ = pool.QueryRow(ctx, `SELECT COALESCE(MAX(batch_number),0) FROM e_coffees WHERE company_id=$1`, companyID).Scan(&batch)
		batch++
		_, _ = pool.Exec(ctx, `UPDATE e_coffees SET active=false WHERE company_id=$1 AND active=true`, companyID)
		sessionID := uuidv7.New()
		_, err = pool.Exec(ctx, `
			INSERT INTO e_coffees (id, company_id, batch_number, active) VALUES ($1,$2,$3,true)`,
			sessionID, companyID, batch)
		if err != nil {
			continue
		}
		if len(ids) >= 2 {
			half := len(ids) / 2
			first := ids[:half]
			second := ids[half:]
			for len(second) > len(first) && len(first) > 0 {
				first = append(first, first[rand.Intn(len(first))])
			}
			n := len(first)
			if len(second) < n {
				n = len(second)
			}
			for i := 0; i < n; i++ {
				_, _ = pool.Exec(ctx, `
					INSERT INTO e_coffee_matches (id, e_coffee_id, employee_id, with_employee_id, happened)
					VALUES ($1,$2,$3,$4,false)`, uuidv7.New(), sessionID, first[i], second[i])
			}
		}
		started++
	}
	fmt.Printf("e-coffee-start: started %d session(s)\n", started)
	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
