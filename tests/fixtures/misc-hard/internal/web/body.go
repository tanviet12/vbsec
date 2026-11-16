package web

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const maxBody = 1 << 20

var errBadBody = errors.New("invalid request body")

func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	var src io.Reader = http.MaxBytesReader(w, r.Body, maxBody)
	if strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
		zr, err := gzip.NewReader(src)
		if err != nil {
			return errBadBody
		}
		defer zr.Close()
		src = zr
	}
	raw, err := io.ReadAll(src)
	if err != nil {
		return errBadBody
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return errBadBody
	}
	return nil
}

func (s *Server) feedback(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Topic   string `json:"topic"`
		Message string `json:"message"`
	}
	if err := readJSON(w, r, &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Topic == "" || len(body.Topic) > 80 || len(body.Message) > 4000 {
		http.Error(w, "invalid feedback", http.StatusUnprocessableEntity)
		return
	}
	if err := s.store.SaveFeedback(r.Context(), currentUser(r), body.Topic, body.Message); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
