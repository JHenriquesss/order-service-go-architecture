package metrics

import (
	"net/http"
)

// Handler serves GET /metrics in architecture Â§22 text format.
type Handler struct {
	collector *Collector
}

// NewHandler builds the metrics HTTP handler.
func NewHandler(collector *Collector) *Handler {
	return &Handler{collector: collector}
}

// Register mounts GET /metrics on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /metrics", h.ServeHTTP)
}

// ServeHTTP renders the metrics snapshot as plain text.
func (h *Handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(h.collector.RenderText()))
}
