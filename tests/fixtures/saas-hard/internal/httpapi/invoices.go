package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"example.com/tallyboard/internal/repo"
	"example.com/tallyboard/internal/service"
)

func (s *Server) listInvoices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}
	f := repo.InvoiceFilter{Status: q.Get("status"), Sort: q.Get("sort"), Limit: limit, Offset: offset}
	items, err := s.Invoices.Search(r.Context(), actor(r), f)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getInvoice(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	inv, err := s.Invoices.Get(r.Context(), actor(r), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, inv)
}

func (s *Server) createInvoice(w http.ResponseWriter, r *http.Request) {
	var in service.NewInvoice
	if !decode(w, r, &in) {
		return
	}
	inv, err := s.Invoices.Create(r.Context(), actor(r), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, inv)
}

func (s *Server) markInvoicePaid(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Invoices.MarkPaid(r.Context(), actor(r), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) voidInvoice(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Invoices.Void(r.Context(), actor(r), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) restoreInvoice(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	inv, err := s.Invoices.Restore(r.Context(), actor(r), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": inv.ID, "number": inv.Number, "status": inv.Status})
}

func (s *Server) replyDraft(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	draft, err := s.Writer.ReplyDraft(r.Context(), actor(r), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"draft": draft})
}

func (s *Server) requestExport(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Kind string     `json:"kind"`
		From *time.Time `json:"from"`
		To   *time.Time `json:"to"`
	}
	if !decode(w, r, &in) {
		return
	}
	jobID, err := s.Invoices.RequestExport(r.Context(), actor(r), in.Kind, in.From, in.To)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"job_id": jobID})
}

// leaveCustomerNote is called from the public payment page.
func (s *Server) leaveCustomerNote(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Note string `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.Invoices.LeaveNote(r.Context(), r.PathValue("token"), in.Note); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
