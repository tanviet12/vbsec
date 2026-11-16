package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"

	"example.com/ledger/internal/repo"
)

type BankEvent struct {
	Ref    string `json:"ref"`
	FundID int64  `json:"fund_id"`
	Amount int64  `json:"amount"`
}

type PayoutEvent struct {
	Provider  string `json:"provider"`
	EventID   string `json:"event_id"`
	RequestID int64  `json:"request_id"`
}

type Payments struct {
	db       *sql.DB
	funds    *repo.FundRepo
	requests *repo.ExpenseRepo
	secret   []byte
}

func NewPayments(db *sql.DB, funds *repo.FundRepo, requests *repo.ExpenseRepo, secret []byte) *Payments {
	return &Payments{db: db, funds: funds, requests: requests, secret: secret}
}

func (s *Payments) Authentic(body []byte, signature string) bool {
	if len(s.secret) == 0 {
		return false
	}
	mac := hmac.New(sha256.New, s.secret)
	mac.Write(body)
	return hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(signature))
}

func (s *Payments) ApplyBankEvent(ctx context.Context, ev BankEvent) error {
	if ev.Ref == "" || len(ev.Ref) > 64 || ev.Amount <= 0 {
		return ErrInvalid
	}
	return repo.InTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := s.funds.RecordBankEvent(ctx, tx, ev.Ref, ev.FundID, ev.Amount); err != nil {
			if repo.IsDuplicate(err) {
				return nil
			}
			return err
		}
		if _, err := s.funds.Lock(ctx, tx, ev.FundID); err != nil {
			return notFound(err)
		}
		return s.funds.AddBalance(ctx, tx, ev.FundID, ev.Amount)
	})
}

func (s *Payments) ApplyPayoutEvent(ctx context.Context, ev PayoutEvent) error {
	if ev.Provider == "" || len(ev.Provider) > 32 || ev.EventID == "" || len(ev.EventID) > 64 {
		return ErrInvalid
	}
	return repo.InTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := s.funds.RecordPayoutReceipt(ctx, tx, ev.Provider, ev.EventID, ev.RequestID); err != nil {
			if repo.IsDuplicate(err) {
				return nil
			}
			return err
		}
		ok, err := s.requests.Transition(ctx, tx, ev.RequestID, "approved", "paid")
		if err != nil {
			return err
		}
		if !ok {
			return ErrConflict
		}
		return nil
	})
}
