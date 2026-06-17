//go:build integration

package integration

import (
	"net/http"
	"testing"
	"time"
)

func TestE2EWorkflowReachesPaidAndIncrementsMetrics(t *testing.T) {
	apiBase := requireAPIEnv(t)
	client := newAPIClient(apiBase)

	before := client.fetchMetrics(t)
	beforeCreated, _ := parseMetricValue(before, "orders_created_total")

	client.login(t, "admin@example.com", "123456")

	suffix := uniqueSuffix()
	var customer customerResponse
	client.doJSON(t, http.MethodPost, "/api/customers", map[string]any{
		"name":     "E2E Customer " + suffix,
		"document": "e2e-" + suffix,
		"email":    "e2e-" + suffix + "@example.com",
		"phone":    "+55 11 90000-0000",
	}, http.StatusCreated, &customer)

	var product productResponse
	client.doJSON(t, http.MethodPost, "/api/products", map[string]any{
		"name":  "E2E Product " + suffix,
		"sku":   "E2E-SKU-" + suffix,
		"price": 49.90,
	}, http.StatusCreated, &product)

	var order orderResponse
	client.doJSON(t, http.MethodPost, "/api/orders", map[string]any{
		"customer_id": customer.ID,
		"items": []map[string]any{
			{"product_id": product.ID, "quantity": 2},
		},
	}, http.StatusCreated, &order)

	if order.Status != "CREATED" {
		t.Fatalf("expected CREATED, got %s", order.Status)
	}

	final := waitForOrderStatus(t, client, order.ID, "PAID", 30*time.Second)
	if final.Status != "PAID" {
		t.Fatalf("expected PAID, got %s", final.Status)
	}
	if final.TotalAmount <= 0 {
		t.Fatalf("expected positive total_amount, got %v", final.TotalAmount)
	}

	after := client.fetchMetrics(t)
	afterCreated, ok := parseMetricValue(after, "orders_created_total")
	if !ok {
		t.Fatal("orders_created_total missing from /metrics")
	}
	if afterCreated <= beforeCreated {
		t.Fatalf("orders_created_total did not increase: before=%d after=%d", beforeCreated, afterCreated)
	}
}
