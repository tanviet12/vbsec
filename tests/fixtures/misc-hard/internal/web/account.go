package web

import (
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"

	"example.com/portal/internal/store"
)

const resetTTL = 30 * time.Minute

func sendMail(to, subject, body string) error {
	msg := "To: " + to + "\r\nSubject: " + subject + "\r\n\r\n" + body
	return smtp.SendMail(os.Getenv("SMTP_ADDR"), nil, "no-reply@portal.example", []string{to}, []byte(msg))
}

func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if err := readJSON(w, r, &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	addr, err := mail.ParseAddress(strings.TrimSpace(body.Email))
	if err != nil {
		http.Error(w, "invalid email", http.StatusUnprocessableEntity)
		return
	}
	email := strings.ToLower(addr.Address)
	if ok, err := s.store.Allow(r.Context(), "forgot:"+email, 3, time.Hour); err != nil || !ok {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	token, err := store.NewToken()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	known, err := s.store.SaveResetToken(r.Context(), email, token, time.Now().Add(resetTTL))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if known {
		link := strings.TrimRight(s.cfg.BaseURL, "/") + "/password/reset#" + token
		if err := sendMail(email, "Reset your password", "Use this link within 30 minutes: "+link); err != nil {
			s.log.Printf("reset mail failed: %v", err)
		} else {
			s.log.Printf("reset mail sent to=%s ip=%s link=%s", email, clientIP(r), link)
		}
	}
	w.WriteHeader(http.StatusAccepted)
}
