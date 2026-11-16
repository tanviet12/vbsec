package web

import (
	"archive/zip"
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

const (
	maxArchiveBytes = 8 << 20
	maxArchiveFiles = 50
	maxEntryBytes   = 5 << 20
	maxTotalBytes   = 40 << 20
)

var documentName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _.-]{0,80}\.(pdf|png|jpg|csv)$`)

func (s *Server) importArchive(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxArchiveBytes))
	if err != nil {
		http.Error(w, "archive too large", http.StatusRequestEntityTooLarge)
		return
	}
	archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		http.Error(w, "invalid archive", http.StatusBadRequest)
		return
	}
	if len(archive.File) > maxArchiveFiles {
		http.Error(w, "too many files", http.StatusUnprocessableEntity)
		return
	}
	userID := currentUser(r)
	dir := filepath.Join(s.cfg.UploadDir, strconv.FormatInt(userID, 10))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	var total int64
	saved := 0
	for _, entry := range archive.File {
		name := filepath.Base(entry.Name)
		if entry.FileInfo().IsDir() || !documentName.MatchString(name) {
			continue
		}
		size, err := extractEntry(entry, filepath.Join(dir, name), maxTotalBytes-total)
		if err != nil {
			http.Error(w, "archive rejected", http.StatusUnprocessableEntity)
			return
		}
		total += size
		if err := s.store.AddDocument(r.Context(), userID, name, size); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		saved++
	}
	writeJSON(w, http.StatusOK, map[string]int{"saved": saved})
}

func extractEntry(entry *zip.File, dest string, budget int64) (int64, error) {
	limit := int64(maxEntryBytes)
	if budget < limit {
		limit = budget
	}
	src, err := entry.Open()
	if err != nil {
		return 0, err
	}
	defer src.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	written, err := io.Copy(out, io.LimitReader(src, limit+1))
	if err != nil {
		return written, err
	}
	if written > limit {
		_ = os.Remove(dest)
		return written, io.ErrShortBuffer
	}
	return written, nil
}
