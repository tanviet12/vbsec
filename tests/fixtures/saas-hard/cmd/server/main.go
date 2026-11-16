package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"example.com/tallyboard/internal/assistant"
	"example.com/tallyboard/internal/httpapi"
	"example.com/tallyboard/internal/jobs"
	"example.com/tallyboard/internal/llm"
	"example.com/tallyboard/internal/repo"
	"example.com/tallyboard/internal/service"
)

type smtpMailer struct {
	addr string
	from string
}

func (m smtpMailer) Send(_ context.Context, to, subject, body string) error {
	msg := "From: " + m.from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\n\r\n" + body
	return smtp.SendMail(m.addr, nil, m.from, []string{to}, []byte(msg))
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing %s", key)
	}
	return v
}

func every(d time.Duration, name string, fn func(context.Context) error) {
	go func() {
		for range time.Tick(d) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			if err := fn(ctx); err != nil {
				log.Printf("%s: %v", name, err)
			}
			cancel()
		}
	}()
}

func main() {
	raw, err := sql.Open("mysql", mustEnv("DATABASE_DSN"))
	if err != nil {
		log.Fatal(err)
	}
	db := repo.NewDB(raw)
	mailer := smtpMailer{addr: mustEnv("SMTP_ADDR"), from: mustEnv("MAIL_FROM")}

	projects := repo.NewProjects(db)
	invoices := repo.NewInvoices(db)
	members := repo.NewMembers(db)
	comments := repo.NewComments(db)

	confirm := &service.Confirmations{DB: db, Mail: mailer}
	srv := &httpapi.Server{
		Sessions: members,
		Projects: &service.Projects{Repo: projects, Comments: comments, Confirm: confirm},
		Invoices: &service.Invoices{Repo: invoices, Projects: projects},
		Members:  &service.Members{Repo: members},
		Imports:  &service.Imports{Invoices: invoices, Projects: projects},
		Reports:  service.NewReports(repo.NewReports(db), projects),
		Writer: &assistant.Writer{
			LLM:      llm.New(mustEnv("LLM_ENDPOINT"), mustEnv("LLM_API_KEY"), os.Getenv("LLM_MODEL")),
			Projects: projects, Comments: comments, Invoices: invoices,
		},
	}

	reminders := &jobs.Reminders{Invoices: invoices, System: raw, Mail: mailer}
	exports := &jobs.Exports{Invoices: invoices, Mail: mailer, Dir: mustEnv("EXPORT_DIR")}
	every(24*time.Hour, "reminders", reminders.Run)
	every(time.Minute, "exports", exports.RunOnce)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
