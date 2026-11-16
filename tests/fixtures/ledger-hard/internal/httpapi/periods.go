package httpapi

import "net/http"

func (a *API) periodSummary(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	summary, err := a.Periods.Summary(r.Context(), id)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (a *API) closePeriod(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := a.Periods.Close(r.Context(), actorFrom(r), id); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) transferFunds(w http.ResponseWriter, r *http.Request) {
	var body struct {
		From   int64 `json:"from"`
		To     int64 `json:"to"`
		Amount int64 `json:"amount"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := a.Funds.Transfer(r.Context(), actorFrom(r), body.From, body.To, body.Amount); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) archiveFund(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Archived bool `json:"archived"`
	}
	if !decode(w, r, &body) {
		return
	}
	if err := a.Funds.SetArchived(r.Context(), actorFrom(r), id, body.Archived); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) cancelOpenRequests(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	count, err := a.Expenses.CancelOpen(r.Context(), actorFrom(r), id)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"cancelled": count})
}
