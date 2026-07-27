// Command migrate discovers and (eventually) applies namespaced SQL
// migrations under migrations/<namespace>. Step 0 has no database wired
// up, so it only lists what it found and exits successfully.
package main

import (
	"fmt"
	"os"

	"github.com/XoDeR/empops/api-go/pkg/migration"
)

func main() {
	migrationsDir := envOrDefault("EMPOPS_MIGRATIONS_DIR", "migrations")

	migrations, err := migration.Discover(migrationsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migrate: fatal:", err)
		os.Exit(1)
	}

	fmt.Println("migrations: use postgres when configured (Step 0 stub)")
	fmt.Printf("migrations: discovered %d migration(s) under %s\n", len(migrations), migrationsDir)
	for _, m := range migrations {
		fmt.Printf("  - [%s] %s_%s (up=%t down=%t)\n", m.Namespace, m.Version, m.Name, m.UpFile != "", m.DownFile != "")
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
