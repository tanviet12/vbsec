package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"example.com/tallyboard/internal/model"
	"example.com/tallyboard/internal/repo"
	"example.com/tallyboard/internal/service"
)

// Reminders mails each company a daily list of its overdue invoices.
type Reminders struct {
	Invoices *repo.Invoices
	System   *sql.DB
	Mail     service.Mailer
}

func (j *Reminders) Run(ctx context.Context) error {
	overdue, err := j.Invoices.ListOverdue(ctx, time.Now().UTC())
	if err != nil {
		return err
	}
	byCompany := map[int64][]*model.Invoice{}
	for _, inv := range overdue {
		byCompany[inv.CompanyID] = append(byCompany[inv.CompanyID], inv)
	}
	for companyID, list := range byCompany {
		var to string
		if err := j.System.QueryRowContext(ctx, "SELECT billing_email FROM companies WHERE id = ?", companyID).Scan(&to); err != nil {
			log.Printf("reminders: company %d has no billing contact: %v", companyID, err)
			continue
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "%d invoice(s) are past their due date:\n\n", len(list))
		for _, inv := range list {
			fmt.Fprintf(&sb, "  %s  %s  due %s\n", inv.Number, inv.CustomerName, inv.DueDate.Format("2006-01-02"))
		}
		if err := j.Mail.Send(ctx, to, "Overdue invoices", sb.String()); err != nil {
			log.Printf("reminders: send failed for company %d: %v", companyID, err)
		}
	}
	return nil
}
