package agentapi

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"

	"example.com/crewdesk/internal/authn"
	"example.com/crewdesk/internal/platform"
)

var periodPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func (d *Dispatcher) catalog() []Tool {
	return []Tool{
		{Name: "whoami", Description: "Show the connected member.", Access: "read", Gate: "member", Run: d.whoami},
		{Name: "list_activity", Description: "List recent workspace activity.", Access: "read", Gate: "admin", Run: d.listActivity},
		{Name: "list_drafts", Description: "List policy drafts waiting for review.", Access: "read", Permission: "projects.edit", Run: d.listDrafts},
		{Name: "get_draft", Description: "Read one policy draft.", Access: "read", Permission: "projects.edit", Run: d.getDraft},
		{Name: "publish_draft", Description: "Publish a reviewed draft. Needs the approval code.", Access: "write",
			Permission: "projects.edit", Destructive: true, Run: d.publishDraft},
		{Name: "import_payroll", Description: "Record net pay lines for a period.", Access: "write",
			Permission: "expenses.write", Run: d.importPayroll},
	}
}

func (d *Dispatcher) whoami(ctx context.Context, p authn.Principal, conn Connection, _ json.RawMessage) (any, error) {
	return map[string]any{"user_id": p.UserID, "role": p.Role, "client_id": conn.ClientID}, nil
}

func (d *Dispatcher) listActivity(ctx context.Context, p authn.Principal, _ Connection, _ json.RawMessage) (any, error) {
	rows, err := d.db.QueryContext(ctx,
		`SELECT actor_id, action, created_at FROM activity
		  WHERE company_id = ? AND created_at > NOW() - INTERVAL 7 DAY ORDER BY created_at DESC LIMIT 100`, p.CompanyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var actorID int64
		var action, at string
		if err := rows.Scan(&actorID, &action, &at); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"actor_id": actorID, "action": action, "at": at})
	}
	return out, rows.Err()
}

func (d *Dispatcher) listDrafts(ctx context.Context, p authn.Principal, _ Connection, _ json.RawMessage) (any, error) {
	rows, err := d.db.QueryContext(ctx,
		"SELECT id, title FROM drafts WHERE company_id = ? AND status = 'pending' ORDER BY id DESC LIMIT 50", p.CompanyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int64
		var title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"draft_id": id, "title_from_author": external("draft title", title)})
	}
	return out, rows.Err()
}

func newCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (d *Dispatcher) getDraft(ctx context.Context, p authn.Principal, conn Connection, args json.RawMessage) (any, error) {
	var in struct {
		DraftID int64 `json:"draft_id"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, ErrBadArgs
	}
	var title, body, status string
	err := d.db.QueryRowContext(ctx,
		"SELECT title, body, status FROM drafts WHERE id = ? AND company_id = ?", in.DraftID, p.CompanyID).
		Scan(&title, &body, &status)
	if err != nil {
		return nil, ErrBadArgs
	}
	out := map[string]any{
		"draft_id":          in.DraftID,
		"status":            status,
		"title_from_author": external("draft title", title),
		"body_from_author":  external("draft body", body),
	}
	if status == "pending" {
		code, err := newCode()
		if err != nil {
			return nil, err
		}
		if _, err := d.db.ExecContext(ctx,
			`REPLACE INTO draft_codes (draft_id, connection_id, code_hash, expires_at)
			 VALUES (?, ?, ?, NOW() + INTERVAL 10 MINUTE)`, in.DraftID, conn.ID, platform.Digest(code)); err != nil {
			return nil, err
		}
		out["approval_code"] = code
		out["approval_code_expires_in"] = "10 minutes"
	}
	return out, nil
}

func (d *Dispatcher) publishDraft(ctx context.Context, p authn.Principal, conn Connection, args json.RawMessage) (any, error) {
	var in struct {
		DraftID int64  `json:"draft_id"`
		Code    string `json:"approval_code"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, ErrBadArgs
	}
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`DELETE FROM draft_codes
		  WHERE draft_id = ? AND connection_id = ? AND code_hash = ? AND expires_at > NOW()`,
		in.DraftID, conn.ID, platform.Digest(in.Code))
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return nil, ErrForbidden
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE drafts SET status = 'published' WHERE id = ? AND company_id = ? AND status = 'pending'",
		in.DraftID, p.CompanyID); err != nil {
		return nil, err
	}
	return map[string]any{"draft_id": in.DraftID, "status": "published"}, tx.Commit()
}

func (d *Dispatcher) importPayroll(ctx context.Context, p authn.Principal, conn Connection, args json.RawMessage) (any, error) {
	if !d.hasGrant(ctx, conn, "payroll") {
		return nil, ErrForbidden
	}
	var in struct {
		Period string `json:"period"`
		Lines  []struct {
			UserID   int64 `json:"user_id"`
			NetCents int64 `json:"net_cents"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(args, &in); err != nil || !periodPattern.MatchString(in.Period) || len(in.Lines) == 0 || len(in.Lines) > 500 {
		return nil, ErrBadArgs
	}
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	for _, line := range in.Lines {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO payroll_lines (company_id, user_id, period, net_cents)
			 SELECT company_id, user_id, ?, ? FROM members WHERE company_id = ? AND user_id = ? AND status = 'active'`,
			in.Period, line.NetCents, p.CompanyID, line.UserID)
		if err != nil {
			return nil, err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return nil, ErrBadArgs
		}
	}
	return map[string]any{"period": in.Period, "lines": len(in.Lines)}, tx.Commit()
}
