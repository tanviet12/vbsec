package service

import (
	"context"
	"database/sql"
	"errors"

	"example.com/ledger/internal/audit"
	"example.com/ledger/internal/repo"
)

type Funds struct {
	db    *sql.DB
	funds *repo.FundRepo
	audit *audit.Log
}

func NewFunds(db *sql.DB, funds *repo.FundRepo, trail *audit.Log) *Funds {
	return &Funds{db: db, funds: funds, audit: trail}
}

func (s *Funds) Transfer(ctx context.Context, actor repo.Actor, fromID, toID, amount int64) error {
	if fromID == toID || amount <= 0 {
		return ErrInvalid
	}
	return repo.InTx(ctx, s.db, func(tx *sql.Tx) error {
		firstID, secondID := fromID, toID
		if secondID < firstID {
			firstID, secondID = secondID, firstID
		}
		first, err := s.funds.Lock(ctx, tx, firstID)
		if err != nil {
			return notFound(err)
		}
		second, err := s.funds.Lock(ctx, tx, secondID)
		if err != nil {
			return notFound(err)
		}
		source := first
		if source.ID != fromID {
			source = second
		}
		if first.Archived || second.Archived {
			return ErrConflict
		}
		if source.Balance < amount {
			return ErrInsufficient
		}
		if err := s.funds.AddBalance(ctx, tx, fromID, -amount); err != nil {
			return err
		}
		if err := s.funds.AddBalance(ctx, tx, toID, amount); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, actor.ID, "fund.transfer", fromID, audit.Fields{"fund_id": toID})
	})
}

func (s *Funds) SetArchived(ctx context.Context, actor repo.Actor, fundID int64, archived bool) error {
	return repo.InTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := s.funds.Lock(ctx, tx, fundID); err != nil {
			return notFound(err)
		}
		if err := s.funds.SetArchived(ctx, tx, fundID, archived); err != nil {
			return err
		}
		status := "active"
		if archived {
			status = "archived"
		}
		return s.audit.Record(ctx, tx, actor.ID, "fund.archive", fundID, audit.Fields{"status": status})
	})
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
