package tenant

import "context"

// Actor is the authenticated member a request runs as.
type Actor struct {
	UserID    int64
	MemberID  int64
	CompanyID int64
	Email     string
	Role      string
	Perms     map[string]bool
}

func (a *Actor) Has(perm string) bool {
	return a != nil && a.Perms[perm]
}

type ctxKey struct{}

func With(ctx context.Context, a *Actor) context.Context {
	return context.WithValue(ctx, ctxKey{}, a)
}

func From(ctx context.Context) (*Actor, bool) {
	a, ok := ctx.Value(ctxKey{}).(*Actor)
	return a, ok && a != nil
}
