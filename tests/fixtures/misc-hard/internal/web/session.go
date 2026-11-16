package web

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"example.com/portal/internal/store"
)

const sessionTTL = 8 * time.Hour

var languages = map[string]bool{"vi": true, "en": true}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.PostForm.Get("email")))
	ok, err := s.store.Allow(r.Context(), "login:"+email+":"+clientIP(r), 5, 15*time.Minute)
	if err != nil || !ok {
		http.Error(w, "too many attempts", http.StatusTooManyRequests)
		return
	}
	userID, err := s.store.CheckPassword(r.Context(), email, r.PostForm.Get("password"))
	if err != nil {
		http.Redirect(w, r, "/login?failed=1", http.StatusSeeOther)
		return
	}
	token, err := store.NewToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := s.store.CreateSession(r.Context(), userID, token, time.Now().Add(sessionTTL)); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", MaxAge: int(sessionTTL.Seconds()),
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	})
	s.log.Printf("login ok user=%d ip=%s", userID, clientIP(r))

	target := "/dashboard"
	if next := r.URL.Query().Get("next"); strings.HasPrefix(next, "/") {
		target = next
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		_ = s.store.DeleteSession(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Path: "/", MaxAge: -1, HttpOnly: true, Secure: true})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func localPath(raw string) (string, bool) {
	if raw == "" || strings.ContainsAny(raw, "\\\r\n") {
		return "", false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil {
		return "", false
	}
	if !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") {
		return "", false
	}
	return u.RequestURI(), true
}

func (s *Server) setLanguage(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	if !languages[lang] {
		lang = "vi"
	}
	http.SetCookie(w, &http.Cookie{Name: "lang", Value: lang, Path: "/", MaxAge: 365 * 24 * 3600, SameSite: http.SameSiteLaxMode})
	target, ok := localPath(r.URL.Query().Get("back"))
	if !ok {
		target = "/"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
