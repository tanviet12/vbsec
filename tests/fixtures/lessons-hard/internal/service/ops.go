package service

import (
	"context"
	"database/sql"

	"example.com/crewdesk/internal/audit"
)

type Operator struct {
	ID   int64
	Name string
}

type CompanyOverview struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Members int    `json:"members"`
}

type RequestRow struct {
	ID          int64  `json:"id"`
	RequesterID int64  `json:"requester_id"`
	AmountCents int64  `json:"amount_cents"`
	Status      string `json:"status"`
}

type Ops struct {
	db    *sql.DB
	audit *audit.Log
}

func NewOps(db *sql.DB, log *audit.Log) *Ops { return &Ops{db: db, audit: log} }

func (s *Ops) Overview(ctx context.Context, op Operator, companyID int64) (CompanyOverview, error) {
	if err := s.audit.CrossTenant(ctx, op.ID, companyID, "overview"); err != nil {
		return CompanyOverview{}, err
	}
	var out CompanyOverview
	err := s.db.QueryRowContext(ctx,
		`SELECT c.name, c.status,
		        (SELECT COUNT(*) FROM members m WHERE m.company_id = c.id AND m.status = 'active')
		   FROM companies c WHERE c.id = ?`, companyID).Scan(&out.Name, &out.Status, &out.Members)
	if err != nil {
		return CompanyOverview{}, ErrNotFound
	}
	return out, nil
}

func (s *Ops) Requests(ctx context.Context, op Operator, companyID int64, status string) ([]RequestRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, requester_id, amount_cents, status FROM requests
		  WHERE company_id = ? AND (? = '' OR status = ?) ORDER BY id DESC LIMIT 200`,
		companyID, status, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RequestRow
	for rows.Next() {
		var r RequestRow
		if err := rows.Scan(&r.ID, &r.RequesterID, &r.AmountCents, &r.Status); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
