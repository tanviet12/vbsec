package httpapi

import (
	"net/http"
	"time"

	"example.com/ledger/internal/service"
)

func (a *API) createEntry(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FundID int64   `json:"fund_id"`
		Date   string  `json:"date"`
		Kind   string  `json:"kind"`
		Amount int64   `json:"amount"`
		Memo   string  `json:"memo"`
		Labels []int64 `json:"labels"`
	}
	if !decode(w, r, &body) {
		return
	}
	date, err := time.Parse("2006-01-02", body.Date)
	if err != nil {
		a.fail(w, r, service.ErrInvalid)
		return
	}
	id, err := a.Entries.Create(r.Context(), actorFrom(r), service.NewEntry{
		FundID: body.FundID, Date: date, Kind: body.Kind, Amount: body.Amount, Memo: body.Memo, Labels: body.Labels,
	})
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (a *API) updateEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Amount int64   `json:"amount"`
		Memo   string  `json:"memo"`
		Labels []int64 `json:"labels"`
	}
	if !decode(w, r, &body) {
		return
	}
	patch := service.EntryPatch{Amount: body.Amount, Memo: body.Memo, Labels: body.Labels}
	if err := a.Entries.Update(r.Context(), actorFrom(r), id, patch); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) deleteEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := a.Entries.Delete(r.Context(), actorFrom(r), id); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) restoreEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := a.Entries.Restore(r.Context(), actorFrom(r), id); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) importStatement(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	count, err := a.Imports.Statement(r.Context(), actorFrom(r), id, r.Body)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"imported": count})
}
