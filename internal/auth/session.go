package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"mantenimiento/internal/db"
	"mantenimiento/internal/models"
)

const SessionTTL = 12 * time.Hour
const CookieName = "msession"

const timeLayout = "2006-01-02T15:04:05.000Z"

type Session struct {
	User models.User
	CSRF string
}

type Sessions struct {
	db *db.DB
}

func NewSessions(d *db.DB) *Sessions {
	return &Sessions{db: d}
}

func newToken() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("auth: read token: %w", err)
	}

	raw = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:]), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (s *Sessions) Create(ctx context.Context, userID int64, ip, userAgent string) (string, error) {
	raw, hash, err := newToken()
	if err != nil {
		return "", err
	}

	csrfBytes := make([]byte, 32)
	if _, err := rand.Read(csrfBytes); err != nil {
		return "", fmt.Errorf("auth: read csrf token: %w", err)
	}

	expiresAt := time.Now().UTC().Add(SessionTTL).Format(timeLayout)
	_, err = s.db.Write.ExecContext(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at, ip, user_agent, csrf_token)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		hash, userID, expiresAt, ip, userAgent, base64.RawURLEncoding.EncodeToString(csrfBytes),
	)
	if err != nil {
		return "", fmt.Errorf("auth: insert session: %w", err)
	}

	return raw, nil
}

func (s *Sessions) Validate(ctx context.Context, rawToken string) (*Session, error) {
	var session Session
	err := s.db.Read.QueryRowContext(ctx,
		`SELECT s.csrf_token, u.id, u.username, u.display_name, u.role, u.active, u.must_change_password
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = ?
		   AND s.expires_at > strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		   AND u.active = 1`,
		hashToken(rawToken),
	).Scan(
		&session.CSRF,
		&session.User.ID,
		&session.User.Username,
		&session.User.DisplayName,
		&session.User.Role,
		&session.User.Active,
		&session.User.MustChangePassword,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auth: validate session: %w", err)
	}
	return &session, nil
}

func (s *Sessions) Delete(ctx context.Context, rawToken string) error {
	_, err := s.db.Write.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hashToken(rawToken))
	return err
}

func (s *Sessions) DeleteAllForUser(ctx context.Context, userID int64) error {
	_, err := s.db.Write.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

func (s *Sessions) DeleteExpired(ctx context.Context) (int64, error) {
	res, err := s.db.Write.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at <= strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func SetCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(SessionTTL / time.Second),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
