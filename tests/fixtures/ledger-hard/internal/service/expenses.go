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

type NewRequest struct {
	FundID int64
	Amount int64
	Reason string
}

type Expenses struct {
	db       *sql.DB
	requests *repo.ExpenseRepo
	funds    *repo.FundRepo
	staff    *repo.StaffRepo
	periods  *repo.PeriodRepo
	audit    *audit.Log
}

func NewExpenses(db *sql.DB, requests *repo.ExpenseRepo, funds *repo.FundRepo, staff *repo.StaffRepo,
	periods *repo.PeriodRepo, trail *audit.Log) *Expenses {
	return &Expenses{db: db, requests: requests, funds: funds, staff: staff, periods: periods, audit: trail}
}

func (s *Expenses) Create(ctx context.Context, actor repo.Actor, in NewRequest) (int64, error) {
	reason := strings.TrimSpace(in.Reason)
	if in.Amount <= 0 || reason == "" || len(reason) > 255 {
		return 0, ErrInvalid
	}
	employee, err := s.staff.EmployeeByUser(ctx, s.db, actor.ID)
	if err != nil || employee.Status != "active" {
		return 0, ErrForbidden
	}
	department, err := s.staff.Department(ctx, s.db, employee.DepartmentID)
	if err != nil {
		return 0, err
	}
	period, err := s.periods.ByDate(ctx, s.db, time.Now())
	if err != nil {
		return 0, err
	}
	if period.Closed() {
		return 0, ErrPeriodClosed
	}
	used, err := s.requests.SumOpenByDepartment(ctx, s.db, department.ID, period.ID)
	if err != nil {
		return 0, err
	}
	if used+in.Amount > department.MonthlyLimit {
		return 0, ErrOverLimit
	}
	id, err := s.requests.Insert(ctx, s.db, repo.ExpenseRequest{
		PeriodID: period.ID, FundID: in.FundID, DepartmentID: department.ID,
		RequesterID: actor.ID, ApproverID: department.HeadID,
		Amount: in.Amount, Reason: reason,
	})
	if err != nil {
		return 0, err
	}
	_ = s.audit.Record(ctx, nil, actor.ID, "expense.create", id, audit.Fields{"department_id": department.ID})
	return id, nil
}

func (s *Expenses) Decide(ctx context.Context, actor repo.Actor, id int64, approve bool) error {
	return repo.InTx(ctx, s.db, func(tx *sql.Tx) error {
		request, err := s.requests.LockByID(ctx, tx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if request.ApproverID != actor.ID || request.RequesterID == actor.ID {
			return ErrForbidden
		}
		if request.Status != "pending" {
			return ErrConflict
		}
		status := "rejected"
		if approve {
			fund, err := s.funds.Lock(ctx, tx, request.FundID)
			if err != nil {
				return err
			}
			if fund.Archived || fund.Balance < request.Amount {
				return ErrInsufficient
			}
			if err := s.funds.AddBalance(ctx, tx, fund.ID, -request.Amount); err != nil {
				return err
			}
			status = "approved"
		}
		if ok, err := s.requests.Transition(ctx, tx, id, "pending", status); err != nil || !ok {
			return errors.Join(ErrConflict, err)
		}
		return s.audit.Record(ctx, tx, actor.ID, "expense.decide", id, audit.Fields{"status": status})
	})
}

func (s *Expenses) CancelOpen(ctx context.Context, actor repo.Actor, fundID int64) (int, error) {
	cancelled := 0
	err := repo.InTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := s.funds.Lock(ctx, tx, fundID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		open, err := s.requests.LockOpenByFund(ctx, tx, fundID)
		if err != nil {
			return err
		}
		var refund int64
		for _, request := range open {
			ok, err := s.requests.Transition(ctx, tx, request.ID, request.Status, "cancelled")
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			if request.Status == "approved" {
				refund += request.Amount
			}
			cancelled++
		}
		if refund > 0 {
			if err := s.funds.AddBalance(ctx, tx, fundID, refund); err != nil {
				return err
			}
		}
		return s.audit.Record(ctx, tx, actor.ID, "expense.cancel_open", fundID, audit.Fields{"fund_id": fundID, "count": cancelled})
	})
	return cancelled, err
}
