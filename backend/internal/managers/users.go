package managers

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"od/internal/postgres"
)

type UsersManager struct {
	db *postgres.DB
}

func NewUsersManager(db *postgres.DB) *UsersManager {
	return &UsersManager{db: db}
}

// userRow is what Login selects from the users table
type userRow struct {
	PasswordHash string `db:"password_hash"`
	Role         string `db:"role"`
}

func (manager *UsersManager) Register(ctx context.Context, username string, password string, role string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}
	if _, err := manager.db.Exec(ctx, "INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3)", username, string(hash), role); err != nil {
		return fmt.Errorf("creating user %s: %w", username, err)
	}
	return nil
}

// Exists reports whether a user with that username exists
func (manager *UsersManager) Exists(ctx context.Context, username string) (bool, error) {
	var rows []userRow
	if err := manager.db.QueryInto(ctx, &rows, "SELECT password_hash, role FROM users WHERE username = $1", username); err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

// Login checks the password and returns the role of the user. A wrong password or an unknown
// user is ErrInvalidCredentials.
func (manager *UsersManager) Login(ctx context.Context, username string, password string) (string, error) {
	var rows []userRow
	if err := manager.db.QueryInto(ctx, &rows, "SELECT password_hash, role FROM users WHERE username = $1", username); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(rows[0].PasswordHash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}
	return rows[0].Role, nil
}
