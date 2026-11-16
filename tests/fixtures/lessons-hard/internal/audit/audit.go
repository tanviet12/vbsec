package audit

import (
	"context"
	"database/sql"
	"encoding/json"
)

type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Log struct{ db *sql.DB }

func New(db *sql.DB) *Log { return &Log{db: db} }

func (l *Log) Record(ctx context.Context, companyID, actorID int64, action string, details map[string]any) error {
	return l.RecordTx(ctx, l.db, companyID, actorID, action, details)
}

func (l *Log) RecordTx(ctx context.Context, ex Execer, companyID, actorID int64, action string, details map[string]any) error {
	raw, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = ex.ExecContext(ctx,
		"INSERT INTO activity (company_id, actor_id, action, details, created_at) VALUES (?, ?, ?, ?, NOW())",
		companyID, actorID, action, string(raw))
	return err
}

func (l *Log) CrossTenant(ctx context.Context, operatorID, companyID int64, kind string) error {
	_, err := l.db.ExecContext(ctx,
		"INSERT INTO ops_access_log (operator_id, company_id, kind, created_at) VALUES (?, ?, ?, NOW())",
		operatorID, companyID, kind)
	return err
}
