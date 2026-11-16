package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"example.com/ledger/internal/service"
)

const maxHookBody = 64 << 10

func (a *API) signedBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxHookBody))
	if err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "body too large"})
		return nil, false
	}
	if !a.Payments.Authentic(body, r.Header.Get("X-Signature")) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "bad signature"})
		return nil, false
	}
	return body, true
}

func (a *API) bankHook(w http.ResponseWriter, r *http.Request) {
	body, ok := a.signedBody(w, r)
	if !ok {
		return
	}
	var ev service.BankEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		a.fail(w, r, service.ErrInvalid)
		return
	}
	if err := a.Payments.ApplyBankEvent(r.Context(), ev); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) payoutHook(w http.ResponseWriter, r *http.Request) {
	body, ok := a.signedBody(w, r)
	if !ok {
		return
	}
	var ev service.PayoutEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		a.fail(w, r, service.ErrInvalid)
		return
	}
	if err := a.Payments.ApplyPayoutEvent(r.Context(), ev); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
