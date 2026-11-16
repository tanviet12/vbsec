package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"example.com/ledger/internal/audit"
	"example.com/ledger/internal/repo"
)

type NewEntry struct {
	FundID int64
	Date   time.Time
	Kind   string
	Amount int64
	Memo   string
	Labels []int64
}

type EntryPatch struct {
	Amount int64
	Memo   string
	Labels []int64
}

type Entries struct {
	db      *sql.DB
	entries *repo.EntryRepo
	periods *repo.PeriodRepo
	audit   *audit.Log
}

func NewEntries(db *sql.DB, entries *repo.EntryRepo, periods *repo.PeriodRepo, trail *audit.Log) *Entries {
	return &Entries{db: db, entries: entries, periods: periods, audit: trail}
}

func validKind(kind string) bool { return kind == "receipt" || kind == "payment" }

func (s *Entries) Create(ctx context.Context, actor repo.Actor, in NewEntry) (int64, error) {
	if !validKind(in.Kind) || in.Amount <= 0 || len(in.Memo) > 255 {
		return 0, ErrInvalid
	}
	var id int64
	err := repo.InTx(ctx, s.db, func(tx *sql.Tx) error {
		period, err := s.periods.LockByDate(ctx, tx, in.Date)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if period.Closed() {
			return ErrPeriodClosed
		}
		no, err := s.entries.NextVoucherNo(ctx, tx, period.ID)
		if err != nil {
			return err
		}
		id, err = s.entries.Insert(ctx, tx, repo.Entry{
			PeriodID: period.ID, FundID: in.FundID, VoucherNo: no, Date: in.Date,
			Kind: in.Kind, Amount: in.Amount, Memo: strings.TrimSpace(in.Memo),
			Source: "manual", CreatedBy: actor.ID,
		})
		if err != nil {
			return err
		}
		if err := s.entries.ReplaceLabels(ctx, tx, id, in.Labels); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, actor.ID, "entry.create", id, audit.Fields{"period": period.Code, "kind": in.Kind})
	})
	return id, err
}

func (s *Entries) openForWrite(ctx context.Context, tx *sql.Tx, id int64) (repo.Entry, repo.Period, error) {
	entry, err := s.entries.Get(ctx, tx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return entry, repo.Period{}, ErrNotFound
	}
	if err != nil {
		return entry, repo.Period{}, err
	}
	period, err := s.periods.Lock(ctx, tx, entry.PeriodID)
	if err != nil {
		return entry, period, err
	}
	if period.Closed() {
		return entry, period, ErrPeriodClosed
	}
	return entry, period, nil
}

func (s *Entries) Update(ctx context.Context, actor repo.Actor, id int64, patch EntryPatch) error {
	if patch.Amount <= 0 || len(patch.Memo) > 255 {
		return ErrInvalid
	}
	return repo.InTx(ctx, s.db, func(tx *sql.Tx) error {
		entry, period, err := s.openForWrite(ctx, tx, id)
		if err != nil {
			return err
		}
		if entry.Deleted {
			return ErrNotFound
		}
		if err := s.entries.UpdateFields(ctx, tx, id, patch.Amount, strings.TrimSpace(patch.Memo)); err != nil {
			return err
		}
		if patch.Labels != nil {
			if err := s.entries.ReplaceLabels(ctx, tx, id, patch.Labels); err != nil {
				return err
			}
		}
		return s.audit.Record(ctx, tx, actor.ID, "entry.update", id, audit.Fields{"period": period.Code})
	})
}

func (s *Entries) Delete(ctx context.Context, actor repo.Actor, id int64) error {
	return repo.InTx(ctx, s.db, func(tx *sql.Tx) error {
		_, period, err := s.openForWrite(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := s.entries.SoftDelete(ctx, tx, id); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, actor.ID, "entry.delete", id, audit.Fields{"period": period.Code})
	})
}

func (s *Entries) Restore(ctx context.Context, actor repo.Actor, id int64) error {
	entry, err := s.entries.Get(ctx, s.db, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !entry.Deleted {
		return ErrConflict
	}
	if err := s.entries.Undelete(ctx, s.db, id); err != nil {
		return err
	}
	return s.audit.Record(ctx, nil, actor.ID, "entry.restore", id, audit.Fields{"kind": entry.Kind})
}
