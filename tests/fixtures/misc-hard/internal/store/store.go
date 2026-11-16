package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrBadCredentials = errors.New("invalid credentials")

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }

func NewToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func digest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Store) CheckPassword(ctx context.Context, email, password string) (int64, error) {
	var userID int64
	var hash string
	err := s.db.QueryRowContext(ctx,
		"SELECT id, password_hash FROM users WHERE email = ? AND active = 1", email).Scan(&userID, &hash)
	if err != nil {
		return 0, ErrBadCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return 0, ErrBadCredentials
	}
	return userID, nil
}

func (s *Store) Allow(ctx context.Context, bucket string, limit int, window time.Duration) (bool, error) {
	if _, err := s.db.ExecContext(ctx,
		"INSERT INTO rate_hits (bucket, hit_at) VALUES (?, NOW())", bucket); err != nil {
		return false, err
	}
	var n int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM rate_hits WHERE bucket = ? AND hit_at > ?", bucket, time.Now().Add(-window)).Scan(&n)
	return n <= limit, err
}

func (s *Store) CreateSession(ctx context.Context, userID int64, token string, expires time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token_hash, expires_at) VALUES (?, ?, ?)", userID, digest(token), expires)
	return err
}

func (s *Store) SessionUser(ctx context.Context, token string) (int64, error) {
	var userID int64
	err := s.db.QueryRowContext(ctx,
		`SELECT s.user_id FROM sessions s JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = ? AND s.expires_at > NOW() AND u.active = 1`, digest(token)).Scan(&userID)
	return userID, err
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash = ?", digest(token))
	return err
}

func (s *Store) SaveResetToken(ctx context.Context, email, token string, expires time.Time) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO password_resets (user_id, token_hash, expires_at)
		 SELECT id, ?, ? FROM users WHERE email = ? AND active = 1`, digest(token), expires, email)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (s *Store) SaveFeedback(ctx context.Context, userID int64, topic, message string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO feedback (user_id, topic, message) VALUES (?, ?, ?)", userID, topic, message)
	return err
}

func (s *Store) AddDocument(ctx context.Context, userID int64, name string, size int64) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO documents (user_id, name, size) VALUES (?, ?, ?)", userID, name, size)
	return err
}
