//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestOrderProcessingFailureReachesFailed(t *testing.T) {
	dsn, _, apiBase := requireE2EEnv(t)
	workerBase := requireWorkerMetricsEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	_ = connectPostgres(t, ctx, dsn)
	client := newAPIClient(apiBase)
	client.login(t, "admin@example.com", "123456")

	beforeFailedBody := fetchWorkerMetrics(t, workerBase)
	beforeFailed, _ := parseMetricValue(beforeFailedBody, "orders_failed_total")

	suffix := uniqueSuffix()
	var customer customerResponse
	client.doJSON(t, http.MethodPost, "/api/customers", map[string]any{
		"name":     "Fail Customer " + suffix,
		"document": "fail-" + suffix,
		"email":    "fail-" + suffix + "@example.com",
		"phone":    "+55 11 90000-0001",
	}, http.StatusCreated, &customer)

	var product productResponse
	client.doJSON(t, http.MethodPost, "/api/products", map[string]any{
		"name":  "Fail Product " + suffix,
		"sku":   "FAIL-SKU-" + suffix,
		"price": 13.37,
	}, http.StatusCreated, &product)

	var order orderResponse
	client.doJSON(t, http.MethodPost, "/api/orders", map[string]any{
		"customer_id": customer.ID,
		"items": []map[string]any{
			{"product_id": product.ID, "quantity": 1},
		},
	}, http.StatusCreated, &order)

	if order.Status != "CREATED" {
		t.Fatalf("expected CREATED, got %s", order.Status)
	}

	final := waitForOrderStatus(t, client, order.ID, "FAILED", 30*time.Second)
	if final.Status != "FAILED" {
		t.Fatalf("expected FAILED, got %s", final.Status)
	}

	deadline := time.Now().Add(30 * time.Second)
	var afterFailed int64
	for time.Now().Before(deadline) {
		body := fetchWorkerMetrics(t, workerBase)
		if v, ok := parseMetricValue(body, "orders_failed_total"); ok && v > beforeFailed {
			afterFailed = v
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if afterFailed <= beforeFailed {
		t.Fatalf("worker orders_failed_total did not increase: before=%d after=%d", beforeFailed, afterFailed)
	}
}

func TestWorkerHealthAndMetricsEndpoints(t *testing.T) {
	workerBase := requireWorkerMetricsEnv(t)
	client := &http.Client{Timeout: 10 * time.Second}

	healthReq, err := http.NewRequest(http.MethodGet, workerBase+"/health", nil)
	if err != nil {
		t.Fatalf("new health request: %v", err)
	}
	healthRes, err := client.Do(healthReq)
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer healthRes.Body.Close()
	if healthRes.StatusCode != http.StatusOK {
		t.Fatalf("GET /health status %d", healthRes.StatusCode)
	}

	_ = fetchWorkerMetrics(t, workerBase)
}
