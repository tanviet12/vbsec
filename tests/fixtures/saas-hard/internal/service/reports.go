package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"example.com/tallyboard/internal/authz"
	"example.com/tallyboard/internal/model"
	"example.com/tallyboard/internal/repo"
	"example.com/tallyboard/internal/tenant"
)

type cacheEntry struct {
	value   any
	expires time.Time
}

type ttlCache struct {
	mu    sync.Mutex
	items map[string]cacheEntry
}

func newTTLCache() *ttlCache {
	return &ttlCache{items: map[string]cacheEntry{}}
}

func (c *ttlCache) get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[key]
	if !ok || time.Now().After(e.expires) {
		delete(c.items, key)
		return nil, false
	}
	return e.value, true
}

func (c *ttlCache) set(key string, v any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheEntry{value: v, expires: time.Now().Add(ttl)}
}

type Reports struct {
	Repo     *repo.Reports
	Projects *repo.Projects
	cache    *ttlCache
}

func NewReports(r *repo.Reports, p *repo.Projects) *Reports {
	return &Reports{Repo: r, Projects: p, cache: newTTLCache()}
}

type Dashboard struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	BudgetCents int64  `json:"budget_cents"`
	SpentCents  int64  `json:"spent_cents"`
	BilledCents int64  `json:"billed_cents"`
	OpenTasks   int    `json:"open_tasks"`
}

// Billing returns billed totals per project for the finance overview.
func (s *Reports) Billing(ctx context.Context, a *tenant.Actor) ([]model.ProjectBilling, error) {
	if !authz.CanViewFinance(a) {
		return nil, ErrForbidden
	}
	key := fmt.Sprintf("billing:%d", a.CompanyID)
	if v, ok := s.cache.get(key); ok {
		return v.([]model.ProjectBilling), nil
	}
	rows, err := s.Repo.ProjectBilling(ctx, a.CompanyID)
	if err != nil {
		return nil, err
	}
	s.cache.set(key, rows, time.Minute)
	return rows, nil
}

// ProjectDashboard is the budget card shown on top of a project page.
func (s *Reports) ProjectDashboard(ctx context.Context, a *tenant.Actor, code string) (*Dashboard, error) {
	if !authz.CanViewFinance(a) {
		return nil, ErrForbidden
	}
	key := "dash:" + strings.ToUpper(strings.TrimSpace(code))
	if v, ok := s.cache.get(key); ok {
		return v.(*Dashboard), nil
	}
	p, err := s.Projects.GetByCode(ctx, a.CompanyID, code)
	if err != nil {
		return nil, err
	}
	d := &Dashboard{Code: p.Code, Name: p.Name, BudgetCents: p.BudgetCents}
	if d.SpentCents, err = s.Projects.SpentCents(ctx, a.CompanyID, p.ID); err != nil {
		return nil, err
	}
	billing, err := s.Billing(ctx, a)
	if err != nil {
		return nil, err
	}
	for _, b := range billing {
		if b.ProjectID == p.ID {
			d.BilledCents = b.BilledCents
		}
	}
	tasks, err := s.Projects.Tasks(ctx, a.CompanyID, p.ID)
	if err != nil {
		return nil, err
	}
	for _, t := range tasks {
		if t.Status != "done" {
			d.OpenTasks++
		}
	}
	s.cache.set(key, d, 5*time.Minute)
	return d, nil
}

func (s *Reports) Load(ctx context.Context, a *tenant.Actor) ([]model.MemberLoad, error) {
	if !authz.IsStaff(a) {
		return nil, ErrForbidden
	}
	return s.Repo.MemberLoad(ctx, a.CompanyID)
}
