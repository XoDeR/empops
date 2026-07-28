package migration

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Runner applies namespaced SQL migrations to Postgres, tracking what has
// already run in a schema_migrations table (namespace, version).
type Runner struct {
	pool *pgxpool.Pool
}

// NewRunner builds a Runner backed by pool.
func NewRunner(pool *pgxpool.Pool) *Runner {
	return &Runner{pool: pool}
}

// AppliedMigration describes one row of schema_migrations.
type AppliedMigration struct {
	Namespace string
	Version   string
	Name      string
}

const schemaMigrationsDDL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    namespace VARCHAR(64) NOT NULL,
    version VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (namespace, version)
);`

// EnsureSchemaTable creates the schema_migrations bookkeeping table if it
// does not already exist.
func (r *Runner) EnsureSchemaTable(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, schemaMigrationsDDL)
	if err != nil {
		return fmt.Errorf("migration: ensure schema_migrations: %w", err)
	}
	return nil
}

// Applied returns every migration already recorded in schema_migrations.
func (r *Runner) Applied(ctx context.Context) (map[string]AppliedMigration, error) {
	rows, err := r.pool.Query(ctx, "SELECT namespace, version, name FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("migration: query applied: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]AppliedMigration)
	for rows.Next() {
		var m AppliedMigration
		if err := rows.Scan(&m.Namespace, &m.Version, &m.Name); err != nil {
			return nil, fmt.Errorf("migration: scan applied: %w", err)
		}
		applied[m.Namespace+"/"+m.Version] = m
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migration: iterate applied: %w", err)
	}
	return applied, nil
}

// Up applies every pending migration under root, restricted to namespaces
// (in the given order), and returns the migrations it actually ran.
func (r *Runner) Up(ctx context.Context, root string, namespaces []string) ([]Migration, error) {
	if err := r.EnsureSchemaTable(ctx); err != nil {
		return nil, err
	}

	all, err := Discover(root)
	if err != nil {
		return nil, err
	}

	byNamespace := make(map[string][]Migration)
	for _, m := range all {
		byNamespace[m.Namespace] = append(byNamespace[m.Namespace], m)
	}

	applied, err := r.Applied(ctx)
	if err != nil {
		return nil, err
	}

	var ran []Migration
	for _, ns := range namespaces {
		migrations := byNamespace[ns]
		sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })

		for _, m := range migrations {
			key := m.Namespace + "/" + m.Version
			if _, ok := applied[key]; ok {
				continue
			}
			if m.UpFile == "" {
				return ran, fmt.Errorf("migration: %s has no .up.sql file", key)
			}

			if err := r.applyOne(ctx, m); err != nil {
				return ran, err
			}
			ran = append(ran, m)
		}
	}

	return ran, nil
}

func (r *Runner) applyOne(ctx context.Context, m Migration) error {
	sqlBytes, err := os.ReadFile(m.UpFile)
	if err != nil {
		return fmt.Errorf("migration: read %s: %w", m.UpFile, err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("migration: begin tx for %s/%s: %w", m.Namespace, m.Version, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op if committed

	// No arguments -> pgx uses the simple query protocol, which allows
	// multiple ;-separated statements in one Exec call.
	if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("migration: apply %s/%s: %w", m.Namespace, m.Version, err)
	}

	if _, err := tx.Exec(ctx,
		"INSERT INTO schema_migrations (namespace, version, name) VALUES ($1, $2, $3)",
		m.Namespace, m.Version, m.Name,
	); err != nil {
		return fmt.Errorf("migration: record %s/%s: %w", m.Namespace, m.Version, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("migration: commit %s/%s: %w", m.Namespace, m.Version, err)
	}

	return nil
}

// Down rolls back the most recently applied migration in namespace (or the
// single most recent across all given namespaces if namespace is empty).
// steps controls how many migrations to roll back (minimum 1).
func (r *Runner) Down(ctx context.Context, root string, namespaces []string, steps int) ([]Migration, error) {
	if steps < 1 {
		steps = 1
	}
	if err := r.EnsureSchemaTable(ctx); err != nil {
		return nil, err
	}

	all, err := Discover(root)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]Migration, len(all))
	for _, m := range all {
		byKey[m.Namespace+"/"+m.Version] = m
	}

	nsSet := make(map[string]bool, len(namespaces))
	for _, ns := range namespaces {
		nsSet[ns] = true
	}

	rows, err := r.pool.Query(ctx, "SELECT namespace, version, name FROM schema_migrations ORDER BY applied_at DESC")
	if err != nil {
		return nil, fmt.Errorf("migration: query applied: %w", err)
	}
	defer rows.Close()

	var appliedDesc []AppliedMigration
	for rows.Next() {
		var m AppliedMigration
		if err := rows.Scan(&m.Namespace, &m.Version, &m.Name); err != nil {
			return nil, fmt.Errorf("migration: scan applied: %w", err)
		}
		if len(nsSet) == 0 || nsSet[m.Namespace] {
			appliedDesc = append(appliedDesc, m)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var reverted []Migration
	for i := 0; i < steps && i < len(appliedDesc); i++ {
		am := appliedDesc[i]
		key := am.Namespace + "/" + am.Version
		m, ok := byKey[key]
		if !ok || m.DownFile == "" {
			return reverted, fmt.Errorf("migration: %s has no .down.sql file on disk", key)
		}

		sqlBytes, err := os.ReadFile(m.DownFile)
		if err != nil {
			return reverted, fmt.Errorf("migration: read %s: %w", m.DownFile, err)
		}

		tx, err := r.pool.Begin(ctx)
		if err != nil {
			return reverted, fmt.Errorf("migration: begin tx for %s: %w", key, err)
		}

		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			tx.Rollback(ctx) //nolint:errcheck
			return reverted, fmt.Errorf("migration: revert %s: %w", key, err)
		}
		if _, err := tx.Exec(ctx, "DELETE FROM schema_migrations WHERE namespace = $1 AND version = $2", m.Namespace, m.Version); err != nil {
			tx.Rollback(ctx) //nolint:errcheck
			return reverted, fmt.Errorf("migration: unrecord %s: %w", key, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return reverted, fmt.Errorf("migration: commit revert %s: %w", key, err)
		}

		reverted = append(reverted, m)
	}

	return reverted, nil
}
