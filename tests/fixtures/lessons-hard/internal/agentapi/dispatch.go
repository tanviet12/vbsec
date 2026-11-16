package agentapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"example.com/crewdesk/internal/authn"
	"example.com/crewdesk/internal/platform"
)

var (
	ErrUnknownTool = errors.New("unknown tool")
	ErrScope       = errors.New("connection scope does not allow this tool")
	ErrForbidden   = errors.New("forbidden")
	ErrBadArgs     = errors.New("invalid arguments")
)

type Connection struct {
	ID        int64
	CompanyID int64
	UserID    int64
	ClientID  string
	scopes    map[string]bool
}

func (c Connection) HasScope(name string) bool { return c.scopes[name] }

type Tool struct {
	Name        string
	Description string
	Access      string
	Gate        string
	Permission  string
	Destructive bool
	Run         func(ctx context.Context, p authn.Principal, conn Connection, args json.RawMessage) (any, error)
}

type Dispatcher struct {
	db    *sql.DB
	tools map[string]Tool
}

func NewDispatcher(db *sql.DB) *Dispatcher {
	d := &Dispatcher{db: db, tools: map[string]Tool{}}
	for _, tool := range d.catalog() {
		if tool.Access != "read" && tool.Access != "write" {
			panic("tool " + tool.Name + " must declare read or write access")
		}
		d.tools[tool.Name] = tool
	}
	return d
}

func (d *Dispatcher) Authenticate(ctx context.Context, bearer string) (Connection, error) {
	conn := Connection{scopes: map[string]bool{}}
	var scopes string
	err := d.db.QueryRowContext(ctx,
		`SELECT id, company_id, user_id, client_id, scopes FROM agent_connections
		  WHERE token_hash = ? AND revoked_at IS NULL`, platform.Digest(bearer)).
		Scan(&conn.ID, &conn.CompanyID, &conn.UserID, &conn.ClientID, &scopes)
	if err != nil {
		return Connection{}, authn.ErrNoSession
	}
	for _, name := range strings.Split(scopes, ",") {
		conn.scopes[strings.TrimSpace(name)] = true
	}
	return conn, nil
}

func (d *Dispatcher) Call(ctx context.Context, conn Connection, name string, args json.RawMessage) (any, error) {
	tool, ok := d.tools[name]
	if !ok {
		return nil, ErrUnknownTool
	}
	p, err := authn.Load(ctx, d.db, conn.CompanyID, conn.UserID)
	if err != nil {
		return nil, err
	}
	switch tool.Gate {
	case "member":
		return tool.Run(ctx, p, conn, args)
	case "admin":
		if !p.IsAdmin() {
			return nil, ErrForbidden
		}
		return tool.Run(ctx, p, conn, args)
	}
	if !conn.HasScope(tool.Access) {
		return nil, ErrScope
	}
	if !p.Can(tool.Permission) {
		return nil, ErrForbidden
	}
	return tool.Run(ctx, p, conn, args)
}

func (d *Dispatcher) Connect(ctx context.Context, p authn.Principal, clientID string, scopes []string) (string, error) {
	for _, name := range scopes {
		if name != "read" && name != "write" {
			return "", ErrBadArgs
		}
	}
	if clientID == "" || len(clientID) > 64 || len(scopes) == 0 {
		return "", ErrBadArgs
	}
	token, err := platform.NewToken()
	if err != nil {
		return "", err
	}
	_, err = d.db.ExecContext(ctx,
		`INSERT INTO agent_connections (company_id, user_id, client_id, token_hash, scopes, created_at)
		 VALUES (?, ?, ?, ?, ?, NOW())`,
		p.CompanyID, p.UserID, clientID, platform.Digest(token), strings.Join(scopes, ","))
	return token, err
}

func (d *Dispatcher) Disconnect(ctx context.Context, p authn.Principal, connectionID int64) error {
	_, err := d.db.ExecContext(ctx,
		`UPDATE agent_connections SET revoked_at = NOW()
		  WHERE id = ? AND company_id = ? AND user_id = ? AND revoked_at IS NULL`,
		connectionID, p.CompanyID, p.UserID)
	return err
}

func (d *Dispatcher) ApprovePayrollGrant(ctx context.Context, owner authn.Principal, clientID string, userID int64) error {
	if !owner.IsOwner() {
		return ErrForbidden
	}
	_, err := d.db.ExecContext(ctx,
		`INSERT INTO special_grants (company_id, client_id, user_id, grant_name, approved_by, approved_at)
		 SELECT m.company_id, ?, m.user_id, 'payroll', ?, NOW() FROM members m
		  WHERE m.company_id = ? AND m.user_id = ? AND m.status = 'active'
		 ON DUPLICATE KEY UPDATE approved_by = VALUES(approved_by), approved_at = VALUES(approved_at)`,
		clientID, owner.UserID, owner.CompanyID, userID)
	return err
}

func (d *Dispatcher) hasGrant(ctx context.Context, conn Connection, grant string) bool {
	var one int
	err := d.db.QueryRowContext(ctx,
		`SELECT 1 FROM special_grants
		  WHERE company_id = ? AND client_id = ? AND user_id = ? AND grant_name = ?`,
		conn.CompanyID, conn.ClientID, conn.UserID, grant).Scan(&one)
	return err == nil
}
