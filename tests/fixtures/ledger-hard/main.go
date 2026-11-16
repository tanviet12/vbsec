package main

import (
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"example.com/ledger/internal/audit"
	"example.com/ledger/internal/httpapi"
	"example.com/ledger/internal/platform"
	"example.com/ledger/internal/repo"
	"example.com/ledger/internal/service"
)

func mustKey(name string) []byte {
	key, err := hex.DecodeString(os.Getenv(name))
	if err != nil || len(key) != 32 {
		log.Fatalf("%s must be 32 bytes hex", name)
	}
	return key
}

func main() {
	db, err := sql.Open("mysql", os.Getenv("DATABASE_DSN"))
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(5 * time.Minute)

	logger := platform.NewLogger()
	box, err := platform.NewBox(mustKey("FIELD_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	identity := repo.NewIdentityRepo(db)
	entries := repo.NewEntryRepo()
	periods := repo.NewPeriodRepo()
	requests := repo.NewExpenseRepo()
	funds := repo.NewFundRepo()
	staff := repo.NewStaffRepo()
	trail := audit.New(db)

	api := httpapi.New(httpapi.Deps{
		Log:      logger,
		Identity: identity,
		Auth:     service.NewAuth(db, identity, box, mustKey("TICKET_KEY"), logger),
		Entries:  service.NewEntries(db, entries, periods, trail),
		Imports:  service.NewImports(db, entries, periods, trail),
		Periods:  service.NewPeriods(db, periods, requests, trail),
		Expenses: service.NewExpenses(db, requests, funds, staff, periods, trail),
		Funds:    service.NewFunds(db, funds, trail),
		Staff: service.NewStaff(db, staff, identity, box, trail, logger,
			platform.NewMailer(os.Getenv("SMTP_ADDR")), os.Getenv("APP_BASE_URL")),
		Payments: service.NewPayments(db, funds, requests, []byte(os.Getenv("HOOK_SECRET"))),
	})

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
