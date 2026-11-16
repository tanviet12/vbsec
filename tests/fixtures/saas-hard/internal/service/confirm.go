package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"example.com/tallyboard/internal/repo"
	"example.com/tallyboard/internal/tenant"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrInvalid      = errors.New("invalid input")
	ErrConfirmation = errors.New("confirmation required")
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// Confirmations issues one-time codes for destructive actions. The code is
// delivered to the member's mailbox and is bound to one action and target.
type Confirmations struct {
	DB   *repo.DB
	Mail Mailer
}

const confirmationTTL = 10 * time.Minute

func hashCode(memberID int64, code string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s", memberID, code)))
	return hex.EncodeToString(sum[:])
}

func (c *Confirmations) Issue(ctx context.Context, a *tenant.Actor, action string, target int64, summary string) (time.Time, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return time.Time{}, err
	}
	code := fmt.Sprintf("%06d", n.Int64())
	expires := time.Now().UTC().Add(confirmationTTL)
	_, err = c.DB.Exec(ctx,
		`INSERT INTO action_confirmations (company_id, member_id, action, target, code_hash, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		a.CompanyID, a.MemberID, action, target, hashCode(a.MemberID, code), expires)
	if err != nil {
		return time.Time{}, err
	}
	body := fmt.Sprintf("You asked to %s.\n\nConfirmation code: %s\nThe code expires in 10 minutes. If this was not you, ignore this message.", summary, code)
	if err := c.Mail.Send(ctx, a.Email, "Confirm your request", body); err != nil {
		return time.Time{}, err
	}
	return expires, nil
}

func (c *Confirmations) Verify(ctx context.Context, a *tenant.Actor, action string, target int64, code string) error {
	now := time.Now().UTC()
	res, err := c.DB.Exec(ctx,
		`UPDATE action_confirmations SET attempts = attempts + 1
		  WHERE company_id = ? AND member_id = ? AND action = ? AND target = ?
		    AND used_at IS NULL AND expires_at > ? AND attempts < 5`,
		a.CompanyID, a.MemberID, action, target, now)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrConfirmation
	}
	row, err := c.DB.QueryRow(ctx,
		`SELECT id, code_hash FROM action_confirmations
		  WHERE company_id = ? AND member_id = ? AND action = ? AND target = ?
		    AND used_at IS NULL AND expires_at > ?
		  ORDER BY id DESC LIMIT 1`,
		a.CompanyID, a.MemberID, action, target, now)
	if err != nil {
		return err
	}
	var id int64
	var want string
	if err := row.Scan(&id, &want); err != nil {
		return ErrConfirmation
	}
	if subtle.ConstantTimeCompare([]byte(want), []byte(hashCode(a.MemberID, code))) != 1 {
		return ErrConfirmation
	}
	res, err = c.DB.Exec(ctx,
		"UPDATE action_confirmations SET used_at = ? WHERE company_id = ? AND id = ? AND used_at IS NULL",
		now, a.CompanyID, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrConfirmation
	}
	return nil
}
