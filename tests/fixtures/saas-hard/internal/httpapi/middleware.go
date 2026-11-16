package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"example.com/tallyboard/internal/repo"
	"example.com/tallyboard/internal/service"
	"example.com/tallyboard/internal/tenant"
)

const maxJSONBody = 1 << 20

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repo.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, service.ErrInvalid):
		writeError(w, http.StatusUnprocessableEntity, "invalid input")
	case errors.Is(err, service.ErrConfirmation):
		writeError(w, http.StatusPreconditionRequired, "confirmation code missing or wrong")
	default:
		log.Printf("httpapi: internal error: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "malformed body")
		return false
	}
	return true
}

func pathID(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	return id, err == nil && id > 0
}

func actor(r *http.Request) *tenant.Actor {
	a, _ := tenant.From(r.Context())
	return a
}

// auth resolves the bearer token (web session or assistant token) to a
// member and attaches it to the request.
func (s *Server) auth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || len(raw) < 32 {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		sum := sha256.Sum256([]byte(raw))
		a, err := s.Sessions.BySessionToken(r.Context(), hex.EncodeToString(sum[:]), time.Now().UTC())
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r.WithContext(tenant.With(r.Context(), a)))
	})
}
