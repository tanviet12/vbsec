package jobs

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"example.com/tallyboard/internal/model"
	"example.com/tallyboard/internal/repo"
	"example.com/tallyboard/internal/service"
)

// Exports turns queued export requests into CSV files and tells the
// requester when the file is ready.
type Exports struct {
	Invoices *repo.Invoices
	Mail     service.Mailer
	Dir      string
}

func (w *Exports) RunOnce(ctx context.Context) error {
	pending, err := w.Invoices.PendingExports(ctx, 20)
	if err != nil {
		return err
	}
	for _, job := range pending {
		path, n, err := w.build(ctx, job)
		status := "done"
		if err != nil {
			status = "failed"
			log.Printf("exports: job %d failed: %v", job.ID, err)
		}
		if ferr := w.Invoices.FinishExport(ctx, job, status, path); ferr != nil {
			return ferr
		}
		if err == nil {
			msg := fmt.Sprintf("Your export with %d row(s) is ready. Open Exports in the app to download it.", n)
			_ = w.Mail.Send(ctx, job.RequesterEmail, "Export ready", msg)
		}
	}
	return nil
}

func (w *Exports) build(ctx context.Context, job model.ExportJob) (string, int, error) {
	var (
		rows []*model.Invoice
		err  error
	)
	switch job.Kind {
	case "invoices":
		if job.From == nil || job.To == nil {
			return "", 0, fmt.Errorf("missing date range")
		}
		rows, err = w.Invoices.ListByCompany(ctx, job.CompanyID, *job.From, *job.To)
	case "overdue":
		rows, err = w.Invoices.ListOverdue(ctx, time.Now().UTC())
	default:
		return "", 0, fmt.Errorf("unknown export kind %q", job.Kind)
	}
	if err != nil {
		return "", 0, err
	}
	dir := filepath.Join(w.Dir, strconv.FormatInt(job.CompanyID, 10))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", 0, err
	}
	path := filepath.Join(dir, fmt.Sprintf("export-%d.csv", job.ID))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	cw := csv.NewWriter(f)
	_ = cw.Write([]string{"number", "project", "customer", "email", "amount_cents", "status", "due_date"})
	for _, inv := range rows {
		_ = cw.Write([]string{
			inv.Number, inv.ProjectCode, inv.CustomerName, inv.CustomerEmail,
			strconv.FormatInt(inv.AmountCents, 10), inv.Status, inv.DueDate.Format("2006-01-02"),
		})
	}
	cw.Flush()
	return path, len(rows), cw.Error()
}
