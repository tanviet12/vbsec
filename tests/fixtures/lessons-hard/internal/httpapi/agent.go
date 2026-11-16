package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (s *Server) connectAgent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClientID string   `json:"client_id"`
		Scopes   []string `json:"scopes"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	token, err := s.agent.Connect(r.Context(), actor(r), body.ClientID, body.Scopes)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"token": token})
}

func (s *Server) disconnectAgent(w http.ResponseWriter, r *http.Request) {
	s.done(w, s.agent.Disconnect(r.Context(), actor(r), pathID(r)))
}

func (s *Server) approvePayrollGrant(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ClientID string `json:"client_id"`
		UserID   int64  `json:"user_id"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	s.done(w, s.agent.ApprovePayrollGrant(r.Context(), actor(r), body.ClientID, body.UserID))
}

func (s *Server) agentCall(w http.ResponseWriter, r *http.Request) {
	bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	conn, err := s.agent.Authenticate(r.Context(), bearer)
	if err != nil {
		if ok, _ := s.accounts.Allow(r.Context(), "agent-auth:"+clientIP(r), 60, 15*time.Minute); !ok {
			writeError(w, http.StatusTooManyRequests, "too many requests")
			return
		}
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	var body struct {
		Tool string          `json:"tool"`
		Args json.RawMessage `json:"arguments"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	result, err := s.agent.Call(r.Context(), conn, body.Tool, body.Args)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": result})
}
