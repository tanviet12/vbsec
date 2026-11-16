package httpapi

import (
	"net/http"

	"example.com/tallyboard/internal/authz"
)

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	a := actor(r)
	projects, err := s.Projects.List(r.Context(), a, r.URL.Query().Get("sort"), r.URL.Query().Get("archived") == "1")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	out := make([]map[string]any, 0, len(projects))
	for _, p := range projects {
		out = append(out, authz.ProjectView(a, p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": out})
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	a := actor(r)
	p, tasks, err := s.Projects.Get(r.Context(), a, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project":    p,
		"tasks":      tasks,
		"can_manage": authz.CanManageProject(a, p),
	})
}

func (s *Server) archiveProject(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Projects.Archive(r.Context(), actor(r), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) restoreProject(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := s.Projects.Restore(r.Context(), actor(r), id); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// requestProjectDelete mails a one-time code to the caller. The response
// only says where the code went.
func (s *Server) requestProjectDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	expires, err := s.Projects.RequestDelete(r.Context(), actor(r), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"delivery": "email", "expires_at": expires})
}

func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	code := r.Header.Get("X-Confirmation-Code")
	if err := s.Projects.Delete(r.Context(), actor(r), id, code); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) addExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	var in struct {
		AmountCents int64  `json:"amount_cents"`
		Note        string `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	e, p, spent, err := s.Projects.AddExpense(r.Context(), actor(r), id, in.AmountCents, in.Note)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"expense":         e,
		"project_code":    p.Code,
		"spent_cents":     spent,
		"remaining_cents": p.BudgetCents - spent,
	})
}

func (s *Server) projectDigest(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	text, err := s.Writer.ProjectDigest(r.Context(), actor(r), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"digest": text})
}
