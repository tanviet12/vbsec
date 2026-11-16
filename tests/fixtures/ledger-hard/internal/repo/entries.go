package repo

import (
	"context"
	"time"
)

type Entry struct {
	ID        int64
	PeriodID  int64
	FundID    int64
	VoucherNo int64
	Date      time.Time
	Kind      string
	Amount    int64
	Memo      string
	Source    string
	CreatedBy int64
	Deleted   bool
}

type EntryRepo struct{}

func NewEntryRepo() *EntryRepo { return &EntryRepo{} }

func (r *EntryRepo) NextVoucherNo(ctx context.Context, q Querier, periodID int64) (int64, error) {
	var n int64
	err := q.QueryRowContext(ctx,
		"SELECT next_no FROM voucher_counters WHERE period_id = ? FOR UPDATE", periodID).Scan(&n)
	if err != nil {
		return 0, err
	}
	_, err = q.ExecContext(ctx, "UPDATE voucher_counters SET next_no = ? WHERE period_id = ?", n+1, periodID)
	return n, err
}

func (r *EntryRepo) Insert(ctx context.Context, q Querier, e Entry) (int64, error) {
	res, err := q.ExecContext(ctx,
		`INSERT INTO entries (period_id, fund_id, voucher_no, entry_date, kind, amount, memo, source, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.PeriodID, e.FundID, e.VoucherNo, e.Date, e.Kind, e.Amount, e.Memo, e.Source, e.CreatedBy)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *EntryRepo) Get(ctx context.Context, q Querier, id int64) (Entry, error) {
	var e Entry
	err := q.QueryRowContext(ctx,
		`SELECT id, period_id, fund_id, voucher_no, entry_date, kind, amount, memo, source, created_by,
		        deleted_at IS NOT NULL
		 FROM entries WHERE id = ?`, id).
		Scan(&e.ID, &e.PeriodID, &e.FundID, &e.VoucherNo, &e.Date, &e.Kind, &e.Amount, &e.Memo, &e.Source,
			&e.CreatedBy, &e.Deleted)
	return e, err
}

func (r *EntryRepo) UpdateFields(ctx context.Context, q Querier, id, amount int64, memo string) error {
	_, err := q.ExecContext(ctx,
		"UPDATE entries SET amount = ?, memo = ? WHERE id = ? AND deleted_at IS NULL", amount, memo, id)
	return err
}

func (r *EntryRepo) SoftDelete(ctx context.Context, q Querier, id int64) error {
	_, err := q.ExecContext(ctx, "UPDATE entries SET deleted_at = NOW() WHERE id = ? AND deleted_at IS NULL", id)
	return err
}

func (r *EntryRepo) Undelete(ctx context.Context, q Querier, id int64) error {
	_, err := q.ExecContext(ctx, "UPDATE entries SET deleted_at = NULL WHERE id = ?", id)
	return err
}

func (r *EntryRepo) ReplaceLabels(ctx context.Context, q Querier, entryID int64, labelIDs []int64) error {
	if _, err := q.ExecContext(ctx, "DELETE FROM entry_labels WHERE entry_id = ?", entryID); err != nil {
		return err
	}
	for _, id := range labelIDs {
		if _, err := q.ExecContext(ctx,
			"INSERT IGNORE INTO entry_labels (entry_id, label_id) VALUES (?, ?)", entryID, id); err != nil {
			return err
		}
	}
	return nil
}
