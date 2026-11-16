package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/mail"
	"strings"
	"time"

	"example.com/ledger/internal/audit"
	"example.com/ledger/internal/platform"
	"example.com/ledger/internal/repo"
)

const inviteTTL = 72 * time.Hour

type PayrollInput struct {
	BaseSalary  int64
	TaxCode     string
	BankAccount string
}

type Staff struct {
	db       *sql.DB
	staff    *repo.StaffRepo
	identity *repo.IdentityRepo
	box      *platform.Box
	audit    *audit.Log
	log      *slog.Logger
	mailer   *platform.Mailer
	baseURL  string
}

func NewStaff(db *sql.DB, staff *repo.StaffRepo, identity *repo.IdentityRepo, box *platform.Box,
	trail *audit.Log, log *slog.Logger, mailer *platform.Mailer, baseURL string) *Staff {
	return &Staff{db: db, staff: staff, identity: identity, box: box, audit: trail, log: log,
		mailer: mailer, baseURL: strings.TrimRight(baseURL, "/")}
}

func (s *Staff) Invite(ctx context.Context, actor repo.Actor, email string, departmentID int64) error {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return ErrInvalid
	}
	if _, err := s.staff.Department(ctx, s.db, departmentID); err != nil {
		return notFound(err)
	}
	token, err := platform.NewToken("")
	if err != nil {
		return err
	}
	if err := s.staff.CreateInvite(ctx, s.db, addr.Address, platform.HashToken(token), departmentID,
		actor.ID, time.Now().Add(inviteTTL)); err != nil {
		return err
	}
	link := s.baseURL + "/join#" + token
	if err := s.mailer.Send(ctx, addr.Address, "You have been invited", "Open this link to join: "+link); err != nil {
		s.log.Error("invite mail failed", "department_id", departmentID, "err", err.Error())
		return err
	}
	s.log.Info("invite sent", "by", actor.ID, "department_id", departmentID, "link", link)
	return s.audit.Record(ctx, nil, actor.ID, "staff.invite", departmentID, audit.Fields{"department_id": departmentID})
}

func (s *Staff) UpdatePayroll(ctx context.Context, actor repo.Actor, employeeID int64, in PayrollInput) error {
	if in.BaseSalary < 0 || len(in.TaxCode) > 14 || len(in.BankAccount) > 32 {
		return ErrInvalid
	}
	changed := []string{"base_salary"}
	var taxCode, bankAccount []byte
	var err error
	if in.TaxCode != "" {
		if taxCode, err = s.box.Seal([]byte(in.TaxCode)); err != nil {
			return err
		}
		changed = append(changed, "tax_code")
	}
	if in.BankAccount != "" {
		if bankAccount, err = s.box.Seal([]byte(in.BankAccount)); err != nil {
			return err
		}
		changed = append(changed, "bank_account")
	}
	err = repo.InTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := s.staff.LockEmployee(ctx, tx, employeeID); err != nil {
			return notFound(err)
		}
		if err := s.staff.UpdatePayroll(ctx, tx, employeeID, in.BaseSalary, taxCode, bankAccount); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, actor.ID, "staff.payroll", employeeID, audit.Fields{"changed": changed})
	})
	if err != nil {
		return err
	}
	s.log.Info("payroll updated", "by", actor.ID, "employee_id", employeeID, "fields", changed)
	return nil
}

func (s *Staff) Offboard(ctx context.Context, actor repo.Actor, employeeID int64) error {
	err := repo.InTx(ctx, s.db, func(tx *sql.Tx) error {
		employee, err := s.staff.LockEmployee(ctx, tx, employeeID)
		if err != nil {
			return notFound(err)
		}
		if employee.UserID == actor.ID {
			return ErrForbidden
		}
		if employee.Status != "active" {
			return ErrConflict
		}
		if err := s.staff.Terminate(ctx, tx, employee.ID); err != nil {
			return err
		}
		if err := s.identity.SetActive(ctx, tx, employee.UserID, false); err != nil {
			return err
		}
		if err := s.identity.DeleteSessionsByUser(ctx, tx, employee.UserID); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, actor.ID, "staff.offboard", employee.ID, audit.Fields{"status": "terminated"})
	})
	if err != nil {
		return err
	}
	s.log.Info("employee offboarded", "by", actor.ID, "employee_id", employeeID)
	return nil
}

func (s *Staff) ChangeHead(ctx context.Context, actor repo.Actor, departmentID, userID int64) error {
	return repo.InTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := s.staff.Department(ctx, tx, departmentID); err != nil {
			return notFound(err)
		}
		head, err := s.staff.EmployeeByUser(ctx, tx, userID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalid
		}
		if err != nil {
			return err
		}
		if head.Status != "active" || head.DepartmentID != departmentID {
			return ErrInvalid
		}
		if err := s.staff.SetHead(ctx, tx, departmentID, userID); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, actor.ID, "department.head", departmentID, audit.Fields{"department_id": departmentID})
	})
}
