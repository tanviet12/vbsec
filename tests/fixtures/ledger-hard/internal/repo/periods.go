package repo

import (
	"context"
	"time"
)

type Period struct {
	ID     int64
	Code   string
	Status string
}

func (p Period) Closed() bool { return p.Status != "open" }

type LabelTotal struct {
	Label string `json:"label"`
	Total int64  `json:"total"`
}

type PeriodRepo struct{}

func NewPeriodRepo() *PeriodRepo { return &PeriodRepo{} }

func (r *PeriodRepo) ByDate(ctx context.Context, q Querier, date time.Time) (Period, error) {
	var p Period
	err := q.QueryRowContext(ctx,
		"SELECT id, code, status FROM periods WHERE code = ?", date.Format("2006-01")).
		Scan(&p.ID, &p.Code, &p.Status)
	return p, err
}

func (r *PeriodRepo) ByID(ctx context.Context, q Querier, id int64) (Period, error) {
	var p Period
	err := q.QueryRowContext(ctx, "SELECT id, code, status FROM periods WHERE id = ?", id).
		Scan(&p.ID, &p.Code, &p.Status)
	return p, err
}

func (r *PeriodRepo) Lock(ctx context.Context, q Querier, id int64) (Period, error) {
	var p Period
	err := q.QueryRowContext(ctx,
		"SELECT id, code, status FROM periods WHERE id = ? FOR UPDATE", id).
		Scan(&p.ID, &p.Code, &p.Status)
	return p, err
}

func (r *PeriodRepo) LockByDate(ctx context.Context, q Querier, date time.Time) (Period, error) {
	var p Period
	err := q.QueryRowContext(ctx,
		"SELECT id, code, status FROM periods WHERE code = ? FOR UPDATE", date.Format("2006-01")).
		Scan(&p.ID, &p.Code, &p.Status)
	return p, err
}

func (r *PeriodRepo) MarkClosed(ctx context.Context, q Querier, id, closedBy, balance int64) error {
	_, err := q.ExecContext(ctx,
		`UPDATE periods SET status = 'closed', closed_by = ?, closed_at = NOW(), closing_balance = ?
		 WHERE id = ? AND status = 'open'`, closedBy, balance, id)
	return err
}

func (r *PeriodRepo) Receipts(ctx context.Context, q Querier, periodID int64) (int64, error) {
	var total int64
	err := q.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(e.amount), 0)
		 FROM entries e
		 JOIN entry_labels el ON el.entry_id = e.id
		 JOIN labels l ON l.id = el.label_id
		 WHERE e.period_id = ? AND e.kind = 'receipt' AND e.deleted_at IS NULL AND l.archived = 0`,
		periodID).Scan(&total)
	return total, err
}

func (r *PeriodRepo) Payments(ctx context.Context, q Querier, periodID int64) (int64, error) {
	var total int64
	err := q.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM entries
		 WHERE period_id = ? AND kind = 'payment' AND deleted_at IS NULL`,
		periodID).Scan(&total)
	return total, err
}

func (r *PeriodRepo) ByLabel(ctx context.Context, q Querier, periodID int64) ([]LabelTotal, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT l.name, COALESCE(SUM(e.amount), 0)
		 FROM labels l
		 JOIN entry_labels el ON el.label_id = l.id
		 JOIN entries e ON e.id = el.entry_id
		 WHERE e.period_id = ? AND e.deleted_at IS NULL
		 GROUP BY l.id, l.name ORDER BY l.name`, periodID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LabelTotal
	for rows.Next() {
		var t LabelTotal
		if err := rows.Scan(&t.Label, &t.Total); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
