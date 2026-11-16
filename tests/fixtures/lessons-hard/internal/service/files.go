package service

import (
	"context"
	"database/sql"

	"example.com/crewdesk/internal/authn"
)

type Attachment struct {
	FileID int64  `json:"file_id"`
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

type Files struct{ db *sql.DB }

func NewFiles(db *sql.DB) *Files { return &Files{db: db} }

func (s *Files) Register(ctx context.Context, actor authn.Principal, name, sha, storageKey string) (int64, error) {
	if len(name) == 0 || len(name) > 255 {
		return 0, ErrInvalid
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO files (company_id, uploader_id, original_name, sha256, storage_key, status, created_at)
		 VALUES (?, ?, ?, ?, ?, 'temp', NOW())`, actor.CompanyID, actor.UserID, name, sha, storageKey)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Files) AttachReceipt(ctx context.Context, actor authn.Principal, requestID, fileID int64) (Attachment, error) {
	if !actor.Can("requests.create") {
		return Attachment{}, ErrForbidden
	}
	att := Attachment{FileID: fileID}
	err := s.db.QueryRowContext(ctx,
		"SELECT original_name, sha256 FROM files WHERE id = ? AND company_id = ?", fileID, actor.CompanyID).
		Scan(&att.Name, &att.SHA256)
	if err != nil {
		return Attachment{}, ErrNotFound
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE requests SET receipt_file_id = ?
		  WHERE id = ? AND company_id = ? AND requester_id = ? AND status = 'draft'`,
		fileID, requestID, actor.CompanyID, actor.UserID)
	if err != nil {
		return Attachment{}, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return Attachment{}, ErrNotFound
	}
	return att, nil
}

func (s *Files) StorageKey(ctx context.Context, actor authn.Principal, fileID int64) (string, string, error) {
	var key, name string
	err := s.db.QueryRowContext(ctx,
		`SELECT f.storage_key, f.original_name
		   FROM files f
		   LEFT JOIN requests r ON r.receipt_file_id = f.id AND r.company_id = f.company_id
		  WHERE f.id = ? AND f.company_id = ?
		    AND (f.uploader_id = ? OR r.requester_id = ? OR r.approver_id = ? OR ? = 1)
		  LIMIT 1`,
		fileID, actor.CompanyID, actor.UserID, actor.UserID, actor.UserID, boolInt(actor.Can("finance.view"))).
		Scan(&key, &name)
	if err != nil {
		return "", "", ErrNotFound
	}
	return key, name, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
