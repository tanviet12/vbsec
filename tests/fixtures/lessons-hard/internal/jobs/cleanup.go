package jobs

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type Cleanup struct {
	db        *sql.DB
	uploadDir string
	log       *slog.Logger
}

func NewCleanup(db *sql.DB, uploadDir string, logger *slog.Logger) *Cleanup {
	return &Cleanup{db: db, uploadDir: uploadDir, log: logger}
}

func (c *Cleanup) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.sweepUploads(ctx); err != nil {
				c.log.Error("upload sweep failed", "err", err)
			}
			if err := c.sweepRateHits(ctx); err != nil {
				c.log.Error("rate sweep failed", "err", err)
			}
		}
	}
}

func (c *Cleanup) sweepUploads(ctx context.Context) error {
	rows, err := c.db.QueryContext(ctx,
		"SELECT id, storage_key FROM files WHERE status = 'temp' AND created_at < NOW() - INTERVAL 1 DAY LIMIT 500")
	if err != nil {
		return err
	}
	type stale struct {
		id  int64
		key string
	}
	var batch []stale
	for rows.Next() {
		var s stale
		if err := rows.Scan(&s.id, &s.key); err != nil {
			rows.Close()
			return err
		}
		batch = append(batch, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, s := range batch {
		if err := os.Remove(filepath.Join(c.uploadDir, s.key)); err != nil && !os.IsNotExist(err) {
			return err
		}
		if _, err := c.db.ExecContext(ctx, "DELETE FROM files WHERE id = ? AND status = 'temp'", s.id); err != nil {
			return err
		}
	}
	c.log.Info("upload sweep done", "removed", len(batch))
	return nil
}

func (c *Cleanup) sweepRateHits(ctx context.Context) error {
	_, err := c.db.ExecContext(ctx, "DELETE FROM rate_hits WHERE hit_at < NOW() - INTERVAL 1 DAY")
	return err
}
