package auth

import (
	"context"
	"path/filepath"
	"testing"

	"mantenimiento/internal/db"
	"mantenimiento/internal/models"
)

func openSessionTestDB(t *testing.T) *db.DB {
	t.Helper()

	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := database.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	return database
}

func insertUser(t *testing.T, database *db.DB, username string, active bool) int64 {
	t.Helper()

	hash, err := HashPassword("clave-segura-123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	id, err := models.CreateUser(context.Background(), database, username, "Nombre "+username, models.RoleTechnician, hash, false)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if !active {
		if _, err := database.Write.Exec(`UPDATE users SET active = 0 WHERE id = ?`, id); err != nil {
			t.Fatalf("deactivate user: %v", err)
		}
	}

	return id
}

func TestSessionLifecycle(t *testing.T) {
	database := openSessionTestDB(t)
	sessions := NewSessions(database)
	ctx := context.Background()
	userID := insertUser(t, database, "tecnico1", true)

	token, err := sessions.Create(ctx, userID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	session, err := sessions.Validate(ctx, token)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if session == nil {
		t.Fatal("fresh session did not validate")
	}
	if session.User.ID != userID || session.User.Username != "tecnico1" {
		t.Fatalf("unexpected session user: %+v", session.User)
	}
	if session.CSRF == "" {
		t.Fatal("session has empty CSRF token")
	}

	if got, err := sessions.Validate(ctx, "no-existe"); err != nil || got != nil {
		t.Fatalf("unknown token validated: sess=%v err=%v", got, err)
	}

	if err := sessions.Delete(ctx, token); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, err := sessions.Validate(ctx, token); err != nil || got != nil {
		t.Fatalf("deleted session still validates: sess=%v err=%v", got, err)
	}
}

func TestExpiredSessionDoesNotValidate(t *testing.T) {
	database := openSessionTestDB(t)
	sessions := NewSessions(database)
	ctx := context.Background()
	userID := insertUser(t, database, "tecnico2", true)

	token, err := sessions.Create(ctx, userID, "", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := database.Write.Exec(
		`UPDATE sessions SET expires_at = '2000-01-01T00:00:00.000Z' WHERE token_hash = ?`,
		hashToken(token),
	); err != nil {
		t.Fatalf("force expiry: %v", err)
	}

	if got, err := sessions.Validate(ctx, token); err != nil || got != nil {
		t.Fatalf("expired session validated: sess=%v err=%v", got, err)
	}

	count, err := sessions.DeleteExpired(ctx)
	if err != nil || count != 1 {
		t.Fatalf("DeleteExpired = %d, %v; want 1, nil", count, err)
	}
}

func TestInactiveUserSessionDoesNotValidate(t *testing.T) {
	database := openSessionTestDB(t)
	sessions := NewSessions(database)
	ctx := context.Background()
	userID := insertUser(t, database, "tecnico3", true)

	token, err := sessions.Create(ctx, userID, "", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err := database.Write.Exec(`UPDATE users SET active = 0 WHERE id = ?`, userID); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	if got, err := sessions.Validate(ctx, token); err != nil || got != nil {
		t.Fatalf("inactive user session validated: sess=%v err=%v", got, err)
	}
}

func TestDeleteAllForUser(t *testing.T) {
	database := openSessionTestDB(t)
	sessions := NewSessions(database)
	ctx := context.Background()
	first := insertUser(t, database, "uno", true)
	second := insertUser(t, database, "dos", true)

	tokenOne, _ := sessions.Create(ctx, first, "", "")
	tokenTwo, _ := sessions.Create(ctx, first, "", "")
	tokenThree, _ := sessions.Create(ctx, second, "", "")

	if err := sessions.DeleteAllForUser(ctx, first); err != nil {
		t.Fatalf("DeleteAllForUser: %v", err)
	}

	for _, token := range []string{tokenOne, tokenTwo} {
		if got, _ := sessions.Validate(ctx, token); got != nil {
			t.Fatal("session survived DeleteAllForUser")
		}
	}
	if got, _ := sessions.Validate(ctx, tokenThree); got == nil {
		t.Fatal("other user session was wrongly revoked")
	}
}

func TestTokenIsStoredHashed(t *testing.T) {
	database := openSessionTestDB(t)
	sessions := NewSessions(database)
	ctx := context.Background()
	userID := insertUser(t, database, "tecnico4", true)

	token, err := sessions.Create(ctx, userID, "", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var count int
	if err := database.Read.QueryRow(
		`SELECT COUNT(*) FROM sessions WHERE token_hash = ?`,
		token,
	).Scan(&count); err != nil {
		t.Fatalf("query raw token: %v", err)
	}
	if count != 0 {
		t.Fatal("raw token found in DB")
	}
}
