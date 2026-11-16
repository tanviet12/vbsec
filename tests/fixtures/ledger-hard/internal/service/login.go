package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"example.com/ledger/internal/platform"
	"example.com/ledger/internal/repo"
)

const (
	ticketTTL      = 15 * time.Minute
	sessionTTL     = 12 * time.Hour
	otpWindow      = 15 * time.Minute
	maxOTPFailures = 5

	placeholderHash = "$2a$12$C6UzMDM.H6dfI/f/IKcEeO5uYh1zQ0p8mVQyqQhY0aY1x9b0mJq1u"
)

type Auth struct {
	db       *sql.DB
	identity *repo.IdentityRepo
	box      *platform.Box
	key      []byte
	log      *slog.Logger
}

func NewAuth(db *sql.DB, identity *repo.IdentityRepo, box *platform.Box, key []byte, log *slog.Logger) *Auth {
	return &Auth{db: db, identity: identity, box: box, key: key, log: log}
}

func (a *Auth) Login(ctx context.Context, email, password string) (string, error) {
	u, err := a.identity.UserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		platform.VerifyPassword(placeholderHash, password)
		return "", ErrBadCredentials
	}
	if !platform.VerifyPassword(u.PasswordHash, password) || !u.Active {
		a.log.Warn("login rejected", "user_id", u.ID)
		return "", ErrBadCredentials
	}
	return a.signTicket(u.ID), nil
}

func (a *Auth) signTicket(userID int64) string {
	body := strconv.FormatInt(userID, 10) + "." + strconv.FormatInt(time.Now().Add(ticketTTL).Unix(), 10)
	mac := hmac.New(sha256.New, a.key)
	mac.Write([]byte(body))
	return body + "." + hex.EncodeToString(mac.Sum(nil))
}

func (a *Auth) readTicket(ticket string) (int64, error) {
	parts := strings.Split(ticket, ".")
	if len(parts) != 3 {
		return 0, ErrBadTicket
	}
	mac := hmac.New(sha256.New, a.key)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	want := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(parts[2])) {
		return 0, ErrBadTicket
	}
	expires, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > expires {
		return 0, ErrBadTicket
	}
	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, ErrBadTicket
	}
	return userID, nil
}

func (a *Auth) VerifyCode(ctx context.Context, ticket, code string) (string, error) {
	userID, err := a.readTicket(ticket)
	if err != nil {
		return "", err
	}
	u, err := a.identity.UserByID(ctx, userID)
	if err != nil || !u.Active {
		return "", ErrBadTicket
	}
	failures, err := a.identity.CountOTPFailures(ctx, u.ID, time.Now().Add(-otpWindow))
	if err != nil {
		return "", err
	}
	if failures >= maxOTPFailures {
		return "", ErrLocked
	}
	secret, err := a.box.Open(u.TOTPSecret)
	if err != nil {
		return "", err
	}
	if !platform.TOTPValid(secret, code, time.Now()) {
		if err := a.identity.RecordOTPAttempt(ctx, u.ID, false); err != nil {
			return "", err
		}
		a.log.Warn("second factor rejected", "user_id", u.ID, "failures", failures+1)
		return "", ErrBadCode
	}
	_ = a.identity.RecordOTPAttempt(ctx, u.ID, true)

	token, err := platform.NewToken("ses_")
	if err != nil {
		return "", err
	}
	sessionID, err := a.identity.CreateSession(ctx, u.ID, platform.HashToken(token), time.Now().Add(sessionTTL))
	if err != nil {
		return "", err
	}
	a.log.Info("session issued", "user_id", u.ID, "session_id", sessionID)
	return token, nil
}

func (a *Auth) Logout(ctx context.Context, token string) error {
	return a.identity.DeleteSession(ctx, platform.HashToken(token))
}

func (a *Auth) ChangePassword(ctx context.Context, actor repo.Actor, current, next string) error {
	if len(next) < 12 || len(next) > 72 {
		return ErrInvalid
	}
	u, err := a.identity.UserByID(ctx, actor.ID)
	if err != nil {
		return err
	}
	if !platform.VerifyPassword(u.PasswordHash, current) {
		return ErrBadCredentials
	}
	hash, err := platform.HashPassword(next)
	if err != nil {
		return err
	}
	return repo.InTx(ctx, a.db, func(tx *sql.Tx) error {
		if err := a.identity.SetPassword(ctx, tx, u.ID, hash); err != nil {
			return err
		}
		return a.identity.RevokeAll(ctx, tx, u.ID)
	})
}

func (a *Auth) IssueAPIToken(ctx context.Context, actor repo.Actor, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 {
		return "", ErrInvalid
	}
	token, err := platform.NewToken("pat_")
	if err != nil {
		return "", err
	}
	if _, err := a.identity.CreateAPIToken(ctx, actor.ID, name, platform.HashToken(token)); err != nil {
		return "", err
	}
	a.log.Info("api token issued", "user_id", actor.ID, "name", name)
	return token, nil
}
