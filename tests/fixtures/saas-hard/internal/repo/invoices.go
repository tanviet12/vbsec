package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"example.com/tallyboard/internal/model"
)

type Invoices struct {
	db *DB
}

func NewInvoices(db *DB) *Invoices { return &Invoices{db: db} }

type InvoiceFilter struct {
	Status string
	Sort   string
	Limit  int
	Offset int
}

const invoiceCols = `id, company_id, project_code, number, customer_name, customer_email, COALESCE(customer_note, ''), amount_cents, status, due_date, issued_at, voided_at`

func scanInvoice(s interface{ Scan(...any) error }) (*model.Invoice, error) {
	var inv model.Invoice
	var voided sql.NullTime
	err := s.Scan(&inv.ID, &inv.CompanyID, &inv.ProjectCode, &inv.Number, &inv.CustomerName, &inv.CustomerEmail,
		&inv.CustomerNote, &inv.AmountCents, &inv.Status, &inv.DueDate, &inv.IssuedAt, &voided)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if voided.Valid {
		inv.VoidedAt = &voided.Time
	}
	return &inv, nil
}

func collectInvoices(rows *sql.Rows) ([]*model.Invoice, error) {
	defer rows.Close()
	var out []*model.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

var sortExpr = regexp.MustCompile(`^[a-z_]+( (asc|desc))?`)

func orderClause(sort string) string {
	sort = strings.ToLower(strings.TrimSpace(sort))
	if sort == "" || !sortExpr.MatchString(sort) {
		return "issued_at desc"
	}
	return sort
}

func (r *Invoices) Search(ctx context.Context, companyID int64, f InvoiceFilter) ([]*model.Invoice, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	where := "company_id = ?"
	args := []any{companyID}
	if f.Status != "" {
		where += " AND status = ?"
		args = append(args, f.Status)
	}
	q := fmt.Sprintf("SELECT %s FROM invoices WHERE %s ORDER BY %s LIMIT ? OFFSET ?", invoiceCols, where, orderClause(f.Sort))
	args = append(args, f.Limit, f.Offset)
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	return collectInvoices(rows)
}

func (r *Invoices) Get(ctx context.Context, companyID, id int64) (*model.Invoice, error) {
	row, err := r.db.QueryRow(ctx, "SELECT "+invoiceCols+" FROM invoices WHERE company_id = ? AND id = ?", companyID, id)
	if err != nil {
		return nil, err
	}
	return scanInvoice(row)
}

func (r *Invoices) Insert(ctx context.Context, companyID int64, inv *model.Invoice, payToken string) error {
	res, err := r.db.Exec(ctx,
		`INSERT INTO invoices (company_id, project_code, number, customer_name, customer_email, pay_token, amount_cents, status, due_date)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		companyID, inv.ProjectCode, inv.Number, inv.CustomerName, inv.CustomerEmail, payToken, inv.AmountCents, inv.Status, inv.DueDate)
	if err != nil {
		return err
	}
	inv.ID, _ = res.LastInsertId()
	return nil
}

func (r *Invoices) SetStatus(ctx context.Context, companyID, id int64, status string) error {
	_, err := r.db.Exec(ctx, "UPDATE invoices SET status = ? WHERE company_id = ? AND id = ?", status, companyID, id)
	return err
}

func (r *Invoices) Void(ctx context.Context, companyID, id int64, at time.Time) error {
	_, err := r.db.Exec(ctx,
		"UPDATE invoices SET prev_status = status, status = 'void', voided_at = ? WHERE company_id = ? AND id = ? AND status <> 'void'",
		at, companyID, id)
	return err
}

func (r *Invoices) Unvoid(ctx context.Context, companyID, id int64) error {
	_, err := r.db.Exec(ctx,
		"UPDATE invoices SET status = COALESCE(prev_status, 'draft'), prev_status = NULL, voided_at = NULL WHERE company_id = ? AND id = ? AND status = 'void'",
		companyID, id)
	return err
}

func (r *Invoices) ListByCompany(ctx context.Context, companyID int64, from, to time.Time) ([]*model.Invoice, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+invoiceCols+" FROM invoices WHERE company_id = ? AND issued_at >= ? AND issued_at < ? ORDER BY issued_at",
		companyID, from, to)
	if err != nil {
		return nil, err
	}
	return collectInvoices(rows)
}

// ListOverdue returns sent invoices past their due date, ordered by company.
func (r *Invoices) ListOverdue(ctx context.Context, asOf time.Time) ([]*model.Invoice, error) {
	rows, err := r.db.System().QueryContext(ctx,
		"SELECT "+invoiceCols+" FROM invoices WHERE status = 'sent' AND due_date < ? ORDER BY company_id, due_date", asOf)
	if err != nil {
		return nil, err
	}
	return collectInvoices(rows)
}

// SetCustomerNote stores the message a customer leaves on the payment page.
func (r *Invoices) SetCustomerNote(ctx context.Context, payToken, note string) error {
	res, err := r.db.System().ExecContext(ctx,
		"UPDATE invoices SET customer_note = ? WHERE pay_token = ? AND status IN ('sent', 'paid')", note, payToken)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Invoices) EnqueueExport(ctx context.Context, companyID, memberID int64, kind string, from, to *time.Time) (int64, error) {
	res, err := r.db.Exec(ctx,
		"INSERT INTO export_jobs (company_id, requested_by, kind, date_from, date_to) VALUES (?, ?, ?, ?, ?)",
		companyID, memberID, kind, from, to)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Invoices) PendingExports(ctx context.Context, limit int) ([]model.ExportJob, error) {
	rows, err := r.db.System().QueryContext(ctx,
		`SELECT j.id, j.company_id, j.requested_by, u.email, j.kind, j.date_from, j.date_to
		   FROM export_jobs j
		   JOIN members m ON m.id = j.requested_by AND m.company_id = j.company_id
		   JOIN users u ON u.id = m.user_id
		  WHERE j.status = 'pending' AND m.status = 'active'
		  ORDER BY j.id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ExportJob
	for rows.Next() {
		var j model.ExportJob
		var from, to sql.NullTime
		if err := rows.Scan(&j.ID, &j.CompanyID, &j.RequestedBy, &j.RequesterEmail, &j.Kind, &from, &to); err != nil {
			return nil, err
		}
		if from.Valid {
			j.From = &from.Time
		}
		if to.Valid {
			j.To = &to.Time
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (r *Invoices) FinishExport(ctx context.Context, job model.ExportJob, status, path string) error {
	_, err := r.db.Exec(ctx,
		"UPDATE export_jobs SET status = ?, file_path = ? WHERE company_id = ? AND id = ?",
		status, path, job.CompanyID, job.ID)
	return err
}
