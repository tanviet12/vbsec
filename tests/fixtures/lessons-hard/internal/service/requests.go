package service

import (
	"context"
	"database/sql"

	"example.com/crewdesk/internal/audit"
	"example.com/crewdesk/internal/authn"
)

type Approver struct {
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
}

type Requests struct {
	db       *sql.DB
	audit    *audit.Log
	settings *Settings
}

func NewRequests(db *sql.DB, log *audit.Log, settings *Settings) *Requests {
	return &Requests{db: db, audit: log, settings: settings}
}

func (s *Requests) Create(ctx context.Context, actor authn.Principal, amountCents int64) (int64, error) {
	if !actor.Can("requests.create") {
		return 0, ErrForbidden
	}
	if amountCents <= 0 || amountCents > 1_000_000_00 {
		return 0, ErrInvalid
	}
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO requests (company_id, requester_id, amount_cents) VALUES (?, ?, ?)",
		actor.CompanyID, actor.UserID, amountCents)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Requests) Approvers(ctx context.Context, actor authn.Principal) ([]Approver, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT u.id, u.display_name
		   FROM members m
		   JOIN users u ON u.id = m.user_id
		   LEFT JOIN member_permissions p
		     ON p.company_id = m.company_id AND p.user_id = m.user_id AND p.permission = 'requests.approve'
		  WHERE m.company_id = ? AND m.status = 'active' AND u.active = 1 AND m.user_id <> ?
		    AND (m.role = 'owner' OR p.permission IS NOT NULL)
		  ORDER BY u.display_name`, actor.CompanyID, actor.UserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Approver
	for rows.Next() {
		var a Approver
		if err := rows.Scan(&a.UserID, &a.Name); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Requests) Submit(ctx context.Context, actor authn.Principal, requestID, approverID int64) error {
	if !actor.Can("requests.create") {
		return ErrForbidden
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE requests SET status = 'pending', approver_id = ?
		  WHERE id = ? AND company_id = ? AND requester_id = ? AND status = 'draft'`,
		approverID, requestID, actor.CompanyID, actor.UserID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrNotFound
	}
	return s.audit.Record(ctx, actor.CompanyID, actor.UserID, "request_submitted",
		map[string]any{"request_id": requestID})
}

func (s *Requests) Decide(ctx context.Context, actor authn.Principal, requestID int64, approve bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	var approverID sql.NullInt64
	if err := tx.QueryRowContext(ctx,
		"SELECT status, approver_id FROM requests WHERE id = ? AND company_id = ? FOR UPDATE",
		requestID, actor.CompanyID).Scan(&status, &approverID); err != nil {
		return ErrNotFound
	}
	if status != "pending" {
		return ErrConflict
	}
	mode, err := s.settings.approvalMode(ctx, tx, actor.CompanyID)
	if err != nil {
		return err
	}
	if mode == "strict" {
		if !actor.IsOwner() {
			return ErrForbidden
		}
	} else if approverID.Int64 != actor.UserID && !actor.IsAdmin() {
		return ErrForbidden
	}
	next := "rejected"
	if approve {
		next = "approved"
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE requests SET status = ? WHERE id = ? AND company_id = ?", next, requestID, actor.CompanyID); err != nil {
		return err
	}
	if err := s.audit.RecordTx(ctx, tx, actor.CompanyID, actor.UserID, "request_decided",
		map[string]any{"request_id": requestID, "status": next}); err != nil {
		return err
	}
	return tx.Commit()
}
