package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"order-service-worker/internal/metrics"
	"order-service-worker/internal/order"
)

func TestPublicMetricsEndpoint(t *testing.T) {
	c := metrics.NewCollector()
	c.IncOrdersCreated()
	h := order.NewHandler(order.NewService(order.NewInMemoryRepository(), nil, nil, nil, nil, c, nil))
	srv := NewWithPublicMetrics(fakeVerifier{}, h, metrics.NewHandler(c))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", nil)
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "orders_created_total 1") {
		t.Fatalf("body %q", rec.Body.String())
	}
}
