package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"example.com/ledger/internal/repo"
)

type Fields map[string]any

type Log struct{ db *sql.DB }

func New(db *sql.DB) *Log { return &Log{db: db} }

var recordable = map[string]bool{
	"changed":       true,
	"period":        true,
	"fund_id":       true,
	"department_id": true,
	"status":        true,
	"count":         true,
	"kind":          true,
}

const maxValueLen = 120

func clean(f Fields) map[string]any {
	out := make(map[string]any, len(f))
	for key, value := range f {
		if !recordable[key] {
			continue
		}
		switch v := value.(type) {
		case string:
			if len(v) > maxValueLen {
				v = v[:maxValueLen]
			}
			out[key] = v
		case []string:
			joined := strings.Join(v, ",")
			if len(joined) > maxValueLen {
				joined = joined[:maxValueLen]
			}
			out[key] = joined
		case int, int64, bool:
			out[key] = v
		default:
			out[key] = "(set)"
		}
	}
	return out
}

func (l *Log) Record(ctx context.Context, q repo.Querier, actorID int64, action string, targetID int64, f Fields) error {
	if q == nil {
		q = l.db
	}
	detail, err := json.Marshal(clean(f))
	if err != nil {
		return err
	}
	_, err = q.ExecContext(ctx,
		"INSERT INTO audit_log (actor_id, action, target_id, detail) VALUES (?, ?, ?, ?)",
		actorID, action, targetID, detail)
	return err
}
