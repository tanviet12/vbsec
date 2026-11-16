package platform

import (
	"fmt"
	"log/slog"
	"net/mail"
	"net/smtp"
)

type Message struct {
	To      string
	Subject string
	Body    string
}

type Mailer struct {
	addr  string
	from  string
	queue chan Message
	log   *slog.Logger
}

func NewMailer(addr string, logger *slog.Logger) *Mailer {
	m := &Mailer{addr: addr, from: "no-reply@crewdesk.example", queue: make(chan Message, 256), log: logger}
	for i := 0; i < 4; i++ {
		go m.run()
	}
	return m
}

func (m *Mailer) Enqueue(msg Message) {
	select {
	case m.queue <- msg:
	default:
	}
}

func (m *Mailer) run() {
	for msg := range m.queue {
		if err := m.send(msg); err != nil {
			m.log.Error("mail delivery failed", "subject", msg.Subject, "err", err)
		}
	}
}

func (m *Mailer) send(msg Message) error {
	to, err := mail.ParseAddress(msg.To)
	if err != nil {
		return fmt.Errorf("recipient %q: %w", msg.To, err)
	}
	subject, ok := subjects[msg.Subject]
	if !ok {
		return fmt.Errorf("unknown subject key %q", msg.Subject)
	}
	raw := "From: " + m.from + "\r\nTo: " + to.Address + "\r\nSubject: " + subject +
		"\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + msg.Body
	if err := smtp.SendMail(m.addr, nil, m.from, []string{to.Address}, []byte(raw)); err != nil {
		return fmt.Errorf("smtp %s: %w", to.Address, err)
	}
	return nil
}

var subjects = map[string]string{
	"reset":    "Reset your password",
	"invite":   "You have been invited to a workspace",
	"security": "Security notice for your account",
}

func ResetBody(link string) string {
	return "Use the link below within 24 hours to choose a new password.\n\n" + link + "\n"
}

func InviteBody(company, inviter, link string) string {
	return inviter + " invited you to join " + company + " on Crewdesk.\n\n" +
		"Accept the invitation: " + link + "\n"
}

func TwoFactorResetBody(company, admin string) string {
	return "Two-step verification on your account was reset by " + admin + " (" + company + ").\n\n" +
		"If you did not ask for this, change your password and contact your workspace owner.\n"
}
