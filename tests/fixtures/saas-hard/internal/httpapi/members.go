package httpapi

import "net/http"

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	members, err := s.Members.List(r.Context(), actor(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": members})
}

func (s *Server) setMemberRole(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	var in struct {
		Role string `json:"role"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.Members.SetRole(r.Context(), actor(r), id, in.Role); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) setMemberPermissions(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	var in struct {
		Permissions []string `json:"permissions"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.Members.SetPermissions(r.Context(), actor(r), id, in.Permissions); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) copyMemberPermissions(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	var in struct {
		FromMemberID int64 `json:"from_member_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	perms, err := s.Members.CopyPermissions(r.Context(), actor(r), in.FromMemberID, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"permissions": perms})
}

func (s *Server) acceptInvite(w http.ResponseWriter, r *http.Request) {
	session, err := s.Members.AcceptInvite(r.Context(), r.PathValue("token"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"session_token": session})
}
