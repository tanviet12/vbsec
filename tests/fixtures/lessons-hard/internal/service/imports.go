package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"example.com/crewdesk/internal/authn"
)

const maxImportRows = 1000

type ExpenseRow struct {
	Date        string `json:"date"`
	Category    string `json:"category"`
	AmountCents int64  `json:"amount_cents"`
}

type Imports struct{ db *sql.DB }

func NewImports(db *sql.DB) *Imports { return &Imports{db: db} }

func (s *Imports) Expenses(ctx context.Context, actor authn.Principal, rows []ExpenseRow) (int, error) {
	if !actor.Can("expenses.write") {
		return 0, ErrForbidden
	}
	if len(rows) == 0 || len(rows) > maxImportRows {
		return 0, ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	known := map[string]int64{}
	for _, row := range rows {
		day, err := time.Parse("2006-01-02", row.Date)
		name := strings.TrimSpace(row.Category)
		if err != nil || row.AmountCents <= 0 || name == "" || len(name) > 80 {
			return 0, ErrInvalid
		}
		categoryID, ok := known[name]
		if !ok {
			if categoryID, err = categoryFor(ctx, tx, actor.CompanyID, name); err != nil {
				return 0, err
			}
			known[name] = categoryID
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO expenses (company_id, category_id, spent_on, amount_cents, created_by)
			 VALUES (?, ?, ?, ?, ?)`, actor.CompanyID, categoryID, day, row.AmountCents, actor.UserID); err != nil {
			return 0, err
		}
	}
	return len(rows), tx.Commit()
}

func categoryFor(ctx context.Context, tx *sql.Tx, companyID int64, name string) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx,
		"SELECT id FROM categories WHERE company_id = ? AND name = ?", companyID, name).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, "INSERT INTO categories (company_id, name) VALUES (?, ?)", companyID, name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
