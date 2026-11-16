package httpapi

import "net/http"

func (s *Server) listComments(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	comments, err := s.Projects.ListComments(r.Context(), actor(r), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"comments": comments})
}

func (s *Server) addComment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	var in struct {
		Body       string `json:"body"`
		Visibility string `json:"visibility"`
	}
	if !decode(w, r, &in) {
		return
	}
	c, err := s.Projects.AddComment(r.Context(), actor(r), id, in.Body, in.Visibility)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}
