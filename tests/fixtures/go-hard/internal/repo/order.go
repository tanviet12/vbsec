package repo

import "github.com/jmoiron/sqlx"

type Order struct {
	ID      int64  `db:"id"`
	UserID  int64  `db:"user_id"`
	Address string `db:"address"`
	Total   int64  `db:"total"`
}

type OrderRepo struct{ db *sqlx.DB }

func NewOrderRepo(db *sqlx.DB) *OrderRepo { return &OrderRepo{db: db} }

func (r *OrderRepo) Get(id int64) (*Order, error) {
	var o Order
	err := r.db.Get(&o, "SELECT id, user_id, address, total FROM orders WHERE id = ?", id)
	return &o, err
}

func (r *OrderRepo) UpdateAddress(id int64, address string) error {
	_, err := r.db.Exec("UPDATE orders SET address = ? WHERE id = ?", address, id)
	return err
}
