package repo

import "context"

type Fund struct {
	ID       int64
	Name     string
	Balance  int64
	Archived bool
}

type FundRepo struct{}

func NewFundRepo() *FundRepo { return &FundRepo{} }

func (r *FundRepo) Lock(ctx context.Context, q Querier, id int64) (Fund, error) {
	var f Fund
	err := q.QueryRowContext(ctx,
		"SELECT id, name, balance, archived FROM funds WHERE id = ? FOR UPDATE", id).
		Scan(&f.ID, &f.Name, &f.Balance, &f.Archived)
	return f, err
}

func (r *FundRepo) AddBalance(ctx context.Context, q Querier, id, delta int64) error {
	_, err := q.ExecContext(ctx, "UPDATE funds SET balance = balance + ? WHERE id = ?", delta, id)
	return err
}

func (r *FundRepo) SetArchived(ctx context.Context, q Querier, id int64, archived bool) error {
	_, err := q.ExecContext(ctx, "UPDATE funds SET archived = ? WHERE id = ?", archived, id)
	return err
}

func (r *FundRepo) RecordBankEvent(ctx context.Context, q Querier, ref string, fundID, amount int64) error {
	_, err := q.ExecContext(ctx,
		"INSERT INTO bank_events (provider_ref, fund_id, amount) VALUES (?, ?, ?)", ref, fundID, amount)
	return err
}

func (r *FundRepo) RecordPayoutReceipt(ctx context.Context, q Querier, provider, eventID string, requestID int64) error {
	_, err := q.ExecContext(ctx,
		"INSERT INTO payout_receipts (provider, event_id, expense_request_id) VALUES (?, ?, ?)",
		provider, eventID, requestID)
	return err
}
