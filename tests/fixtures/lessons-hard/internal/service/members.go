package service

import (
	"context"
	"database/sql"
	"net/mail"
	"strings"
	"time"

	"example.com/crewdesk/internal/audit"
	"example.com/crewdesk/internal/authn"
	"example.com/crewdesk/internal/platform"
)

type Members struct {
	db      *sql.DB
	audit   *audit.Log
	mail    *platform.Mailer
	baseURL string
}

func NewMembers(db *sql.DB, log *audit.Log, mailer *platform.Mailer, baseURL string) *Members {
	return &Members{db: db, audit: log, mail: mailer, baseURL: baseURL}
}

func roleAllowed(actor authn.Principal, role string) bool {
	switch role {
	case "member":
		return true
	case "admin":
		return actor.IsOwner()
	}
	return false
}

func (s *Members) Invite(ctx context.Context, actor authn.Principal, email, role string) error {
	if !actor.CanManageMembers() || !roleAllowed(actor, role) {
		return ErrForbidden
	}
	addr, err := mail.ParseAddress(strings.TrimSpace(email))
	if err != nil {
		return ErrInvalid
	}
	token, err := platform.NewToken()
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO invites (company_id, email, role, token_hash, invited_by, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		actor.CompanyID, strings.ToLower(addr.Address), role, platform.Digest(token), actor.UserID,
		time.Now().Add(72*time.Hour)); err != nil {
		return err
	}
	link := s.baseURL + "/invite#" + token
	s.mail.Enqueue(platform.Message{
		To: addr.Address, Subject: "invite", Body: platform.InviteBody(actor.CompanyName, actor.Name, link),
	})
	return s.audit.Record(ctx, actor.CompanyID, actor.UserID, "member_invited", map[string]any{"role": role})
}

func (s *Members) Remove(ctx context.Context, actor authn.Principal, targetID int64) error {
	if !actor.CanManageMembers() || targetID == actor.UserID {
		return ErrForbidden
	}
	var role string
	err := s.db.QueryRowContext(ctx,
		"SELECT role FROM members WHERE company_id = ? AND user_id = ? AND status = 'active'",
		actor.CompanyID, targetID).Scan(&role)
	if err != nil {
		return ErrNotFound
	}
	if role == "owner" || (role == "admin" && !actor.IsOwner()) {
		return ErrForbidden
	}
	if _, err := s.db.ExecContext(ctx,
		"UPDATE members SET status = 'left' WHERE company_id = ? AND user_id = ?", actor.CompanyID, targetID); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx,
		"DELETE FROM sessions WHERE company_id = ? AND user_id = ?", actor.CompanyID, targetID); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx,
		"UPDATE agent_connections SET revoked_at = NOW() WHERE company_id = ? AND user_id = ? AND revoked_at IS NULL",
		actor.CompanyID, targetID); err != nil {
		return err
	}
	return s.audit.Record(ctx, actor.CompanyID, actor.UserID, "member_removed", map[string]any{"user_id": targetID})
}

func (s *Members) Reinstate(ctx context.Context, actor authn.Principal, targetID int64, role string) error {
	if !actor.CanManageMembers() || !roleAllowed(actor, role) {
		return ErrForbidden
	}
	res, err := s.db.ExecContext(ctx,
		"UPDATE members SET status = 'active', role = ? WHERE company_id = ? AND user_id = ? AND status = 'left'",
		role, actor.CompanyID, targetID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrNotFound
	}
	return s.audit.Record(ctx, actor.CompanyID, actor.UserID, "member_reinstated",
		map[string]any{"user_id": targetID, "role": role})
}

func (s *Members) SetPermissions(ctx context.Context, actor authn.Principal, targetID int64, permissions []string) error {
	if !actor.IsOwner() {
		return ErrForbidden
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var one int
	if err := tx.QueryRowContext(ctx,
		"SELECT 1 FROM members WHERE company_id = ? AND user_id = ? AND status = 'active' FOR UPDATE",
		actor.CompanyID, targetID).Scan(&one); err != nil {
		return ErrNotFound
	}
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM member_permissions WHERE company_id = ? AND user_id = ?", actor.CompanyID, targetID); err != nil {
		return err
	}
	for _, name := range permissions {
		if !knownPermissions[name] {
			return ErrInvalid
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO member_permissions (company_id, user_id, permission) VALUES (?, ?, ?)",
			actor.CompanyID, targetID, name); err != nil {
			return err
		}
	}
	if err := s.audit.RecordTx(ctx, tx, actor.CompanyID, actor.UserID, "permissions_set",
		map[string]any{"user_id": targetID, "count": len(permissions)}); err != nil {
		return err
	}
	return tx.Commit()
}

var knownPermissions = map[string]bool{
	"projects.edit":    true,
	"finance.view":     true,
	"requests.create":  true,
	"requests.approve": true,
	"expenses.write":   true,
}

func (s *Members) ResetTwoFactor(ctx context.Context, actor authn.Principal, targetID int64) error {
	if !actor.CanManageMembers() {
		return ErrForbidden
	}
	var email string
	err := s.db.QueryRowContext(ctx,
		`SELECT u.email FROM members m JOIN users u ON u.id = m.user_id
		  WHERE m.company_id = ? AND m.user_id = ? AND (m.role <> 'owner' OR ? = 'owner')`,
		actor.CompanyID, targetID, actor.Role).Scan(&email)
	if err != nil {
		return ErrNotFound
	}
	if _, err := s.db.ExecContext(ctx, "DELETE FROM two_factor WHERE user_id = ?", targetID); err != nil {
		return err
	}
	s.mail.Enqueue(platform.Message{
		To: email, Subject: "security", Body: platform.TwoFactorResetBody(actor.CompanyName, actor.Name),
	})
	return s.audit.Record(ctx, actor.CompanyID, actor.UserID, "two_factor_reset", map[string]any{"user_id": targetID})
}

func (s *Members) UpdateProfile(ctx context.Context, actor authn.Principal, patch map[string]any) error {
	name, _ := patch["display_name"].(string)
	phone, _ := patch["phone"].(string)
	bank, _ := patch["bank_account"].(string)
	if len(name) > 160 || len(phone) > 32 || len(bank) > 64 {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET display_name = COALESCE(NULLIF(?, ''), display_name),
		        phone = COALESCE(NULLIF(?, ''), phone), bank_account = COALESCE(NULLIF(?, ''), bank_account)
		  WHERE id = ?`, name, phone, bank, actor.UserID); err != nil {
		return err
	}
	if err := s.audit.RecordTx(ctx, tx, actor.CompanyID, actor.UserID, "profile_updated",
		map[string]any{"changes": patch}); err != nil {
		return err
	}
	return tx.Commit()
}
