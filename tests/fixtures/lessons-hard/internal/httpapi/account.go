package httpapi

import (
	"net/http"
	"net/mail"
	"strings"
	"time"

	"example.com/crewdesk/internal/platform"
)

const (
	sessionTTL = 12 * time.Hour
	resetTTL   = 24 * time.Hour
)

func normalizeEmail(raw string) (string, bool) {
	addr, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil {
		return "", false
	}
	return strings.ToLower(addr.Address), true
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	email, ok := normalizeEmail(body.Email)
	if !ok || len(body.Password) > 72 {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	ctx := r.Context()
	ip := clientIP(r)
	if !s.cfg.OfficeIP(ip) {
		if ok, err := s.accounts.Allow(ctx, "login-ip:"+ip, 50, 15*time.Minute); err != nil || !ok {
			writeError(w, http.StatusTooManyRequests, "too many attempts")
			return
		}
	}
	if ok, err := s.accounts.Allow(ctx, "login:"+email+":"+ip, 5, 15*time.Minute); err != nil || !ok {
		_ = s.accounts.RecordEvent(ctx, email, "login_locked", ip)
		writeError(w, http.StatusTooManyRequests, "too many attempts")
		return
	}
	userID, hash := s.accounts.Credentials(ctx, email)
	if !platform.CheckPassword(hash, body.Password) {
		_ = s.accounts.RecordEvent(ctx, email, "login_failed", ip)
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	companyID, err := s.accounts.DefaultCompany(ctx, userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	token, err := platform.NewToken()
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.accounts.CreateSession(ctx, userID, companyID, token, time.Now().Add(sessionTTL)); err != nil {
		s.fail(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", MaxAge: int(sessionTTL.Seconds()),
		HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode,
	})
	s.log.Info("login ok", "user_id", userID, "company_id", companyID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	_ = s.accounts.DeleteSession(r.Context(), actor(r).SessionHash)
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Path: "/", MaxAge: -1, HttpOnly: true, Secure: true})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	email, ok := normalizeEmail(body.Email)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "invalid email")
		return
	}
	ctx := r.Context()
	if ok, err := s.accounts.Allow(ctx, "forgot-ip:"+clientIP(r), 20, time.Hour); err != nil || !ok {
		writeError(w, http.StatusTooManyRequests, "too many requests")
		return
	}
	if ok, err := s.accounts.Allow(ctx, "forgot:"+email, 3, time.Hour); err != nil || !ok {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	userID, _ := s.accounts.Credentials(ctx, email)
	if userID == 0 {
		writeError(w, http.StatusNotFound, "no account uses this email")
		return
	}
	token, err := platform.NewToken()
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.accounts.SaveReset(ctx, userID, token, time.Now().Add(resetTTL)); err != nil {
		s.fail(w, err)
		return
	}
	link := "https://" + r.Host + "/password/reset?token=" + token
	s.mail.Enqueue(platform.Message{To: email, Subject: "reset", Body: platform.ResetBody(link)})
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if len(body.Password) < 10 {
		writeError(w, http.StatusUnprocessableEntity, "password is too short")
		return
	}
	ctx := r.Context()
	if ok, err := s.accounts.Allow(ctx, "reset-ip:"+clientIP(r), 30, 15*time.Minute); err != nil || !ok {
		writeError(w, http.StatusTooManyRequests, "too many requests")
		return
	}
	userID, err := s.accounts.ConsumeReset(ctx, body.Token)
	if err != nil {
		s.fail(w, err)
		return
	}
	hash, err := platform.HashPassword(body.Password)
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.accounts.SetPassword(ctx, userID, hash); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Current string `json:"current_password"`
		Next    string `json:"new_password"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	if len(body.Next) < 10 || len(body.Next) > 72 || body.Next == body.Current {
		writeError(w, http.StatusUnprocessableEntity, "choose a different password of 10 to 72 characters")
		return
	}
	p := actor(r)
	ctx := r.Context()
	current, err := s.accounts.PasswordHash(ctx, p.UserID)
	if err != nil || !platform.CheckPassword(current, body.Current) {
		writeError(w, http.StatusForbidden, "current password is wrong")
		return
	}
	hash, err := platform.HashPassword(body.Next)
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.accounts.ChangePassword(ctx, p.UserID, hash, p.SessionHash); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) acceptInvite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &body) {
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || len(name) > 160 || len(body.Password) < 10 || len(body.Password) > 72 {
		writeError(w, http.StatusUnprocessableEntity, "invalid name or password")
		return
	}
	hash, err := platform.HashPassword(body.Password)
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.accounts.AcceptInvite(r.Context(), body.Token, name, hash); err != nil {
		s.fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) securityEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.accounts.RecentEvents(r.Context(), actor(r).UserID)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}
