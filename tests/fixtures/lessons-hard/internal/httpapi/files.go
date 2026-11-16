package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

const maxUpload = 10 << 20

var plainName = regexp.MustCompile(`[^A-Za-z0-9._ -]`)

func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	name := plainName.ReplaceAllString(filepath.Base(r.URL.Query().Get("name")), "_")
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		s.fail(w, err)
		return
	}
	key := hex.EncodeToString(raw)
	dst, err := os.OpenFile(filepath.Join(s.cfg.UploadDir, key), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		s.fail(w, err)
		return
	}
	defer dst.Close()
	sum := sha256.New()
	if _, err := io.Copy(io.MultiWriter(dst, sum), r.Body); err != nil {
		_ = os.Remove(dst.Name())
		writeError(w, http.StatusRequestEntityTooLarge, "file is too large")
		return
	}
	id, err := s.files.Register(r.Context(), actor(r), name, hex.EncodeToString(sum.Sum(nil)), key)
	if err != nil {
		_ = os.Remove(dst.Name())
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (s *Server) downloadFile(w http.ResponseWriter, r *http.Request) {
	key, name, err := s.files.StorageKey(r.Context(), actor(r), pathID(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	http.ServeFile(w, r, filepath.Join(s.cfg.UploadDir, key))
}
