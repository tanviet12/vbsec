package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/smtp"
	"os"
	"strings"
)

func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

var scrubbed = map[string]bool{
	"password":         true,
	"current_password": true,
	"new_password":     true,
	"code":             true,
	"token":            true,
	"secret":           true,
}

func Scrub(body []byte) string {
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		return fmt.Sprintf("(%d bytes)", len(body))
	}
	for key := range doc {
		if scrubbed[strings.ToLower(key)] {
			doc[key] = "***"
		}
	}
	out, _ := json.Marshal(doc)
	return string(out)
}

type Mailer struct{ addr string }

func NewMailer(addr string) *Mailer { return &Mailer{addr: addr} }

func (m *Mailer) Send(_ context.Context, to, subject, body string) error {
	if strings.ContainsAny(to+subject, "\r\n") {
		return fmt.Errorf("invalid header value")
	}
	msg := "To: " + to + "\r\nSubject: " + subject + "\r\n\r\n" + body
	return smtp.SendMail(m.addr, nil, "no-reply@ledger.example", []string{to}, []byte(msg))
}
