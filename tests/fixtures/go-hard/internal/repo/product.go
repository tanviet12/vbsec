package repo

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Product struct {
	ID    int64  `db:"id"`
	Name  string `db:"name"`
	Price int64  `db:"price"`
	Tag   string `db:"tag"`
}

type ProductRepo struct{ db *sqlx.DB }

func NewProductRepo(db *sqlx.DB) *ProductRepo { return &ProductRepo{db: db} }

var sortColumns = map[string]string{
	"price": "price",
	"name":  "name",
	"new":   "created_at",
}

func (r *ProductRepo) Search(q, sort string) ([]Product, error) {
	col, ok := sortColumns[sort]
	if !ok {
		col = "created_at"
	}
	query := fmt.Sprintf("SELECT id, name, price, tag FROM products WHERE name LIKE ? ORDER BY %s DESC LIMIT 50", col)
	var out []Product
	err := r.db.Select(&out, query, "%"+q+"%")
	return out, err
}

func (r *ProductRepo) ListByTag(tag string) ([]Product, error) {
	query := fmt.Sprintf("SELECT id, name, price, tag FROM products WHERE tag = '%s' LIMIT 50", tag)
	var out []Product
	err := r.db.Select(&out, query)
	return out, err
}
