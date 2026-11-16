package service

import (
	"context"
	"database/sql"
	"errors"

	"example.com/ledger/internal/audit"
	"example.com/ledger/internal/repo"
)

type Summary struct {
	Period   string            `json:"period"`
	Status   string            `json:"status"`
	Receipts int64             `json:"receipts"`
	Payments int64             `json:"payments"`
	Balance  int64             `json:"balance"`
	ByLabel  []repo.LabelTotal `json:"by_label"`
}

type Periods struct {
	db       *sql.DB
	periods  *repo.PeriodRepo
	requests *repo.ExpenseRepo
	audit    *audit.Log
}

func NewPeriods(db *sql.DB, periods *repo.PeriodRepo, requests *repo.ExpenseRepo, trail *audit.Log) *Periods {
	return &Periods{db: db, periods: periods, requests: requests, audit: trail}
}

func (s *Periods) totals(ctx context.Context, q repo.Querier, periodID int64) (receipts, payments int64, err error) {
	if receipts, err = s.periods.Receipts(ctx, q, periodID); err != nil {
		return 0, 0, err
	}
	payments, err = s.periods.Payments(ctx, q, periodID)
	return receipts, payments, err
}

func (s *Periods) Close(ctx context.Context, actor repo.Actor, periodID int64) error {
	return repo.InTx(ctx, s.db, func(tx *sql.Tx) error {
		period, err := s.periods.Lock(ctx, tx, periodID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if period.Closed() {
			return ErrPeriodClosed
		}
		open, err := s.requests.CountOpenInPeriod(ctx, tx, periodID)
		if err != nil {
			return err
		}
		if open > 0 {
			return ErrOpenRequests
		}
		receipts, payments, err := s.totals(ctx, tx, periodID)
		if err != nil {
			return err
		}
		if err := s.periods.MarkClosed(ctx, tx, periodID, actor.ID, receipts-payments); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, actor.ID, "period.close", periodID, audit.Fields{"period": period.Code, "status": "closed"})
	})
}

func (s *Periods) Summary(ctx context.Context, periodID int64) (Summary, error) {
	period, err := s.periods.ByID(ctx, s.db, periodID)
	if errors.Is(err, sql.ErrNoRows) {
		return Summary{}, ErrNotFound
	}
	if err != nil {
		return Summary{}, err
	}
	receipts, payments, err := s.totals(ctx, s.db, periodID)
	if err != nil {
		return Summary{}, err
	}
	byLabel, err := s.periods.ByLabel(ctx, s.db, periodID)
	if err != nil {
		return Summary{}, err
	}
	return Summary{
		Period: period.Code, Status: period.Status,
		Receipts: receipts, Payments: payments, Balance: receipts - payments, ByLabel: byLabel,
	}, nil
}
