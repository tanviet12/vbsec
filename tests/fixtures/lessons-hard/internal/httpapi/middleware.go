package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"

	"example.com/crewdesk/internal/agentapi"
	"example.com/crewdesk/internal/authn"
	"example.com/crewdesk/internal/platform"
	"example.com/crewdesk/internal/repo"
	"example.com/crewdesk/internal/service"
)

const sessionCookie = "crewdesk_session"

type principalKey struct{}
type operatorKey struct{}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *Server) signedIn(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		p, err := authn.FromSession(r.Context(), s.db, cookie.Value)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		if r.Method != http.MethodGet && r.Header.Get("X-Requested-With") != "crewdesk-web" {
			writeError(w, http.StatusForbidden, "missing request marker")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, p)))
	})
}

func (s *Server) operator(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		operatorID, err := strconv.ParseInt(r.Header.Get("X-Operator-Id"), 10, 64)
		digest := platform.Digest(token)
		if err != nil || s.cfg.OpsTokenHash == "" ||
			subtle.ConstantTimeCompare([]byte(digest), []byte(s.cfg.OpsTokenHash)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		op := service.Operator{ID: operatorID}
		next(w, r.WithContext(context.WithValue(r.Context(), operatorKey{}, op)))
	})
}

func actor(r *http.Request) authn.Principal {
	p, _ := r.Context().Value(principalKey{}).(authn.Principal)
	return p
}

func pathID(r *http.Request) int64 {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden), errors.Is(err, agentapi.ErrForbidden), errors.Is(err, agentapi.ErrScope):
		writeError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, service.ErrNotFound), errors.Is(err, agentapi.ErrUnknownTool):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, service.ErrConflict):
		writeError(w, http.StatusConflict, "conflict")
	case errors.Is(err, service.ErrInvalid), errors.Is(err, agentapi.ErrBadArgs), errors.Is(err, repo.ErrInvalidToken):
		writeError(w, http.StatusUnprocessableEntity, "invalid input")
	case errors.Is(err, authn.ErrNoSession):
		writeError(w, http.StatusUnauthorized, "unauthenticated")
	default:
		s.log.Error("request failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
