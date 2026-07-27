// Package migration provides a minimal namespaced SQL migration runner.
// Step 0 only needs to discover *.up.sql / *.down.sql files per namespace
// (e.g. "core") so cmd/migrate has something real to grow into once a
// database connection is wired up.
package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Migration describes a single discovered migration file pair.
type Migration struct {
	Namespace string
	Version   string
	Name      string
	UpFile    string
	DownFile  string
}

// Discover walks root (e.g. "migrations") and returns every migration
// found under each namespace subdirectory, sorted by version.
func Discover(root string) ([]Migration, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("migration: read dir %s: %w", root, err)
	}

	byKey := map[string]*Migration{}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		namespace := entry.Name()
		nsDir := filepath.Join(root, namespace)

		files, err := os.ReadDir(nsDir)
		if err != nil {
			return nil, fmt.Errorf("migration: read namespace dir %s: %w", nsDir, err)
		}

		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			isUp := strings.HasSuffix(name, ".up.sql")
			isDown := strings.HasSuffix(name, ".down.sql")
			if !isUp && !isDown {
				continue
			}

			base := strings.TrimSuffix(strings.TrimSuffix(name, ".up.sql"), ".down.sql")
			parts := strings.SplitN(base, "_", 2)
			version := parts[0]
			label := base
			if len(parts) == 2 {
				label = parts[1]
			}

			key := namespace + "/" + version
			m, ok := byKey[key]
			if !ok {
				m = &Migration{Namespace: namespace, Version: version, Name: label}
				byKey[key] = m
			}
			if isUp {
				m.UpFile = filepath.Join(nsDir, name)
			} else {
				m.DownFile = filepath.Join(nsDir, name)
			}
		}
	}

	migrations := make([]Migration, 0, len(byKey))
	for _, m := range byKey {
		migrations = append(migrations, *m)
	}
	sort.Slice(migrations, func(i, j int) bool {
		if migrations[i].Namespace != migrations[j].Namespace {
			return migrations[i].Namespace < migrations[j].Namespace
		}
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}
