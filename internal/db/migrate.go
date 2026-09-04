package db

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func (d *DB) Migrate(ctx context.Context) error {
	_, err := d.Write.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	var currentVersion int
	err = d.Write.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("read current version: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		name := entry.Name()
		versionText, _, ok := strings.Cut(name, "_")
		if !ok {
			return fmt.Errorf("migration %q: name must be NNN_description.sql", name)
		}

		version, err := strconv.Atoi(versionText)
		if err != nil {
			return fmt.Errorf("migration %q: bad version prefix: %w", name, err)
		}

		if version <= currentVersion {
			continue
		}

		sqlBytes, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", name, err)
		}

		if err := d.applyMigration(ctx, version, name, string(sqlBytes)); err != nil {
			return fmt.Errorf("apply migration %q: %w", name, err)
		}

		currentVersion = version
	}

	return nil
}

func (d *DB) applyMigration(ctx context.Context, version int, name, sqlText string) error {
	tx, err := d.Write.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, sqlText); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, version, name); err != nil {
		return err
	}

	return tx.Commit()
}
