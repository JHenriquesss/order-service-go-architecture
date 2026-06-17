package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderTextAllCountersWithAvg(t *testing.T) {
	c := NewCollector()
	c.IncOrdersCreated()
	c.IncOrdersCreated()
	c.IncOrdersProcessed()
	c.RecordProcessingDuration(2.0)
	c.RecordProcessingDuration(4.0)

	body := c.RenderText()
	want := []string{
		"orders_created_total 2",
		"orders_processed_total 1",
		"orders_failed_total 0",
		"orders_processing_duration_seconds_sum 6.00",
		"orders_processing_duration_seconds_count 2",
		"orders_processing_duration_seconds_avg 3.00",
	}
	for _, line := range want {
		if !strings.Contains(body, line) {
			t.Fatalf("missing %q in:\n%s", line, body)
		}
	}
}

func TestRenderTextZeroCountAvgWithoutDivideByZero(t *testing.T) {
	body := NewCollector().RenderText()
	if !strings.Contains(body, "orders_processing_duration_seconds_avg 0.00") {
		t.Fatalf("expected avg 0.00, got:\n%s", body)
	}
}

func TestHandlerGETMetrics(t *testing.T) {
	c := NewCollector()
	c.IncOrdersCreated()
	c.IncOrdersProcessed()
	c.IncOrdersFailed()
	c.RecordProcessingDuration(1.5)

	mux := http.NewServeMux()
	NewHandler(c).Register(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Fatalf("content-type %q", ct)
	}
	for _, line := range []string{
		"orders_created_total 1",
		"orders_processed_total 1",
		"orders_failed_total 1",
		"orders_processing_duration_seconds_sum 1.50",
		"orders_processing_duration_seconds_count 1",
		"orders_processing_duration_seconds_avg 1.50",
	} {
		if !strings.Contains(rec.Body.String(), line) {
			t.Fatalf("missing %q in %q", line, rec.Body.String())
		}
	}
}

func TestCollectorConcurrencySafe(t *testing.T) {
	c := NewCollector()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			c.IncOrdersCreated()
			c.IncOrdersProcessed()
			c.IncOrdersFailed()
			c.RecordProcessingDuration(0.01)
		}
		close(done)
	}()
	for i := 0; i < 100; i++ {
		_ = c.RenderText()
	}
	<-done
	created, processed, failed, _, count := c.Snapshot()
	if created != 100 || processed != 100 || failed != 100 || count != 100 {
		t.Fatalf("got created=%d processed=%d failed=%d count=%d", created, processed, failed, count)
	}
}
