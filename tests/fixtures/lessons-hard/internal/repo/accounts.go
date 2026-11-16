package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"example.com/crewdesk/internal/platform"
)

var ErrInvalidToken = errors.New("invalid or expired token")

type Accounts struct{ db *sql.DB }

func NewAccounts(db *sql.DB) *Accounts { return &Accounts{db: db} }

type SecurityEvent struct {
	Kind string    `json:"kind"`
	IP   string    `json:"ip"`
	At   time.Time `json:"at"`
}

func (a *Accounts) Allow(ctx context.Context, bucket string, limit int, window time.Duration) (bool, error) {
	if _, err := a.db.ExecContext(ctx,
		"INSERT INTO rate_hits (bucket, hit_at) VALUES (?, NOW())", bucket); err != nil {
		return false, err
	}
	var n int
	err := a.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM rate_hits WHERE bucket = ? AND hit_at > ?", bucket, time.Now().Add(-window)).Scan(&n)
	return n <= limit, err
}

func (a *Accounts) Credentials(ctx context.Context, email string) (int64, string) {
	var userID int64
	var hash string
	err := a.db.QueryRowContext(ctx,
		"SELECT id, password_hash FROM users WHERE email = ? AND active = 1", email).Scan(&userID, &hash)
	if err != nil {
		return 0, ""
	}
	return userID, hash
}

func (a *Accounts) PasswordHash(ctx context.Context, userID int64) (string, error) {
	var hash string
	err := a.db.QueryRowContext(ctx, "SELECT password_hash FROM users WHERE id = ? AND active = 1", userID).Scan(&hash)
	return hash, err
}

func (a *Accounts) DefaultCompany(ctx context.Context, userID int64) (int64, error) {
	var companyID int64
	err := a.db.QueryRowContext(ctx,
		`SELECT m.company_id FROM members m JOIN companies c ON c.id = m.company_id
		  WHERE m.user_id = ? AND m.status = 'active' AND c.status = 'active'
		  ORDER BY m.company_id LIMIT 1`, userID).Scan(&companyID)
	return companyID, err
}

func (a *Accounts) CreateSession(ctx context.Context, userID, companyID int64, token string, expires time.Time) error {
	_, err := a.db.ExecContext(ctx,
		"INSERT INTO sessions (user_id, company_id, token_hash, expires_at) VALUES (?, ?, ?, ?)",
		userID, companyID, platform.Digest(token), expires)
	return err
}

func (a *Accounts) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := a.db.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash = ?", tokenHash)
	return err
}

func (a *Accounts) RecordEvent(ctx context.Context, email, kind, ip string) error {
	_, err := a.db.ExecContext(ctx,
		"INSERT INTO security_events (email_hash, kind, ip, created_at) VALUES (?, ?, ?, NOW())",
		platform.Digest(email), kind, ip)
	return err
}

func (a *Accounts) RecentEvents(ctx context.Context, userID int64) ([]SecurityEvent, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT e.kind, e.ip, e.created_at FROM security_events e
		   JOIN users u ON e.email_hash = SHA2(u.email, 256)
		  WHERE u.id = ? ORDER BY e.created_at DESC LIMIT 50`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SecurityEvent
	for rows.Next() {
		var e SecurityEvent
		if err := rows.Scan(&e.Kind, &e.IP, &e.At); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (a *Accounts) SaveReset(ctx context.Context, userID int64, token string, expires time.Time) error {
	_, err := a.db.ExecContext(ctx,
		"INSERT INTO password_resets (user_id, token, expires_at) VALUES (?, ?, ?)", userID, token, expires)
	return err
}

func (a *Accounts) ConsumeReset(ctx context.Context, token string) (int64, error) {
	res, err := a.db.ExecContext(ctx,
		"UPDATE password_resets SET used_at = NOW() WHERE token = ? AND used_at IS NULL AND expires_at > NOW()", token)
	if err != nil {
		return 0, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return 0, ErrInvalidToken
	}
	var userID int64
	err = a.db.QueryRowContext(ctx, "SELECT user_id FROM password_resets WHERE token = ?", token).Scan(&userID)
	return userID, err
}

func (a *Accounts) SetPassword(ctx context.Context, userID int64, hash string) error {
	_, err := a.db.ExecContext(ctx, "UPDATE users SET password_hash = ? WHERE id = ?", hash, userID)
	return err
}

func (a *Accounts) ChangePassword(ctx context.Context, userID int64, hash, keepSession string) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, "UPDATE users SET password_hash = ? WHERE id = ? AND active = 1", hash, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrInvalidToken
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM sessions WHERE user_id = ? AND token_hash <> ?", userID, keepSession); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE password_resets SET used_at = NOW() WHERE user_id = ? AND used_at IS NULL", userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (a *Accounts) AcceptInvite(ctx context.Context, token, name, hash string) error {
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var inviteID, companyID int64
	var email, role string
	err = tx.QueryRowContext(ctx,
		`SELECT id, company_id, email, role FROM invites
		  WHERE token_hash = ? AND accepted_at IS NULL AND expires_at > NOW() FOR UPDATE`,
		platform.Digest(token)).Scan(&inviteID, &companyID, &email, &role)
	if err != nil {
		return ErrInvalidToken
	}
	res, err := tx.ExecContext(ctx,
		"INSERT INTO users (email, display_name, password_hash) VALUES (?, ?, ?)", email, name, hash)
	if err != nil {
		return err
	}
	userID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO members (company_id, user_id, role) VALUES (?, ?, ?)", companyID, userID, role); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE invites SET accepted_at = NOW() WHERE id = ?", inviteID); err != nil {
		return err
	}
	return tx.Commit()
}
