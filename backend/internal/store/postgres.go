package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore wraps a pgxpool connection pool and provides Cinova-specific
// persistence operations for users, sessions, and refresh tokens.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore connects to PostgreSQL using the given DSN and verifies
// connectivity. It also runs schema migrations synchronously.
func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx parse config: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("pgx pool connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pgx pool ping: %w", err)
	}

	s := &PostgresStore{pool: pool}
	if err := s.migrate(ctx); err != nil {
		return nil, fmt.Errorf("postgres migrate: %w", err)
	}
	return s, nil
}

// Close releases all database connections.
func (s *PostgresStore) Close() {
	s.pool.Close()
}

// Pool returns the underlying pgxpool.Pool for raw SQL execution.
func (s *PostgresStore) Pool() *pgxpool.Pool {
	return s.pool
}

// Ping checks database connectivity.
func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// migrate applies idempotent DDL for the Cinova schema.
func (s *PostgresStore) migrate(ctx context.Context) error {
	ddl := `
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    email         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    display_name  TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS anonymous_sessions (
    uuid       TEXT PRIMARY KEY,
    device_id  TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    merged_at  TIMESTAMPTZ,
    user_id    TEXT REFERENCES users(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    session_id TEXT,
    token_hash TEXT UNIQUE NOT NULL,
    anonymous  BOOLEAN NOT NULL DEFAULT FALSE,
    issued_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id    ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_anon_sessions_user_id     ON anonymous_sessions(user_id);
`
	_, err := s.pool.Exec(ctx, ddl)
	return err
}

// ---- User operations ----

// UserRow is a flat row returned from the users table.
type UserRow struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CreateUser inserts a new user and returns the created row.
func (s *PostgresStore) CreateUser(ctx context.Context, id, email, passwordHash, displayName string) (*UserRow, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO users (id, email, password_hash, display_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, password_hash, display_name, created_at, updated_at
	`, id, email, passwordHash, displayName)

	u := &UserRow{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// GetUserByEmail retrieves a user row by email address.
func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*UserRow, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, created_at, updated_at
		FROM users WHERE email = $1
	`, email)

	u := &UserRow{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

// GetUserByID retrieves a user row by primary key.
func (s *PostgresStore) GetUserByID(ctx context.Context, id string) (*UserRow, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, display_name, created_at, updated_at
		FROM users WHERE id = $1
	`, id)

	u := &UserRow{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

// ---- Anonymous session operations ----

// CreateAnonymousSession records a new anonymous session.
func (s *PostgresStore) CreateAnonymousSession(ctx context.Context, uuid, deviceID string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO anonymous_sessions (uuid, device_id)
		VALUES ($1, $2)
		ON CONFLICT (uuid) DO NOTHING
	`, uuid, deviceID)
	return err
}

// MergeAnonymousSession marks an anonymous session as merged into the given
// user account. The caller is responsible for also merging Neo4j graph edges
// (see graph.MergeSessionToUser).
func (s *PostgresStore) MergeAnonymousSession(ctx context.Context, sessionUUID, userID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE anonymous_sessions
		SET merged_at = now(), user_id = $1
		WHERE uuid = $2 AND merged_at IS NULL
	`, userID, sessionUUID)
	return err
}

// GetAnonymousSession retrieves an anonymous session by UUID.
func (s *PostgresStore) GetAnonymousSession(ctx context.Context, uuid string) (*AnonymousSessionRow, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT uuid, device_id, created_at, merged_at, user_id
		FROM anonymous_sessions WHERE uuid = $1
	`, uuid)

	a := &AnonymousSessionRow{}
	err := row.Scan(&a.UUID, &a.DeviceID, &a.CreatedAt, &a.MergedAt, &a.UserID)
	if err != nil {
		return nil, fmt.Errorf("get anonymous session: %w", err)
	}
	return a, nil
}

// AnonymousSessionRow is a flat row from anonymous_sessions.
type AnonymousSessionRow struct {
	UUID      string
	DeviceID  string
	CreatedAt time.Time
	MergedAt  *time.Time
	UserID    *string
}

// ---- Refresh token operations ----

// StoreRefreshToken persists a hashed refresh token.
func (s *PostgresStore) StoreRefreshToken(ctx context.Context, id, userID, sessionID, tokenHash string, anonymous bool, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (id, user_id, session_id, token_hash, anonymous, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, userID, sessionID, tokenHash, anonymous, expiresAt)
	return err
}

// GetRefreshToken retrieves an active (non-revoked, non-expired) refresh token row by hash.
func (s *PostgresStore) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshTokenRow, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, user_id, session_id, anonymous, issued_at, expires_at
		FROM refresh_tokens
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > now()
	`, tokenHash)

	r := &RefreshTokenRow{}
	err := row.Scan(&r.ID, &r.UserID, &r.SessionID, &r.Anonymous, &r.IssuedAt, &r.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return r, nil
}

// RevokeRefreshToken marks a refresh token as revoked by its DB ID.
func (s *PostgresStore) RevokeRefreshToken(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1
	`, id)
	return err
}

// RevokeAllUserRefreshTokens revokes every active token belonging to a user.
func (s *PostgresStore) RevokeAllUserRefreshTokens(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = now()
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID)
	return err
}

// RefreshTokenRow is a flat row from refresh_tokens.
type RefreshTokenRow struct {
	ID        string
	UserID    string
	SessionID string
	Anonymous bool
	IssuedAt  time.Time
	ExpiresAt time.Time
}
