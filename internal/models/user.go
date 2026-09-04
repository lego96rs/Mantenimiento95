package models

import (
	"context"
	"database/sql"
	"fmt"

	"mantenimiento/internal/db"
)

const (
	RoleAdmin      = "admin"
	RolePlanner    = "planner"
	RoleSupervisor = "supervisor"
	RoleTechnician = "technician"
	RoleViewer     = "viewer"
)

type User struct {
	ID                 int64
	Username           string
	DisplayName        string
	Role               string
	Active             bool
	MustChangePassword bool
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func UserByUsername(ctx context.Context, d *db.DB, username string) (*User, string, error) {
	var user User
	var hash string
	err := d.Read.QueryRowContext(ctx,
		`SELECT id, username, password_hash, display_name, role, active, must_change_password
		 FROM users
		 WHERE username = ?`,
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&hash,
		&user.DisplayName,
		&user.Role,
		&user.Active,
		&user.MustChangePassword,
	)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("models: user by username: %w", err)
	}
	return &user, hash, nil
}

func UserByID(ctx context.Context, d *db.DB, id int64) (*User, error) {
	var user User
	err := d.Read.QueryRowContext(ctx,
		`SELECT id, username, display_name, role, active, must_change_password
		 FROM users
		 WHERE id = ?`,
		id,
	).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.Role,
		&user.Active,
		&user.MustChangePassword,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("models: user by id: %w", err)
	}
	return &user, nil
}

func CreateUser(ctx context.Context, d *db.DB, username, displayName, role, passwordHash string, mustChange bool) (int64, error) {
	res, err := d.Write.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, display_name, role, must_change_password)
		 VALUES (?, ?, ?, ?, ?)`,
		username, passwordHash, displayName, role, boolToInt(mustChange),
	)
	if err != nil {
		return 0, fmt.Errorf("models: create user: %w", err)
	}
	return res.LastInsertId()
}

func UpdatePassword(ctx context.Context, d *db.DB, userID int64, passwordHash string, mustChange bool) error {
	res, err := d.Write.ExecContext(ctx,
		`UPDATE users
		 SET password_hash = ?, must_change_password = ?
		 WHERE id = ?`,
		passwordHash, boolToInt(mustChange), userID,
	)
	if err != nil {
		return fmt.Errorf("models: update password: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("models: update password: user %d not found", userID)
	}
	return nil
}

func CountAdmins(ctx context.Context, d *db.DB) (int, error) {
	var count int
	err := d.Read.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE role = ? AND active = 1`,
		RoleAdmin,
	).Scan(&count)
	return count, err
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
