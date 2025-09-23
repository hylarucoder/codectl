package server

import (
	"encoding/json"
	"net/http"

	"codectl/internal/provider"
	appver "codectl/internal/version"
)

func mountAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"version": appver.AppVersion})
	})
	mux.HandleFunc("/api/providers", providersHandler)

	// FS
	mux.HandleFunc("/api/fs/tree", fsTreeHandler)
	mux.HandleFunc("/api/fs/read", fsReadHandler)
	mux.HandleFunc("/api/fs/write", fsWriteHandler)
	mux.HandleFunc("/api/fs/rename", fsRenameHandler)
	mux.HandleFunc("/api/fs/delete", fsDeleteHandler)
	mux.HandleFunc("/api/fs/patch", fsPatchHandler)

	// Spec
	mux.HandleFunc("/api/spec/docs", specListHandler)
	mux.HandleFunc("/api/spec/doc", specDocHandler)
	mux.HandleFunc("/api/spec/validate", specValidateHandler)

	// Sessions (in-memory MVP)
	mux.HandleFunc("/api/sessions", sessionsRootHandler)
	mux.HandleFunc("/api/sessions/", sessionItemHandler) // /api/sessions/{id}/...
}

func providersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cat, err := provider.LoadV2()
		if err != nil {
			// Still return what we have along with 200; LoadV2 already defaulted
		}
		writeJSON(w, http.StatusOK, cat)
	case http.MethodPut:
		var cat provider.CatalogV2
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&cat); err != nil {
			writeJSON(w, http.StatusBadRequest, errJSON(err))
			return
		}
		if err := provider.SaveV2(cat); err != nil {
			writeJSON(w, http.StatusInternalServerError, errJSON(err))
			return
		}
		writeJSON(w, http.StatusOK, cat)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if v == nil {
		return
	}
	if err, ok := v.(error); ok {
		_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func errJSON(err error) map[string]string { return map[string]string{"error": err.Error()} }
