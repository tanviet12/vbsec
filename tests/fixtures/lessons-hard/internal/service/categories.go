package service

import (
	"context"
	"database/sql"
	"strings"

	"example.com/crewdesk/internal/audit"
	"example.com/crewdesk/internal/authn"
)

const maxCategories = 200

type Categories struct {
	db    *sql.DB
	audit *audit.Log
}

func NewCategories(db *sql.DB, log *audit.Log) *Categories { return &Categories{db: db, audit: log} }

func (s *Categories) Create(ctx context.Context, actor authn.Principal, name string) (int64, error) {
	if !actor.CanManageSettings() {
		return 0, ErrForbidden
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 {
		return 0, ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var count int
	if err := tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM categories WHERE company_id = ? FOR UPDATE", actor.CompanyID).Scan(&count); err != nil {
		return 0, err
	}
	if count >= maxCategories {
		return 0, ErrConflict
	}
	res, err := tx.ExecContext(ctx,
		"INSERT INTO categories (company_id, name) VALUES (?, ?)", actor.CompanyID, name)
	if err != nil {
		return 0, ErrConflict
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := s.audit.RecordTx(ctx, tx, actor.CompanyID, actor.UserID, "category_created",
		map[string]any{"category_id": id}); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}
