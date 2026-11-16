package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"example.com/tallyboard/internal/model"
)

var ErrNotFound = errors.New("not found")

type Projects struct {
	db *DB
}

func NewProjects(db *DB) *Projects { return &Projects{db: db} }

const projectCols = `id, company_id, code, name, manager_id, budget_cents, hourly_rate_cents, COALESCE(notes, ''), archived_at`

func scanProject(s interface{ Scan(...any) error }) (*model.Project, error) {
	var p model.Project
	var archived sql.NullTime
	if err := s.Scan(&p.ID, &p.CompanyID, &p.Code, &p.Name, &p.ManagerID, &p.BudgetCents, &p.HourlyRateCents, &p.Notes, &archived); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if archived.Valid {
		p.ArchivedAt = &archived.Time
	}
	return &p, nil
}

var projectOrder = map[string]string{
	"name":    "name ASC",
	"code":    "code ASC",
	"newest":  "id DESC",
	"budget":  "budget_cents DESC",
	"manager": "manager_id ASC, name ASC",
}

func (r *Projects) List(ctx context.Context, companyID int64, sort string, archived bool) ([]*model.Project, error) {
	order, ok := projectOrder[sort]
	if !ok {
		order = projectOrder["name"]
	}
	cond := "archived_at IS NULL"
	if archived {
		cond = "archived_at IS NOT NULL"
	}
	q := fmt.Sprintf("SELECT %s FROM projects WHERE company_id = ? AND %s ORDER BY %s", projectCols, cond, order)
	rows, err := r.db.Query(ctx, q, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Projects) Get(ctx context.Context, companyID, id int64) (*model.Project, error) {
	row, err := r.db.QueryRow(ctx, "SELECT "+projectCols+" FROM projects WHERE company_id = ? AND id = ?", companyID, id)
	if err != nil {
		return nil, err
	}
	return scanProject(row)
}

func (r *Projects) GetByCode(ctx context.Context, companyID int64, code string) (*model.Project, error) {
	row, err := r.db.QueryRow(ctx, "SELECT "+projectCols+" FROM projects WHERE company_id = ? AND code = ?", companyID, code)
	if err != nil {
		return nil, err
	}
	return scanProject(row)
}

func (r *Projects) SetArchived(ctx context.Context, companyID, id int64, at *time.Time) error {
	_, err := r.db.Exec(ctx, "UPDATE projects SET archived_at = ? WHERE company_id = ? AND id = ?", at, companyID, id)
	return err
}

func (r *Projects) Delete(ctx context.Context, companyID, id int64) error {
	if _, err := r.db.Exec(ctx, "DELETE FROM tasks WHERE company_id = ? AND project_id = ?", companyID, id); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, "DELETE FROM projects WHERE company_id = ? AND id = ?", companyID, id)
	return err
}

func (r *Projects) IsParticipant(ctx context.Context, companyID, projectID, memberID int64) (bool, error) {
	row, err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM project_members WHERE company_id = ? AND project_id = ? AND member_id = ?",
		companyID, projectID, memberID)
	if err != nil {
		return false, err
	}
	var n int
	if err := row.Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *Projects) Tasks(ctx context.Context, companyID, projectID int64) ([]model.Task, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id, project_id, title, status, assignee_id FROM tasks WHERE company_id = ? AND project_id = ? ORDER BY id",
		companyID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Task
	for rows.Next() {
		var t model.Task
		var assignee sql.NullInt64
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Status, &assignee); err != nil {
			return nil, err
		}
		if assignee.Valid {
			t.AssigneeID = &assignee.Int64
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// TaskProject resolves the project a task belongs to inside the company.
func (r *Projects) TaskProject(ctx context.Context, companyID, taskID int64) (int64, error) {
	row, err := r.db.QueryRow(ctx, "SELECT project_id FROM tasks WHERE company_id = ? AND id = ?", companyID, taskID)
	if err != nil {
		return 0, err
	}
	var id int64
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return id, nil
}

func (r *Projects) AddExpense(ctx context.Context, companyID int64, e *model.Expense) error {
	res, err := r.db.Exec(ctx,
		"INSERT INTO expenses (company_id, project_id, member_id, amount_cents, note) VALUES (?, ?, ?, ?, ?)",
		companyID, e.ProjectID, e.MemberID, e.AmountCents, e.Note)
	if err != nil {
		return err
	}
	e.ID, _ = res.LastInsertId()
	e.CreatedAt = time.Now().UTC()
	return nil
}

func (r *Projects) SpentCents(ctx context.Context, companyID, projectID int64) (int64, error) {
	row, err := r.db.QueryRow(ctx,
		"SELECT COALESCE(SUM(amount_cents), 0) FROM expenses WHERE company_id = ? AND project_id = ?",
		companyID, projectID)
	if err != nil {
		return 0, err
	}
	var n int64
	return n, row.Scan(&n)
}
