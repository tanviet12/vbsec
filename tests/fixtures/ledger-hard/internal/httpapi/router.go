package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"example.com/ledger/internal/repo"
	"example.com/ledger/internal/service"
)

type Deps struct {
	Log      *slog.Logger
	Identity *repo.IdentityRepo
	Auth     *service.Auth
	Entries  *service.Entries
	Imports  *service.Imports
	Periods  *service.Periods
	Expenses *service.Expenses
	Funds    *service.Funds
	Staff    *service.Staff
	Payments *service.Payments
}

type API struct{ Deps }

func New(d Deps) *API { return &API{Deps: d} }

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	guard := func(h http.HandlerFunc, roles ...string) http.Handler {
		return a.authenticate(a.allow(roles, h))
	}

	mux.HandleFunc("POST /api/login", a.login)
	mux.HandleFunc("POST /api/login/verify", a.verifyCode)
	mux.Handle("POST /api/logout", guard(a.logout))
	mux.Handle("POST /api/password", guard(a.changePassword))
	mux.Handle("POST /api/tokens", guard(a.issueToken))

	mux.Handle("POST /api/entries", guard(a.createEntry, "accountant", "admin"))
	mux.Handle("PATCH /api/entries/{id}", guard(a.updateEntry, "accountant", "admin"))
	mux.Handle("DELETE /api/entries/{id}", guard(a.deleteEntry, "accountant", "admin"))
	mux.Handle("POST /api/entries/{id}/restore", guard(a.restoreEntry, "accountant", "admin"))
	mux.Handle("POST /api/funds/{id}/statement", guard(a.importStatement, "accountant", "admin"))

	mux.Handle("GET /api/periods/{id}/summary", guard(a.periodSummary, "accountant", "admin", "manager"))
	mux.Handle("POST /api/periods/{id}/close", guard(a.closePeriod, "accountant", "admin"))
	mux.Handle("POST /api/funds/transfer", guard(a.transferFunds, "accountant", "admin"))
	mux.Handle("PUT /api/funds/{id}/archived", guard(a.archiveFund, "accountant", "admin"))
	mux.Handle("POST /api/funds/{id}/cancel-open", guard(a.cancelOpenRequests, "admin"))

	mux.Handle("POST /api/expenses", guard(a.createExpense))
	mux.Handle("POST /api/expenses/{id}/decision", guard(a.decideExpense))

	mux.Handle("POST /api/invites", guard(a.inviteEmployee, "admin"))
	mux.Handle("PUT /api/employees/{id}/payroll", guard(a.updatePayroll, "admin"))
	mux.Handle("POST /api/employees/{id}/offboard", guard(a.offboardEmployee, "admin"))
	mux.Handle("PUT /api/departments/{id}/head", guard(a.changeHead, "admin"))

	mux.HandleFunc("POST /hooks/bank", a.bankHook)
	mux.HandleFunc("POST /hooks/payout", a.payoutHook)

	return a.logRequests(mux)
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

var statusByError = []struct {
	err    error
	status int
}{
	{service.ErrBadCredentials, http.StatusUnauthorized},
	{service.ErrBadTicket, http.StatusUnauthorized},
	{service.ErrBadCode, http.StatusUnauthorized},
	{service.ErrLocked, http.StatusTooManyRequests},
	{service.ErrForbidden, http.StatusForbidden},
	{service.ErrNotFound, http.StatusNotFound},
	{service.ErrInvalid, http.StatusUnprocessableEntity},
	{service.ErrConflict, http.StatusConflict},
	{service.ErrPeriodClosed, http.StatusConflict},
	{service.ErrOpenRequests, http.StatusConflict},
	{service.ErrOverLimit, http.StatusConflict},
	{service.ErrInsufficient, http.StatusConflict},
}

func (a *API) fail(w http.ResponseWriter, r *http.Request, err error) {
	for _, m := range statusByError {
		if errors.Is(err, m.err) {
			writeJSON(w, m.status, map[string]string{"error": m.err.Error()})
			return
		}
	}
	a.Log.Error("request failed", "path", r.URL.Path, "err", err.Error())
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}
