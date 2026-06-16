package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"order-service-go/internal/logger"
)

// statusWriter captures the response status code for logging.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Logging logs one structured record per request with the canonical fields
// (architecture §23): request_id, method, path, status_code, duration_ms.
func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			log.Info("request completed",
				logger.FieldRequestID, RequestIDFromContext(r.Context()),
				logger.FieldMethod, r.Method,
				logger.FieldPath, r.URL.Path,
				logger.FieldStatusCode, sw.status,
				logger.FieldDurationMS, time.Since(start).Milliseconds(),
			)
		})
	}
}
