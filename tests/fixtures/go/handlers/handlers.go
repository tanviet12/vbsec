package handlers

import "gorm.io/gorm"

type Handler struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

type User struct {
	ID    uint
	Name  string
	Email string
}

type Order struct {
	ID     uint
	UserID uint
	Total  int
}
