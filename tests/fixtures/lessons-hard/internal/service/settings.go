package service

import (
	"context"
	"database/sql"
	"strconv"

	"example.com/crewdesk/internal/audit"
	"example.com/crewdesk/internal/authn"
)

type Settings struct {
	db    *sql.DB
	audit *audit.Log
}

func NewSettings(db *sql.DB, log *audit.Log) *Settings { return &Settings{db: db, audit: log} }

type queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *Settings) approvalMode(ctx context.Context, q queryer, companyID int64) (string, error) {
	var mode string
	err := q.QueryRowContext(ctx,
		"SELECT approval_mode FROM company_settings WHERE company_id = ?", companyID).Scan(&mode)
	return mode, err
}

var settingColumns = map[string]string{
	"approval_mode":  "approval_mode",
	"retention_days": "retention_days",
}

func (s *Settings) change(ctx context.Context, actor authn.Principal, name, value string) error {
	column, ok := settingColumns[name]
	if !ok {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var old string
	if err := tx.QueryRowContext(ctx,
		"SELECT "+column+" FROM company_settings WHERE company_id = ? FOR UPDATE", actor.CompanyID).Scan(&old); err != nil {
		return ErrNotFound
	}
	if old == value {
		return nil
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE company_settings SET "+column+" = ? WHERE company_id = ?", value, actor.CompanyID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO setting_history (company_id, name, old_value, new_value, changed_by, changed_at)
		 VALUES (?, ?, ?, ?, ?, NOW())`, actor.CompanyID, name, old, value, actor.UserID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Settings) SetApprovalMode(ctx context.Context, actor authn.Principal, mode string) error {
	if !actor.CanManageSettings() {
		return ErrForbidden
	}
	if mode != "strict" && mode != "relaxed" {
		return ErrInvalid
	}
	return s.change(ctx, actor, "approval_mode", mode)
}

func (s *Settings) SetRetention(ctx context.Context, actor authn.Principal, days int) error {
	if !actor.IsOwner() {
		return ErrForbidden
	}
	if days < 30 || days > 3650 {
		return ErrInvalid
	}
	return s.change(ctx, actor, "retention_days", strconv.Itoa(days))
}

func (s *Settings) SetMonthlyLimit(ctx context.Context, actor authn.Principal, cents int64) error {
	if !actor.IsOwner() {
		return ErrForbidden
	}
	if cents < 0 {
		return ErrInvalid
	}
	var old int64
	if err := s.db.QueryRowContext(ctx,
		"SELECT monthly_limit_cents FROM company_settings WHERE company_id = ?", actor.CompanyID).Scan(&old); err != nil {
		return ErrNotFound
	}
	if _, err := s.db.ExecContext(ctx,
		"UPDATE company_settings SET monthly_limit_cents = ? WHERE company_id = ?", cents, actor.CompanyID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO setting_history (company_id, name, old_value, new_value, changed_by, changed_at)
		 VALUES (?, 'monthly_limit_cents', ?, ?, ?, NOW())`,
		actor.CompanyID, strconv.FormatInt(old, 10), strconv.FormatInt(cents, 10), actor.UserID)
	return err
}
