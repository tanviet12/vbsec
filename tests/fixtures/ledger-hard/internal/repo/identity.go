package repo

import (
	"context"
	"database/sql"
	"time"
)

type Actor struct {
	ID   int64
	Role string
}

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         string
	TOTPSecret   []byte
	Active       bool
}

type IdentityRepo struct{ db *sql.DB }

func NewIdentityRepo(db *sql.DB) *IdentityRepo { return &IdentityRepo{db: db} }

const userColumns = "id, email, password_hash, role, totp_secret, active"

func scanUser(row *sql.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TOTPSecret, &u.Active)
	return u, err
}

func (r *IdentityRepo) UserByEmail(ctx context.Context, email string) (User, error) {
	return scanUser(r.db.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE email = ?", email))
}

func (r *IdentityRepo) UserByID(ctx context.Context, id int64) (User, error) {
	return scanUser(r.db.QueryRowContext(ctx, "SELECT "+userColumns+" FROM users WHERE id = ?", id))
}

func (r *IdentityRepo) SetPassword(ctx context.Context, q Querier, id int64, hash string) error {
	_, err := q.ExecContext(ctx, "UPDATE users SET password_hash = ? WHERE id = ?", hash, id)
	return err
}

func (r *IdentityRepo) SetActive(ctx context.Context, q Querier, id int64, active bool) error {
	_, err := q.ExecContext(ctx, "UPDATE users SET active = ? WHERE id = ?", active, id)
	return err
}

func (r *IdentityRepo) CountOTPFailures(ctx context.Context, userID int64, since time.Time) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM otp_attempts WHERE user_id = ? AND ok = 0 AND created_at > ?",
		userID, since).Scan(&n)
	return n, err
}

func (r *IdentityRepo) RecordOTPAttempt(ctx context.Context, userID int64, ok bool) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO otp_attempts (user_id, ok) VALUES (?, ?)", userID, ok)
	return err
}

func (r *IdentityRepo) CreateSession(ctx context.Context, userID int64, tokenHash string, expires time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token_hash, expires_at) VALUES (?, ?, ?)",
		userID, tokenHash, expires)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *IdentityRepo) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash = ?", tokenHash)
	return err
}

func (r *IdentityRepo) DeleteSessionsByUser(ctx context.Context, q Querier, userID int64) error {
	_, err := q.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

func (r *IdentityRepo) RevokeAll(ctx context.Context, q Querier, userID int64) error {
	if _, err := q.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", userID); err != nil {
		return err
	}
	_, err := q.ExecContext(ctx, "DELETE FROM api_tokens WHERE user_id = ?", userID)
	return err
}

func (r *IdentityRepo) CreateAPIToken(ctx context.Context, userID int64, name, tokenHash string) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		"INSERT INTO api_tokens (user_id, name, token_hash) VALUES (?, ?, ?)",
		userID, name, tokenHash)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *IdentityRepo) ActorBySession(ctx context.Context, tokenHash string) (Actor, error) {
	var a Actor
	err := r.db.QueryRowContext(ctx,
		`SELECT u.id, u.role FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = ? AND s.expires_at > NOW()`, tokenHash).Scan(&a.ID, &a.Role)
	return a, err
}

func (r *IdentityRepo) ActorByAPIToken(ctx context.Context, tokenHash string) (Actor, error) {
	var a Actor
	err := r.db.QueryRowContext(ctx,
		`SELECT u.id, u.role FROM api_tokens t
		 JOIN users u ON u.id = t.user_id
		 WHERE t.token_hash = ?`, tokenHash).Scan(&a.ID, &a.Role)
	if err == nil {
		_, _ = r.db.ExecContext(ctx, "UPDATE api_tokens SET last_used_at = NOW() WHERE token_hash = ?", tokenHash)
	}
	return a, err
}
