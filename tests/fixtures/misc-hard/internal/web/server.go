package web

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"example.com/portal/internal/store"
)

type Config struct {
	BaseURL   string
	UploadDir string
}

type Server struct {
	store *store.Store
	log   *log.Logger
	cfg   Config
}

func New(st *store.Store, logger *log.Logger, cfg Config) *Server {
	return &Server{store: st, log: logger, cfg: cfg}
}

type userKey struct{}

const sessionCookie = "portal_session"

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("POST /password/forgot", s.forgotPassword)
	mux.HandleFunc("GET /language", s.setLanguage)
	mux.Handle("POST /logout", s.signedIn(s.logout))
	mux.Handle("POST /api/feedback", s.signedIn(s.feedback))
	mux.Handle("POST /api/documents/archive", s.signedIn(s.importArchive))
	return mux
}

func (s *Server) signedIn(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		userID, err := s.store.SessionUser(r.Context(), cookie.Value)
		if err != nil {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey{}, userID)))
	})
}

func currentUser(r *http.Request) int64 {
	id, _ := r.Context().Value(userKey{}).(int64)
	return id
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
