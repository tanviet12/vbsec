package authn

import (
	"context"
	"database/sql"
	"errors"

	"example.com/crewdesk/internal/platform"
)

var ErrNoSession = errors.New("no active session")

type Principal struct {
	UserID      int64
	CompanyID   int64
	Role        string
	Name        string
	CompanyName string
	SessionHash string
	perms       map[string]bool
}

func (p Principal) IsOwner() bool           { return p.Role == "owner" }
func (p Principal) IsAdmin() bool           { return p.Role == "owner" || p.Role == "admin" }
func (p Principal) CanManageMembers() bool  { return p.IsAdmin() }
func (p Principal) CanManageSettings() bool { return p.IsAdmin() }

func (p Principal) Can(permission string) bool {
	return p.Role == "owner" || p.perms[permission]
}

func Load(ctx context.Context, db *sql.DB, companyID, userID int64) (Principal, error) {
	p := Principal{UserID: userID, CompanyID: companyID, perms: map[string]bool{}}
	err := db.QueryRowContext(ctx,
		`SELECT m.role, u.display_name, c.name
		   FROM members m
		   JOIN users u ON u.id = m.user_id
		   JOIN companies c ON c.id = m.company_id
		  WHERE m.company_id = ? AND m.user_id = ?
		    AND m.status = 'active' AND u.active = 1 AND c.status = 'active'`,
		companyID, userID).Scan(&p.Role, &p.Name, &p.CompanyName)
	if err != nil {
		return Principal{}, ErrNoSession
	}
	rows, err := db.QueryContext(ctx,
		"SELECT permission FROM member_permissions WHERE company_id = ? AND user_id = ?", companyID, userID)
	if err != nil {
		return Principal{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return Principal{}, err
		}
		p.perms[name] = true
	}
	return p, rows.Err()
}

func FromSession(ctx context.Context, db *sql.DB, token string) (Principal, error) {
	hash := platform.Digest(token)
	var userID, companyID int64
	err := db.QueryRowContext(ctx,
		"SELECT user_id, company_id FROM sessions WHERE token_hash = ? AND expires_at > NOW()", hash).
		Scan(&userID, &companyID)
	if err != nil {
		return Principal{}, ErrNoSession
	}
	p, err := Load(ctx, db, companyID, userID)
	p.SessionHash = hash
	return p, err
}
