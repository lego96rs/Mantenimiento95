package db

import (
	"context"
	"path/filepath"
	"testing"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()

	database, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	t.Cleanup(func() { _ = database.Close() })

	if err := database.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	return database
}

func TestMigrateCreatesSchemaAndIsIdempotent(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	if err := database.Migrate(ctx); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	for _, table := range []string{
		"app_metadata",
		"users",
		"sessions",
		"areas",
		"asset_categories",
		"technical_documents",
		"assets",
		"asset_documents",
		"schema_migrations",
	} {
		var count int
		err := database.Read.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`,
			table,
		).Scan(&count)
		if err != nil || count != 1 {
			t.Errorf("table %s: count=%d err=%v, want 1 table", table, count, err)
		}
	}

	var mode string
	if err := database.Write.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}
}

func TestUserConstraints(t *testing.T) {
	database := openTestDB(t)
	ctx := context.Background()

	_, err := database.Write.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, display_name, role) VALUES ('ana', 'x', 'Ana', 'runner')`)
	if err == nil {
		t.Fatal("runner role should be rejected by CHECK constraint")
	}

	_, err = database.Write.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, display_name, role) VALUES ('admin1', 'x', 'Admin Uno', 'admin')`)
	if err != nil {
		t.Fatalf("valid admin rejected: %v", err)
	}

	_, err = database.Write.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, display_name, role) VALUES ('ADMIN1', 'x', 'Duplicado', 'admin')`)
	if err == nil {
		t.Fatal("duplicate username with different case was accepted")
	}
}
