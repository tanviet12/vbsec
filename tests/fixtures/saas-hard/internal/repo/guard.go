package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// scopedTables hold rows that belong to exactly one company.
var scopedTables = []string{
	"members", "member_permissions", "projects", "project_members", "tasks",
	"comments", "invoices", "expenses", "export_jobs", "action_confirmations",
}

// DB is the handle repositories use for request-scoped work. Every statement
// is checked for a company scope before it reaches the driver.
type DB struct {
	raw *sql.DB
}

func NewDB(raw *sql.DB) *DB {
	return &DB{raw: raw}
}

// System returns the unchecked handle for sign-in and scheduled jobs.
func (d *DB) System() *sql.DB {
	return d.raw
}

func (d *DB) Query(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	if err := checkScope(q); err != nil {
		return nil, err
	}
	return d.raw.QueryContext(ctx, q, args...)
}

func (d *DB) QueryRow(ctx context.Context, q string, args ...any) (*sql.Row, error) {
	if err := checkScope(q); err != nil {
		return nil, err
	}
	return d.raw.QueryRowContext(ctx, q, args...), nil
}

func (d *DB) Exec(ctx context.Context, q string, args ...any) (sql.Result, error) {
	if err := checkScope(q); err != nil {
		return nil, err
	}
	return d.raw.ExecContext(ctx, q, args...)
}

func checkScope(q string) error {
	words := strings.Fields(strings.ToLower(q))
	if len(words) == 0 {
		return nil
	}
	switch words[0] {
	case "select", "insert", "update", "delete":
	default:
		return nil
	}
	norm := " " + strings.Join(words, " ") + " "
	for _, t := range scopedTables {
		if !strings.Contains(norm, " "+t+" ") {
			continue
		}
		if !strings.Contains(norm, "company_id") {
			return fmt.Errorf("repo: statement on %s has no company scope", t)
		}
	}
	return nil
}
