// Command migrate applies namespaced SQL migrations under migrations/<namespace>
// to PostgreSQL, tracking what has run in a schema_migrations table.
//
// Usage:
//
//	go run ./cmd/migrate            # apply all pending migrations (default: up)
//	go run ./cmd/migrate up
//	go run ./cmd/migrate down [n]    # roll back the last n migrations (default 1)
//	go run ./cmd/migrate status      # list discovered migrations and whether applied
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/XoDeR/empops/api-go/internal/infrastructure/config"
	"github.com/XoDeR/empops/api-go/internal/infrastructure/database"
	"github.com/XoDeR/empops/api-go/pkg/migration"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate: fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	migrationsDir := envOrDefault("EMPOPS_MIGRATIONS_DIR", "migrations")
	configPath := envOrDefault("EMPOPS_CONFIG", "config/app.dev.yaml")
	modulesPath := envOrDefault("EMPOPS_MODULES_CONFIG", "config/modules.yaml")

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	modulesCfg, err := config.LoadModules(modulesPath)
	if err != nil {
		return fmt.Errorf("load modules config: %w", err)
	}

	// Core owns the users/refresh_tokens/roles/permissions tables every
	// module's migrations may reference, so it always runs first; enabled
	// modules then run in the order listed in config/modules.yaml (which
	// must already respect FK dependencies, e.g. company before employee).
	namespaces := append([]string{"core"}, modulesCfg.Enabled...)

	args := os.Args[1:]
	command := "up"
	if len(args) > 0 {
		command = args[0]
	}

	if command == "status" {
		return runStatus(migrationsDir, namespaces)
	}

	if cfg.DB.DSN == "" {
		return fmt.Errorf("no database DSN configured (set EMPOPS_DB_DSN or db.dsn in %s)", configPath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.Connect(ctx, database.Config{DSN: cfg.DB.DSN, ConnectTimeout: 10 * time.Second})
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	runner := migration.NewRunner(pool)

	switch command {
	case "up":
		applied, err := runner.Up(ctx, migrationsDir, namespaces)
		if err != nil {
			return err
		}
		if len(applied) == 0 {
			fmt.Println("migrate: no pending migrations")
			return nil
		}
		fmt.Printf("migrate: applied %d migration(s)\n", len(applied))
		for _, m := range applied {
			fmt.Printf("  + [%s] %s_%s\n", m.Namespace, m.Version, m.Name)
		}
		return nil

	case "down":
		steps := 1
		if len(args) > 1 {
			if n, err := strconv.Atoi(args[1]); err == nil {
				steps = n
			}
		}
		reverted, err := runner.Down(ctx, migrationsDir, namespaces, steps)
		if err != nil {
			return err
		}
		if len(reverted) == 0 {
			fmt.Println("migrate: nothing to roll back")
			return nil
		}
		fmt.Printf("migrate: reverted %d migration(s)\n", len(reverted))
		for _, m := range reverted {
			fmt.Printf("  - [%s] %s_%s\n", m.Namespace, m.Version, m.Name)
		}
		return nil

	default:
		return fmt.Errorf("unknown command %q (expected up, down, or status)", command)
	}
}

func runStatus(migrationsDir string, namespaces []string) error {
	discovered, err := migration.Discover(migrationsDir)
	if err != nil {
		return err
	}
	fmt.Printf("migrate: discovered %d migration(s) under %s (namespace order: %v)\n", len(discovered), migrationsDir, namespaces)
	for _, m := range discovered {
		fmt.Printf("  - [%s] %s_%s (up=%t down=%t)\n", m.Namespace, m.Version, m.Name, m.UpFile != "", m.DownFile != "")
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
