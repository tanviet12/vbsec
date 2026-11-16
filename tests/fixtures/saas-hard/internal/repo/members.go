package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"example.com/tallyboard/internal/model"
	"example.com/tallyboard/internal/tenant"
)

type Members struct {
	db *DB
}

func NewMembers(db *DB) *Members { return &Members{db: db} }

const memberCols = `m.id, m.company_id, m.user_id, u.name, u.email, m.role, m.status, COALESCE(m.invite_token, ''), m.invited_at`

func scanMember(s interface{ Scan(...any) error }) (*model.Member, error) {
	var m model.Member
	var invited sql.NullTime
	if err := s.Scan(&m.ID, &m.CompanyID, &m.UserID, &m.Name, &m.Email, &m.Role, &m.Status, &m.InviteToken, &invited); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if invited.Valid {
		m.InvitedAt = &invited.Time
	}
	return &m, nil
}

func (r *Members) List(ctx context.Context, companyID int64) ([]*model.Member, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+memberCols+" FROM members m JOIN users u ON u.id = m.user_id WHERE m.company_id = ? AND m.status <> 'left' ORDER BY u.name",
		companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Member
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Members) Get(ctx context.Context, companyID, id int64) (*model.Member, error) {
	row, err := r.db.QueryRow(ctx,
		"SELECT "+memberCols+" FROM members m JOIN users u ON u.id = m.user_id WHERE m.company_id = ? AND m.id = ?",
		companyID, id)
	if err != nil {
		return nil, err
	}
	return scanMember(row)
}

func (r *Members) SetRole(ctx context.Context, companyID, id int64, role string) error {
	_, err := r.db.Exec(ctx, "UPDATE members SET role = ? WHERE company_id = ? AND id = ?", role, companyID, id)
	return err
}

func (r *Members) Permissions(ctx context.Context, companyID, memberID int64) ([]string, error) {
	rows, err := r.db.Query(ctx,
		"SELECT permission FROM member_permissions WHERE company_id = ? AND member_id = ? ORDER BY permission",
		companyID, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Members) ReplacePermissions(ctx context.Context, companyID, memberID, grantedBy int64, perms []string) error {
	if _, err := r.db.Exec(ctx, "DELETE FROM member_permissions WHERE company_id = ? AND member_id = ?", companyID, memberID); err != nil {
		return err
	}
	for _, p := range perms {
		if _, err := r.db.Exec(ctx,
			"INSERT INTO member_permissions (company_id, member_id, permission, granted_by) VALUES (?, ?, ?, ?)",
			companyID, memberID, p, grantedBy); err != nil {
			return err
		}
	}
	return nil
}

// CopyPermissions gives `to` every permission `from` holds.
func (r *Members) CopyPermissions(ctx context.Context, companyID, from, to, grantedBy int64) error {
	_, err := r.db.Exec(ctx,
		`INSERT IGNORE INTO member_permissions (company_id, member_id, permission, granted_by)
		 SELECT company_id, ?, permission, ? FROM member_permissions WHERE company_id = ? AND member_id = ?`,
		to, grantedBy, companyID, from)
	return err
}

// BySessionToken resolves a bearer token to the member it was issued for.
func (r *Members) BySessionToken(ctx context.Context, tokenHash string, now time.Time) (*tenant.Actor, error) {
	a := &tenant.Actor{Perms: map[string]bool{}}
	err := r.db.System().QueryRowContext(ctx,
		`SELECT m.id, m.company_id, m.user_id, u.email, m.role
		   FROM sessions s
		   JOIN members m ON m.id = s.member_id
		   JOIN users u ON u.id = m.user_id
		  WHERE s.token_hash = ? AND s.expires_at > ? AND m.status = 'active'`,
		tokenHash, now).Scan(&a.MemberID, &a.CompanyID, &a.UserID, &a.Email, &a.Role)
	if err != nil {
		return nil, err
	}
	perms, err := r.Permissions(ctx, a.CompanyID, a.MemberID)
	if err != nil {
		return nil, err
	}
	for _, p := range perms {
		a.Perms[p] = true
	}
	return a, nil
}

// AcceptInvite activates the invited member and opens a first session.
func (r *Members) AcceptInvite(ctx context.Context, inviteToken, sessionHash string, expires time.Time) error {
	tx, err := r.db.System().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var memberID int64
	err = tx.QueryRowContext(ctx,
		"SELECT id FROM members WHERE invite_token = ? AND status = 'invited' FOR UPDATE", inviteToken).Scan(&memberID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE members SET status = 'active', invite_token = NULL WHERE id = ?", memberID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO sessions (token_hash, member_id, kind, expires_at) VALUES (?, ?, 'web', ?)",
		sessionHash, memberID, expires); err != nil {
		return err
	}
	return tx.Commit()
}
