package httpapi

import (
	"net/http"

	"example.com/ledger/internal/service"
)

func (a *API) createExpense(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FundID int64  `json:"fund_id"`
		Amount int64  `json:"amount"`
		Reason string `json:"reason"`
	}
	if !decode(w, r, &body) {
		return
	}
	id, err := a.Expenses.Create(r.Context(), actorFrom(r), service.NewRequest{
		FundID: body.FundID, Amount: body.Amount, Reason: body.Reason,
	})
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (a *API) decideExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Approve bool `json:"approve"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := a.Expenses.Decide(r.Context(), actorFrom(r), id, body.Approve); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
