package httpapi

import "net/http"

const maxImportBytes = 5 << 20

func (s *Server) importInvoices(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes)
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "csv file required")
		return
	}
	defer file.Close()
	res, err := s.Imports.ImportInvoices(r.Context(), actor(r), file)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
