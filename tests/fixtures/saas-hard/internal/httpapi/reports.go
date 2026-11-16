package httpapi

import "net/http"

func (s *Server) billingReport(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Reports.Billing(r.Context(), actor(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": rows})
}

func (s *Server) loadReport(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Reports.Load(r.Context(), actor(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": rows})
}

func (s *Server) projectDashboard(w http.ResponseWriter, r *http.Request) {
	d, err := s.Reports.ProjectDashboard(r.Context(), actor(r), r.PathValue("code"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}
