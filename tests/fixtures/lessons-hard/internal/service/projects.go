package service

import (
	"context"
	"database/sql"
	"strings"

	"example.com/crewdesk/internal/audit"
	"example.com/crewdesk/internal/authn"
)

type Project struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	BudgetCents int64  `json:"budget_cents"`
	BudgetShown bool   `json:"budget_shown"`
}

type ProjectPatch struct {
	Name        *string `json:"name"`
	BudgetCents *int64  `json:"budget_cents"`
}

type Projects struct {
	db    *sql.DB
	audit *audit.Log
}

func NewProjects(db *sql.DB, log *audit.Log) *Projects { return &Projects{db: db, audit: log} }

func (s *Projects) Get(ctx context.Context, actor authn.Principal, id int64) (Project, error) {
	p := Project{ID: id}
	err := s.db.QueryRowContext(ctx,
		"SELECT name, status, budget_cents FROM projects WHERE id = ? AND company_id = ?", id, actor.CompanyID).
		Scan(&p.Name, &p.Status, &p.BudgetCents)
	if err != nil {
		return Project{}, ErrNotFound
	}
	p.BudgetShown = actor.Can("finance.view")
	if !p.BudgetShown {
		p.BudgetCents = 0
	}
	return p, nil
}

func (s *Projects) Update(ctx context.Context, actor authn.Principal, id int64, patch ProjectPatch) (Project, error) {
	if !actor.Can("projects.edit") {
		return Project{}, ErrForbidden
	}
	p, err := s.Get(ctx, actor, id)
	if err != nil {
		return Project{}, err
	}
	if p.Status != "open" {
		return Project{}, ErrConflict
	}
	var changed []string
	if patch.Name != nil && strings.TrimSpace(*patch.Name) != "" && len(*patch.Name) <= 160 {
		p.Name = strings.TrimSpace(*patch.Name)
		changed = append(changed, "name")
	}
	if patch.BudgetCents != nil && *patch.BudgetCents >= 0 {
		p.BudgetCents = *patch.BudgetCents
		changed = append(changed, "budget_cents")
	}
	if _, err := s.db.ExecContext(ctx,
		"UPDATE projects SET name = ?, budget_cents = ? WHERE id = ? AND company_id = ? AND status = 'open'",
		p.Name, p.BudgetCents, id, actor.CompanyID); err != nil {
		return Project{}, err
	}
	if err := s.audit.Record(ctx, actor.CompanyID, actor.UserID, "project_updated",
		map[string]any{"project_id": id, "changed": strings.Join(changed, ",")}); err != nil {
		return Project{}, err
	}
	return p, nil
}

func (s *Projects) AddNote(ctx context.Context, actor authn.Principal, id int64, body string) error {
	body = strings.TrimSpace(body)
	if body == "" || len(body) > 500 {
		return ErrInvalid
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO project_notes (company_id, project_id, author_id, body)
		 SELECT company_id, id, ?, ? FROM projects WHERE id = ? AND company_id = ? AND status = 'open'`,
		actor.UserID, body, id, actor.CompanyID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrNotFound
	}
	return nil
}

func (s *Projects) Close(ctx context.Context, actor authn.Principal, id int64) error {
	if !actor.IsOwner() {
		return ErrForbidden
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	if err := tx.QueryRowContext(ctx,
		"SELECT status FROM projects WHERE id = ? AND company_id = ? FOR UPDATE", id, actor.CompanyID).Scan(&status); err != nil {
		return ErrNotFound
	}
	if status != "open" {
		return ErrConflict
	}
	rows, err := tx.QueryContext(ctx,
		"SELECT body FROM project_notes WHERE project_id = ? AND company_id = ? ORDER BY id", id, actor.CompanyID)
	if err != nil {
		return err
	}
	var notes []string
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			rows.Close()
			return err
		}
		notes = append(notes, "- "+body)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		"UPDATE projects SET status = 'closed', closing_summary = ? WHERE id = ? AND company_id = ?",
		strings.Join(notes, "\n"), id, actor.CompanyID); err != nil {
		return err
	}
	if err := s.audit.RecordTx(ctx, tx, actor.CompanyID, actor.UserID, "project_closed",
		map[string]any{"project_id": id, "notes": len(notes)}); err != nil {
		return err
	}
	return tx.Commit()
}
