package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/mail"
	"strings"
	"time"

	"example.com/tallyboard/internal/authz"
	"example.com/tallyboard/internal/model"
	"example.com/tallyboard/internal/repo"
	"example.com/tallyboard/internal/tenant"
)

type Invoices struct {
	Repo     *repo.Invoices
	Projects *repo.Projects
}

type NewInvoice struct {
	ProjectCode   string    `json:"project_code"`
	Number        string    `json:"number"`
	CustomerName  string    `json:"customer_name"`
	CustomerEmail string    `json:"customer_email"`
	AmountCents   int64     `json:"amount_cents"`
	DueDate       time.Time `json:"due_date"`
}

func newToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (in *NewInvoice) valid() bool {
	if in.AmountCents <= 0 || strings.TrimSpace(in.Number) == "" || strings.TrimSpace(in.CustomerName) == "" {
		return false
	}
	_, err := mail.ParseAddress(in.CustomerEmail)
	return err == nil
}

func (s *Invoices) Search(ctx context.Context, a *tenant.Actor, f repo.InvoiceFilter) ([]*model.Invoice, error) {
	if !authz.CanViewFinance(a) {
		return nil, ErrForbidden
	}
	return s.Repo.Search(ctx, a.CompanyID, f)
}

func (s *Invoices) Get(ctx context.Context, a *tenant.Actor, id int64) (*model.Invoice, error) {
	if !authz.CanViewFinance(a) {
		return nil, ErrForbidden
	}
	return s.Repo.Get(ctx, a.CompanyID, id)
}

// Create drafts an invoice for a project the actor is allowed to bill.
func (s *Invoices) Create(ctx context.Context, a *tenant.Actor, in NewInvoice) (*model.Invoice, error) {
	if !in.valid() {
		return nil, ErrInvalid
	}
	p, err := s.Projects.GetByCode(ctx, a.CompanyID, in.ProjectCode)
	if err != nil {
		return nil, err
	}
	if !authz.CanBill(a, p) {
		return nil, ErrForbidden
	}
	token, err := newToken()
	if err != nil {
		return nil, err
	}
	inv := &model.Invoice{
		ProjectCode: p.Code, Number: in.Number, CustomerName: in.CustomerName, CustomerEmail: in.CustomerEmail,
		AmountCents: in.AmountCents, Status: "draft", DueDate: in.DueDate, IssuedAt: time.Now().UTC(),
	}
	if err := s.Repo.Insert(ctx, a.CompanyID, inv, token); err != nil {
		return nil, err
	}
	return inv, nil
}

func (s *Invoices) MarkPaid(ctx context.Context, a *tenant.Actor, id int64) error {
	if !authz.CanManageInvoices(a) {
		return ErrForbidden
	}
	inv, err := s.Repo.Get(ctx, a.CompanyID, id)
	if err != nil {
		return err
	}
	if inv.Status != "sent" {
		return ErrInvalid
	}
	return s.Repo.SetStatus(ctx, a.CompanyID, id, "paid")
}

func (s *Invoices) Void(ctx context.Context, a *tenant.Actor, id int64) error {
	if !authz.CanManageInvoices(a) {
		return ErrForbidden
	}
	inv, err := s.Repo.Get(ctx, a.CompanyID, id)
	if err != nil {
		return err
	}
	if inv.Status == "void" {
		return nil
	}
	return s.Repo.Void(ctx, a.CompanyID, id, time.Now().UTC())
}

func (s *Invoices) Restore(ctx context.Context, a *tenant.Actor, id int64) (*model.Invoice, error) {
	inv, err := s.Repo.Get(ctx, a.CompanyID, id)
	if err != nil {
		return nil, err
	}
	if inv.Status != "void" {
		return nil, ErrInvalid
	}
	if err := s.Repo.Unvoid(ctx, a.CompanyID, id); err != nil {
		return nil, err
	}
	return s.Repo.Get(ctx, a.CompanyID, id)
}

// LeaveNote records what a customer wrote on the public payment page.
func (s *Invoices) LeaveNote(ctx context.Context, payToken, note string) error {
	note = strings.TrimSpace(note)
	if len(payToken) != 48 || note == "" || len(note) > 500 {
		return ErrInvalid
	}
	return s.Repo.SetCustomerNote(ctx, payToken, note)
}

var exportKinds = map[string]bool{"invoices": true, "overdue": true}

func (s *Invoices) RequestExport(ctx context.Context, a *tenant.Actor, kind string, from, to *time.Time) (int64, error) {
	if !authz.CanViewFinance(a) && !a.Has(authz.PermExports) {
		return 0, ErrForbidden
	}
	if !exportKinds[kind] {
		return 0, ErrInvalid
	}
	if kind == "invoices" && (from == nil || to == nil || !from.Before(*to)) {
		return 0, ErrInvalid
	}
	return s.Repo.EnqueueExport(ctx, a.CompanyID, a.MemberID, kind, from, to)
}
