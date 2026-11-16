package repo

import (
	"context"
	"time"
)

type Employee struct {
	ID           int64
	UserID       int64
	DepartmentID int64
	FullName     string
	BaseSalary   int64
	Status       string
}

type Department struct {
	ID           int64
	Name         string
	HeadID       int64
	MonthlyLimit int64
}

type StaffRepo struct{}

func NewStaffRepo() *StaffRepo { return &StaffRepo{} }

const employeeColumns = "id, user_id, department_id, full_name, base_salary, status"

func (r *StaffRepo) EmployeeByUser(ctx context.Context, q Querier, userID int64) (Employee, error) {
	var e Employee
	err := q.QueryRowContext(ctx, "SELECT "+employeeColumns+" FROM employees WHERE user_id = ?", userID).
		Scan(&e.ID, &e.UserID, &e.DepartmentID, &e.FullName, &e.BaseSalary, &e.Status)
	return e, err
}

func (r *StaffRepo) LockEmployee(ctx context.Context, q Querier, id int64) (Employee, error) {
	var e Employee
	err := q.QueryRowContext(ctx, "SELECT "+employeeColumns+" FROM employees WHERE id = ? FOR UPDATE", id).
		Scan(&e.ID, &e.UserID, &e.DepartmentID, &e.FullName, &e.BaseSalary, &e.Status)
	return e, err
}

func (r *StaffRepo) Department(ctx context.Context, q Querier, id int64) (Department, error) {
	var d Department
	err := q.QueryRowContext(ctx, "SELECT id, name, head_id, monthly_limit FROM departments WHERE id = ?", id).
		Scan(&d.ID, &d.Name, &d.HeadID, &d.MonthlyLimit)
	return d, err
}

func (r *StaffRepo) SetHead(ctx context.Context, q Querier, departmentID, userID int64) error {
	_, err := q.ExecContext(ctx, "UPDATE departments SET head_id = ? WHERE id = ?", userID, departmentID)
	return err
}

func (r *StaffRepo) UpdatePayroll(ctx context.Context, q Querier, id, salary int64, taxCode, bankAccount []byte) error {
	_, err := q.ExecContext(ctx,
		`UPDATE employees SET base_salary = ?, tax_code = COALESCE(?, tax_code),
		        bank_account = COALESCE(?, bank_account)
		 WHERE id = ?`, salary, taxCode, bankAccount, id)
	return err
}

func (r *StaffRepo) Terminate(ctx context.Context, q Querier, id int64) error {
	_, err := q.ExecContext(ctx,
		"UPDATE employees SET status = 'terminated', terminated_at = NOW() WHERE id = ?", id)
	return err
}

func (r *StaffRepo) CreateInvite(ctx context.Context, q Querier, email, tokenHash string, departmentID, createdBy int64, expires time.Time) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO invites (email, token_hash, department_id, created_by, expires_at)
		 VALUES (?, ?, ?, ?, ?)`, email, tokenHash, departmentID, createdBy, expires)
	return err
}
