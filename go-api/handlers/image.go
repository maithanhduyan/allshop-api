package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// ServeImage proxies image requests from /api/images/* to MinIO storage
func (h *Handler) ServeImage(w http.ResponseWriter, r *http.Request) {
	if h.storage == nil {
		http.Error(w, "storage not configured", http.StatusServiceUnavailable)
		return
	}

	objectName := chi.URLParam(r, "*")
	if objectName == "" {
		http.Error(w, "missing image path", http.StatusBadRequest)
		return
	}

	// Sanitize: prevent directory traversal
	objectName = strings.TrimPrefix(objectName, "/")
	if strings.Contains(objectName, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	h.storage.ServeImage(w, r, objectName)
}
