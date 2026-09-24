package service

import (
	"strings"

	"example.com/wallet/internal/repo"
)

type Catalog struct{ products *repo.ProductRepo }

func NewCatalog(p *repo.ProductRepo) *Catalog { return &Catalog{products: p} }

func normalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}

func (c *Catalog) ByTag(tag string) ([]repo.Product, error) {
	return c.products.ListByTag(normalizeTag(tag))
}

func (c *Catalog) Search(q, sort string) ([]repo.Product, error) {
	return c.products.Search(strings.TrimSpace(q), sort)
}
