package service

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"example.com/ledger/internal/audit"
	"example.com/ledger/internal/repo"
)

const maxImportRows = 2000

type Imports struct {
	db      *sql.DB
	entries *repo.EntryRepo
	periods *repo.PeriodRepo
	audit   *audit.Log
}

func NewImports(db *sql.DB, entries *repo.EntryRepo, periods *repo.PeriodRepo, trail *audit.Log) *Imports {
	return &Imports{db: db, entries: entries, periods: periods, audit: trail}
}

type statementRow struct {
	date   time.Time
	kind   string
	amount int64
	memo   string
}

func parseStatement(r io.Reader) ([]statementRow, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = 3
	var rows []statementRow
	for {
		rec, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, ErrInvalid
		}
		if len(rows) >= maxImportRows {
			return nil, ErrInvalid
		}
		date, err := time.Parse("2006-01-02", strings.TrimSpace(rec[0]))
		if err != nil {
			return nil, ErrInvalid
		}
		amount, err := strconv.ParseInt(strings.TrimSpace(rec[1]), 10, 64)
		if err != nil || amount == 0 {
			return nil, ErrInvalid
		}
		row := statementRow{date: date, kind: "receipt", amount: amount, memo: strings.TrimSpace(rec[2])}
		if amount < 0 {
			row.kind, row.amount = "payment", -amount
		}
		if len(row.memo) > 255 {
			row.memo = row.memo[:255]
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (s *Imports) Statement(ctx context.Context, actor repo.Actor, fundID int64, r io.Reader) (int, error) {
	rows, err := parseStatement(r)
	if err != nil {
		return 0, err
	}
	err = repo.InTx(ctx, s.db, func(tx *sql.Tx) error {
		for _, row := range rows {
			period, err := s.periods.ByDate(ctx, tx, row.date)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			if err != nil {
				return err
			}
			no, err := s.entries.NextVoucherNo(ctx, tx, period.ID)
			if err != nil {
				return err
			}
			if _, err := s.entries.Insert(ctx, tx, repo.Entry{
				PeriodID: period.ID, FundID: fundID, VoucherNo: no, Date: row.date,
				Kind: row.kind, Amount: row.amount, Memo: row.memo,
				Source: "import", CreatedBy: actor.ID,
			}); err != nil {
				return err
			}
		}
		return s.audit.Record(ctx, tx, actor.ID, "entry.import", fundID, audit.Fields{"count": len(rows)})
	})
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}
