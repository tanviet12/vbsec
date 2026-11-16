package repo

import "context"

type ExpenseRequest struct {
	ID           int64
	PeriodID     int64
	FundID       int64
	DepartmentID int64
	RequesterID  int64
	ApproverID   int64
	Amount       int64
	Reason       string
	Status       string
}

type ExpenseRepo struct{}

func NewExpenseRepo() *ExpenseRepo { return &ExpenseRepo{} }

const requestColumns = "id, period_id, fund_id, department_id, requester_id, approver_id, amount, reason, status"

func (r *ExpenseRepo) Insert(ctx context.Context, q Querier, e ExpenseRequest) (int64, error) {
	res, err := q.ExecContext(ctx,
		`INSERT INTO expense_requests (period_id, fund_id, department_id, requester_id, approver_id, amount, reason)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.PeriodID, e.FundID, e.DepartmentID, e.RequesterID, e.ApproverID, e.Amount, e.Reason)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ExpenseRepo) SumOpenByDepartment(ctx context.Context, q Querier, departmentID, periodID int64) (int64, error) {
	var total int64
	err := q.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM expense_requests
		 WHERE department_id = ? AND period_id = ? AND status IN ('pending','approved','paid')`,
		departmentID, periodID).Scan(&total)
	return total, err
}

func (r *ExpenseRepo) LockByID(ctx context.Context, q Querier, id int64) (ExpenseRequest, error) {
	var e ExpenseRequest
	err := q.QueryRowContext(ctx,
		"SELECT "+requestColumns+" FROM expense_requests WHERE id = ? FOR UPDATE", id).
		Scan(&e.ID, &e.PeriodID, &e.FundID, &e.DepartmentID, &e.RequesterID, &e.ApproverID, &e.Amount, &e.Reason, &e.Status)
	return e, err
}

func (r *ExpenseRepo) LockOpenByFund(ctx context.Context, q Querier, fundID int64) ([]ExpenseRequest, error) {
	rows, err := q.QueryContext(ctx,
		"SELECT "+requestColumns+` FROM expense_requests
		 WHERE fund_id = ? AND status IN ('pending','approved') ORDER BY id FOR UPDATE`, fundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExpenseRequest
	for rows.Next() {
		var e ExpenseRequest
		if err := rows.Scan(&e.ID, &e.PeriodID, &e.FundID, &e.DepartmentID, &e.RequesterID, &e.ApproverID,
			&e.Amount, &e.Reason, &e.Status); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *ExpenseRepo) Transition(ctx context.Context, q Querier, id int64, from, to string) (bool, error) {
	res, err := q.ExecContext(ctx,
		"UPDATE expense_requests SET status = ?, decided_at = NOW() WHERE id = ? AND status = ?", to, id, from)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (r *ExpenseRepo) CountOpenInPeriod(ctx context.Context, q Querier, periodID int64) (int, error) {
	var n int
	err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM expense_requests r
		 JOIN funds f ON f.id = r.fund_id
		 WHERE r.period_id = ? AND r.status IN ('pending','approved') AND f.archived = 0`,
		periodID).Scan(&n)
	return n, err
}
