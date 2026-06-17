package metrics

import (
	"net/http"
)

// Handler serves GET /metrics in Prometheus exposition format.
type Handler struct {
	collector *Collector
}

// NewHandler builds the metrics HTTP handler.
func NewHandler(collector *Collector) *Handler {
	return &Handler{collector: collector}
}

// Register mounts GET /metrics on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.Handle("GET /metrics", h.collector.Handler())
}

// ServeHTTP renders Prometheus exposition for the collector.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.collector.Handler().ServeHTTP(w, r)
}
