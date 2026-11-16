package httpapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"example.com/ledger/internal/platform"
	"example.com/ledger/internal/repo"
)

type ctxKey int

const actorKey ctxKey = 1

func actorFrom(r *http.Request) repo.Actor {
	actor, _ := r.Context().Value(actorKey).(repo.Actor)
	return actor
}

func bearer(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(header[len("Bearer "):])
}

func (a *API) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearer(r)
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
			return
		}
		var actor repo.Actor
		var err error
		if strings.HasPrefix(token, "pat_") {
			actor, err = a.Identity.ActorByAPIToken(r.Context(), platform.HashToken(token))
		} else {
			actor, err = a.Identity.ActorBySession(r.Context(), platform.HashToken(token))
		}
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey, actor)))
	})
}

func (a *API) allow(roles []string, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(roles) > 0 {
			role := actorFrom(r).Role
			permitted := false
			for _, want := range roles {
				if role == want {
					permitted = true
					break
				}
			}
			if !permitted {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
				return
			}
		}
		next(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

const maxLoggedBody = 16 << 10

type replayBody struct {
	io.Reader
	io.Closer
}

func (a *API) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		var body []byte
		if r.Method != http.MethodGet && strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			body, _ = io.ReadAll(io.LimitReader(r.Body, maxLoggedBody))
			r.Body = replayBody{io.MultiReader(bytes.NewReader(body), r.Body), r.Body}
		}
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"ms", time.Since(started).Milliseconds(),
		}
		if rec.status >= 400 && len(body) > 0 {
			attrs = append(attrs, "body", platform.Scrub(body))
		}
		a.Log.Info("request", attrs...)
	})
}
