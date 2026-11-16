package service

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"example.com/crewdesk/internal/authn"
)

type ActivityRow struct {
	ActorID int64     `json:"actor_id"`
	Action  string    `json:"action"`
	Details string    `json:"details"`
	At      time.Time `json:"at"`
}

type Activity struct{ db *sql.DB }

func NewActivity(db *sql.DB) *Activity { return &Activity{db: db} }

func likeEscape(q string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
}

func (s *Activity) Search(ctx context.Context, actor authn.Principal, from, to time.Time, q string, limit int) ([]ActivityRow, error) {
	if !actor.IsAdmin() {
		return nil, ErrForbidden
	}
	if to.IsZero() {
		to = time.Now()
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if len(q) > 80 {
		return nil, ErrInvalid
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT actor_id, action, details, created_at FROM activity
		  WHERE company_id = ? AND created_at BETWEEN ? AND ? AND details LIKE ?
		  ORDER BY created_at DESC LIMIT ?`,
		actor.CompanyID, from, to, "%"+likeEscape(q)+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ActivityRow
	for rows.Next() {
		var r ActivityRow
		if err := rows.Scan(&r.ActorID, &r.Action, &r.Details, &r.At); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Activity) Recent(ctx context.Context, actor authn.Principal) ([]ActivityRow, error) {
	return s.Search(ctx, actor, time.Now().AddDate(0, 0, -30), time.Time{}, "", 50)
}
