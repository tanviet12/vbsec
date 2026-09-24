package service

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

var ErrInsufficient = errors.New("insufficient balance")

type Wallet struct{ db *sqlx.DB }

func NewWallet(db *sqlx.DB) *Wallet { return &Wallet{db: db} }

func (w *Wallet) Withdraw(userID, amount int64) error {
	if amount <= 0 {
		return errors.New("invalid amount")
	}
	var balance int64
	if err := w.db.Get(&balance, "SELECT balance FROM wallets WHERE user_id = ?", userID); err != nil {
		return err
	}
	if balance < amount {
		return ErrInsufficient
	}
	if _, err := w.db.Exec("UPDATE wallets SET balance = ? WHERE user_id = ?", balance-amount, userID); err != nil {
		return err
	}
	_, err := w.db.Exec("INSERT INTO payouts (user_id, amount) VALUES (?, ?)", userID, amount)
	return err
}
