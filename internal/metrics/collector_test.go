package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestCollectorCounterIncrements(t *testing.T) {
	c := NewCollector()
	c.IncOrdersCreated()
	c.IncOrdersCreated()
	c.IncOrdersProcessed()
	c.IncOrdersFailed()

	if got := testutil.ToFloat64(c.ordersCreated); got != 2 {
		t.Fatalf("orders_created_total = %v, want 2", got)
	}
	if got := testutil.ToFloat64(c.ordersProcessed); got != 1 {
		t.Fatalf("orders_processed_total = %v, want 1", got)
	}
	if got := testutil.ToFloat64(c.ordersFailed); got != 1 {
		t.Fatalf("orders_failed_total = %v, want 1", got)
	}
}

func TestCollectorHistogramObservations(t *testing.T) {
	c := NewCollector()
	c.RecordProcessingDuration(2.0)
	c.RecordProcessingDuration(4.0)

	histCount, histSum := histogramSamples(t, c)
	if histCount != 2 {
		t.Fatalf("histogram sample_count = %v, want 2", histCount)
	}
	if histSum != 6.0 {
		t.Fatalf("histogram sample_sum = %v, want 6.0", histSum)
	}
}

func histogramSamples(t *testing.T, c *Collector) (count int64, sum float64) {
	t.Helper()
	mfs, err := c.reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != "orders_processing_duration_seconds" {
			continue
		}
		for _, m := range mf.GetMetric() {
			if h := m.GetHistogram(); h != nil {
				return int64(h.GetSampleCount()), h.GetSampleSum()
			}
		}
	}
	t.Fatal("histogram metric missing")
	return 0, 0
}

func TestCollectorHistogramZeroObservations(t *testing.T) {
	c := NewCollector()
	mfs, err := c.reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != "orders_processing_duration_seconds" {
			continue
		}
		for _, m := range mf.GetMetric() {
			if h := m.GetHistogram(); h != nil {
				if h.GetSampleCount() != 0 || h.GetSampleSum() != 0 {
					t.Fatalf("zero observations want count=0 sum=0, got count=%d sum=%v",
						h.GetSampleCount(), h.GetSampleSum())
				}
				return
			}
		}
	}
	t.Fatal("histogram metric missing")
}

func TestTwoCollectorsDoNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("building two collectors panicked: %v", r)
		}
	}()
	_ = NewCollector()
	_ = NewCollector()
}

func TestHandlerPrometheusExposition(t *testing.T) {
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
	body := rec.Body.String()
	for _, want := range []string{
		"# TYPE orders_created_total counter",
		"orders_created_total",
		"orders_processed_total",
		"orders_failed_total",
		"orders_processing_duration_seconds_sum",
		"orders_processing_duration_seconds_count",
		"orders_processing_duration_seconds_bucket",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in:\n%s", want, body)
		}
	}
	if strings.Contains(body, "orders_processing_duration_seconds_avg") {
		t.Fatalf("avg must not be exposed in Prometheus format:\n%s", body)
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
		rec := httptest.NewRecorder()
		c.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	}
	<-done
	if got := testutil.ToFloat64(c.ordersCreated); got != 100 {
		t.Fatalf("created=%v", got)
	}
	if got := testutil.ToFloat64(c.ordersProcessed); got != 100 {
		t.Fatalf("processed=%v", got)
	}
	if got := testutil.ToFloat64(c.ordersFailed); got != 100 {
		t.Fatalf("failed=%v", got)
	}
	count, sum := histogramSamples(t, c)
	if count != 100 {
		t.Fatalf("duration count=%v", count)
	}
	if sum < 1.0 {
		t.Fatalf("duration sum=%v", sum)
	}
}
