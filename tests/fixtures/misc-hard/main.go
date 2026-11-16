package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"example.com/portal/internal/store"
	"example.com/portal/internal/web"
)

func main() {
	db, err := sql.Open("mysql", os.Getenv("DATABASE_DSN"))
	if err != nil {
		log.Fatal(err)
	}
	srv := web.New(store.New(db), log.New(os.Stdout, "portal ", log.LstdFlags|log.LUTC), web.Config{
		BaseURL:   os.Getenv("APP_BASE_URL"),
		UploadDir: os.Getenv("UPLOAD_DIR"),
	})
	server := &http.Server{
		Addr:              ":8080",
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
