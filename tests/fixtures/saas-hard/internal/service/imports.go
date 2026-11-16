package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"example.com/tallyboard/internal/authz"
	"example.com/tallyboard/internal/model"
	"example.com/tallyboard/internal/repo"
	"example.com/tallyboard/internal/tenant"
)

type Imports struct {
	Invoices *repo.Invoices
	Projects *repo.Projects
}

const maxImportRows = 2000

var importStatuses = map[string]bool{"draft": true, "sent": true, "paid": true}

type ImportResult struct {
	Imported int      `json:"imported"`
	Skipped  []string `json:"skipped"`
}

// ImportInvoices loads invoices from a CSV exported by another billing tool.
// Columns: project_code, number, customer_name, customer_email, amount_cents, due_date, status.
func (s *Imports) ImportInvoices(ctx context.Context, a *tenant.Actor, src io.Reader) (*ImportResult, error) {
	if !authz.IsStaff(a) {
		return nil, ErrForbidden
	}
	r := csv.NewReader(src)
	r.FieldsPerRecord = 7
	if _, err := r.Read(); err != nil {
		return nil, ErrInvalid
	}
	res := &ImportResult{Skipped: []string{}}
	known := map[string]bool{}
	for line := 2; ; line++ {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || line > maxImportRows+1 {
			return nil, ErrInvalid
		}
		code, number := strings.TrimSpace(rec[0]), strings.TrimSpace(rec[1])
		amount, aerr := strconv.ParseInt(rec[4], 10, 64)
		due, derr := time.Parse("2006-01-02", rec[5])
		status := strings.ToLower(strings.TrimSpace(rec[6]))
		_, merr := mail.ParseAddress(rec[3])
		if number == "" || aerr != nil || amount <= 0 || derr != nil || merr != nil || !importStatuses[status] {
			res.Skipped = append(res.Skipped, fmt.Sprintf("line %d: invalid row", line))
			continue
		}
		if !known[code] {
			if _, err := s.Projects.GetByCode(ctx, a.CompanyID, code); err != nil {
				res.Skipped = append(res.Skipped, fmt.Sprintf("line %d: unknown project", line))
				continue
			}
			known[code] = true
		}
		token, err := newToken()
		if err != nil {
			return nil, err
		}
		inv := &model.Invoice{
			ProjectCode: code, Number: number, CustomerName: strings.TrimSpace(rec[2]), CustomerEmail: rec[3],
			AmountCents: amount, Status: status, DueDate: due,
		}
		if err := s.Invoices.Insert(ctx, a.CompanyID, inv, token); err != nil {
			res.Skipped = append(res.Skipped, fmt.Sprintf("line %d: duplicate number", line))
			continue
		}
		res.Imported++
	}
	return res, nil
}
