package httpapi

import (
	"net/http"
	"time"

	"example.com/crewdesk/internal/service"
)

func (s *Server) done(w http.ResponseWriter, err error) {
	if err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	var patch map[string]any
	if !readJSON(w, r, &patch) {
		return
	}
	s.done(w, s.members.UpdateProfile(r.Context(), actor(r), patch))
}

func (s *Server) inviteMember(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	s.done(w, s.members.Invite(r.Context(), actor(r), body.Email, body.Role))
}

func (s *Server) removeMember(w http.ResponseWriter, r *http.Request) {
	s.done(w, s.members.Remove(r.Context(), actor(r), pathID(r)))
}

func (s *Server) reinstateMember(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Role string `json:"role"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	s.done(w, s.members.Reinstate(r.Context(), actor(r), pathID(r), body.Role))
}

func (s *Server) setPermissions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Permissions []string `json:"permissions"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	s.done(w, s.members.SetPermissions(r.Context(), actor(r), pathID(r), body.Permissions))
}

func (s *Server) resetTwoFactor(w http.ResponseWriter, r *http.Request) {
	s.done(w, s.members.ResetTwoFactor(r.Context(), actor(r), pathID(r)))
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	project, err := s.projects.Get(r.Context(), actor(r), pathID(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	var patch service.ProjectPatch
	if !readJSON(w, r, &patch) {
		return
	}
	project, err := s.projects.Update(r.Context(), actor(r), pathID(r), patch)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) addNote(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Body string `json:"body"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	s.done(w, s.projects.AddNote(r.Context(), actor(r), pathID(r), body.Body))
}

func (s *Server) closeProject(w http.ResponseWriter, r *http.Request) {
	s.done(w, s.projects.Close(r.Context(), actor(r), pathID(r)))
}

func (s *Server) createRequest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AmountCents int64 `json:"amount_cents"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	id, err := s.requests.Create(r.Context(), actor(r), body.AmountCents)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (s *Server) listApprovers(w http.ResponseWriter, r *http.Request) {
	approvers, err := s.requests.Approvers(r.Context(), actor(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, approvers)
}

func (s *Server) attachReceipt(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FileID int64 `json:"file_id"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	attachment, err := s.files.AttachReceipt(r.Context(), actor(r), pathID(r), body.FileID)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, attachment)
}

func (s *Server) submitRequest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ApproverID int64 `json:"approver_id"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	s.done(w, s.requests.Submit(r.Context(), actor(r), pathID(r), body.ApproverID))
}

func (s *Server) decideRequest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Approve bool `json:"approve"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	s.done(w, s.requests.Decide(r.Context(), actor(r), pathID(r), body.Approve))
}

func (s *Server) setApprovalMode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Mode string `json:"mode"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	s.done(w, s.settings.SetApprovalMode(r.Context(), actor(r), body.Mode))
}

func (s *Server) setRetention(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Days int `json:"days"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	s.done(w, s.settings.SetRetention(r.Context(), actor(r), body.Days))
}

func (s *Server) setMonthlyLimit(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Cents int64 `json:"cents"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	s.done(w, s.settings.SetMonthlyLimit(r.Context(), actor(r), body.Cents))
}

func (s *Server) createCategory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	id, err := s.categories.Create(r.Context(), actor(r), body.Name)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (s *Server) importExpenses(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Rows []service.ExpenseRow `json:"rows"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	count, err := s.imports.Expenses(r.Context(), actor(r), body.Rows)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"imported": count})
}

func (s *Server) searchActivity(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	from, err := time.Parse("2006-01-02", query.Get("from"))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "from must be a date")
		return
	}
	var to time.Time
	if raw := query.Get("to"); raw != "" {
		if to, err = time.Parse("2006-01-02", raw); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "to must be a date")
			return
		}
	}
	rows, err := s.activity.Search(r.Context(), actor(r), from, to, query.Get("q"), 100)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) opsOverview(w http.ResponseWriter, r *http.Request) {
	op, _ := r.Context().Value(operatorKey{}).(service.Operator)
	overview, err := s.ops.Overview(r.Context(), op, pathID(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func (s *Server) opsRequests(w http.ResponseWriter, r *http.Request) {
	op, _ := r.Context().Value(operatorKey{}).(service.Operator)
	rows, err := s.ops.Requests(r.Context(), op, pathID(r), r.URL.Query().Get("status"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}
