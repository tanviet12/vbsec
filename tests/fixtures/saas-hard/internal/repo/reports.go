package repo

import (
	"context"

	"example.com/tallyboard/internal/model"
)

type Reports struct {
	db *DB
}

func NewReports(db *DB) *Reports { return &Reports{db: db} }

// ProjectBilling totals issued invoices per active project.
func (r *Reports) ProjectBilling(ctx context.Context, companyID int64) ([]model.ProjectBilling, error) {
	rows, err := r.db.Query(ctx,
		`SELECT p.id, p.code, p.name, COALESCE(SUM(i.amount_cents), 0), COUNT(i.id)
		   FROM projects p
		   LEFT JOIN invoices i ON i.project_code = p.code AND i.status IN ('sent', 'paid')
		  WHERE p.company_id = ? AND p.archived_at IS NULL
		  GROUP BY p.id, p.code, p.name
		  ORDER BY p.code`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ProjectBilling
	for rows.Next() {
		var b model.ProjectBilling
		if err := rows.Scan(&b.ProjectID, &b.Code, &b.Name, &b.BilledCents, &b.Invoices); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// MemberLoad counts open tasks per active member.
func (r *Reports) MemberLoad(ctx context.Context, companyID int64) ([]model.MemberLoad, error) {
	rows, err := r.db.Query(ctx,
		`SELECT m.id, u.name, COUNT(t.id)
		   FROM members m
		   JOIN users u ON u.id = m.user_id
		   LEFT JOIN tasks t ON t.assignee_id = m.id AND t.company_id = m.company_id AND t.status <> 'done'
		  WHERE m.company_id = ? AND m.status = 'active'
		  GROUP BY m.id, u.name
		  ORDER BY u.name`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.MemberLoad
	for rows.Next() {
		var l model.MemberLoad
		if err := rows.Scan(&l.MemberID, &l.Name, &l.OpenTasks); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
