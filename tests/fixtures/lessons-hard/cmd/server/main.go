package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"example.com/crewdesk/internal/httpapi"
	"example.com/crewdesk/internal/jobs"
	"example.com/crewdesk/internal/platform"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := platform.Load()
	if err != nil {
		logger.Error("invalid configuration", "err", err)
		os.Exit(1)
	}
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		logger.Error("database", "err", err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(20)

	go jobs.NewCleanup(db, cfg.UploadDir, logger).Run(context.Background())

	root := http.NewServeMux()
	root.Handle("/", httpapi.New(db, cfg, logger).Routes())
	if !cfg.Production() {
		root.HandleFunc("/debug/pprof/", pprof.Index)
		root.HandleFunc("/debug/pprof/profile", pprof.Profile)
		root.HandleFunc("/debug/config", func(w http.ResponseWriter, r *http.Request) {
			logger.Info("config dump requested")
			_, _ = w.Write([]byte(cfg.Env + "\n" + cfg.BaseURL + "\n" + cfg.DSN + "\n"))
		})
	}
	server := &http.Server{
		Addr:              ":8080",
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
	}
	logger.Info("listening", "addr", server.Addr, "env", cfg.Env)
	if err := server.ListenAndServe(); err != nil {
		logger.Error("server stopped", "err", err)
	}
}
