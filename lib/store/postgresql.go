package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	// PostgreSQL driver
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresqlStore is a storage backend using PostgreSQL
type PostgresqlStore struct {
	db *sql.DB
}

// NewPostgresqlClient creates a new PostgreSQL connection
func NewPostgresqlClient(connStr string) *sql.DB {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		slog.Error("Failed to connect to PostgreSQL", "error", err)
		return nil
	}

	// Create users table with JSON config column
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(255) PRIMARY KEY,
			username VARCHAR(255) NOT NULL,
			plex_username VARCHAR(255) DEFAULT '',
			access_token TEXT NOT NULL,
			refresh_token TEXT NOT NULL,
			token_expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			config JSONB DEFAULT '{}'
		)
	`)
	if err != nil {
		slog.Error("Failed to create users table", "error", err)
		return nil
	}

	// Create username index
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_username_lower ON users (LOWER(username))`)

	return db
}

// NewPostgresqlStore creates a new PostgreSQL-backed store
func NewPostgresqlStore(db *sql.DB) PostgresqlStore {
	return PostgresqlStore{db: db}
}

// Ping verifies database connectivity
func (s PostgresqlStore) Ping() error {
	return s.db.Ping()
}

// WriteUser saves a user to PostgreSQL
func (s PostgresqlStore) WriteUser(user User) error {
	configJSON, err := json.Marshal(user.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO users (id, username, plex_username, access_token, refresh_token, token_expires_at, config)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			username = EXCLUDED.username,
			plex_username = EXCLUDED.plex_username,
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			token_expires_at = EXCLUDED.token_expires_at,
			config = EXCLUDED.config
	`, user.ID, user.Username, user.PlexUsername, user.AccessToken, user.RefreshToken, user.TokenExpiresAt, configJSON)

	if err != nil {
		slog.Error("Failed to write user", "id", user.ID, "error", err)
		return err
	}
	return nil
}

// GetUser loads a user by ID
func (s PostgresqlStore) GetUser(id string) *User {
	var user User
	var configJSON []byte

	err := s.db.QueryRow(`
		SELECT id, username, plex_username, access_token, refresh_token, token_expires_at, config
		FROM users WHERE id = $1
	`, id).Scan(
		&user.ID,
		&user.Username,
		&user.PlexUsername,
		&user.AccessToken,
		&user.RefreshToken,
		&user.TokenExpiresAt,
		&configJSON,
	)

	if err != nil {
		if err != sql.ErrNoRows {
			slog.Debug("Failed to get user", "id", id, "error", err)
		}
		return nil
	}

	if err := json.Unmarshal(configJSON, &user.Config); err != nil {
		slog.Warn("Failed to unmarshal config", "id", id, "error", err)
	}

	user.Store = s
	return &user
}

// GetUserByUsername looks up a user by username
func (s PostgresqlStore) GetUserByUsername(username string) *User {
	var id string
	err := s.db.QueryRow(`
		SELECT id FROM users WHERE LOWER(username) = LOWER($1)
	`, username).Scan(&id)

	if err != nil {
		if err != sql.ErrNoRows {
			slog.Debug("Username not found", "username", username)
		}
		return nil
	}

	return s.GetUser(id)
}

// DeleteUser removes a user
func (s PostgresqlStore) DeleteUser(id string) bool {
	_, err := s.db.Exec(`DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		slog.Error("Failed to delete user", "id", id, "error", err)
		return false
	}
	return true
}
