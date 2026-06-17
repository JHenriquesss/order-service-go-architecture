//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"order-service-go/internal/queue"
)

func TestTransientFailureReachesDeadLetterQueue(t *testing.T) {
	_, redisAddr, apiBase := requireE2EEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := connectRedis(t, ctx, redisAddr)
	_ = client.Del(ctx, queue.OrderDeadLetterQueueName).Err()

	api := newAPIClient(apiBase)
	api.login(t, "admin@example.com", "123456")

	suffix := uniqueSuffix()
	var customer customerResponse
	api.doJSON(t, http.MethodPost, "/api/customers", map[string]any{
		"name":     "DLQ Customer " + suffix,
		"document": "dlq-" + suffix,
		"email":    "dlq-" + suffix + "@example.com",
		"phone":    "+55 11 90000-0002",
	}, http.StatusCreated, &customer)

	var product productResponse
	api.doJSON(t, http.MethodPost, "/api/products", map[string]any{
		"name":  "DLQ Product " + suffix,
		"sku":   "DLQ-SKU-" + suffix,
		"price": 42.42,
	}, http.StatusCreated, &product)

	var created orderResponse
	api.doJSON(t, http.MethodPost, "/api/orders", map[string]any{
		"customer_id": customer.ID,
		"items": []map[string]any{
			{"product_id": product.ID, "quantity": 1},
		},
	}, http.StatusCreated, &created)

	deadline := time.Now().Add(45 * time.Second)
	var payloads []string
	for time.Now().Before(deadline) {
		raw, err := client.LRange(ctx, queue.OrderDeadLetterQueueName, 0, -1).Result()
		if err != nil {
			t.Fatalf("lrange dead-letter: %v", err)
		}
		for _, p := range raw {
			var msg struct {
				OrderID    string `json:"order_id"`
				RetryCount int    `json:"retry_count"`
			}
			if err := json.Unmarshal([]byte(p), &msg); err != nil {
				t.Fatalf("unmarshal dead-letter payload: %v", err)
			}
			if msg.OrderID == created.ID {
				payloads = append(payloads, p)
				if msg.RetryCount == 3 {
					break
				}
			}
		}
		if len(payloads) > 0 {
			var msg struct {
				RetryCount int `json:"retry_count"`
			}
			_ = json.Unmarshal([]byte(payloads[len(payloads)-1]), &msg)
			if msg.RetryCount == 3 {
				break
			}
		}
		time.Sleep(300 * time.Millisecond)
	}

	if len(payloads) == 0 {
		raw, _ := client.LRange(ctx, queue.OrderDeadLetterQueueName, 0, -1).Result()
		t.Fatalf("order %s not found in dead-letter queue; LRANGE orders:dead-letter 0 -1 = %v", created.ID, raw)
	}

	var final struct {
		OrderID    string `json:"order_id"`
		RetryCount int    `json:"retry_count"`
	}
	if err := json.Unmarshal([]byte(payloads[len(payloads)-1]), &final); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if final.RetryCount != 3 {
		t.Fatalf("dead-letter retry_count=%d, want 3", final.RetryCount)
	}

	still := waitForOrderStatus(t, api, created.ID, "CREATED", 5*time.Second)
	if still.Status != "CREATED" {
		t.Fatalf("transient-failure order should remain CREATED, got %s", still.Status)
	}
}

func TestPaymentDeclineAbsentFromDeadLetterQueue(t *testing.T) {
	_, redisAddr, apiBase := requireE2EEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	client := connectRedis(t, ctx, redisAddr)
	api := newAPIClient(apiBase)
	api.login(t, "admin@example.com", "123456")

	suffix := uniqueSuffix()
	var customer customerResponse
	api.doJSON(t, http.MethodPost, "/api/customers", map[string]any{
		"name":     "Decline DLQ " + suffix,
		"document": "decline-dlq-" + suffix,
		"email":    "decline-dlq-" + suffix + "@example.com",
		"phone":    "+55 11 90000-0003",
	}, http.StatusCreated, &customer)

	var product productResponse
	api.doJSON(t, http.MethodPost, "/api/products", map[string]any{
		"name":  "Decline Product " + suffix,
		"sku":   "DECLINE-SKU-" + suffix,
		"price": 13.37,
	}, http.StatusCreated, &product)

	var created orderResponse
	api.doJSON(t, http.MethodPost, "/api/orders", map[string]any{
		"customer_id": customer.ID,
		"items": []map[string]any{
			{"product_id": product.ID, "quantity": 1},
		},
	}, http.StatusCreated, &created)

	final := waitForOrderStatus(t, api, created.ID, "FAILED", 30*time.Second)
	if final.Status != "FAILED" {
		t.Fatalf("expected FAILED, got %s", final.Status)
	}

	raw, err := client.LRange(ctx, queue.OrderDeadLetterQueueName, 0, -1).Result()
	if err != nil {
		t.Fatalf("lrange dead-letter: %v", err)
	}
	for _, p := range raw {
		var msg struct {
			OrderID string `json:"order_id"`
		}
		if err := json.Unmarshal([]byte(p), &msg); err != nil {
			continue
		}
		if msg.OrderID == created.ID {
			t.Fatalf("payment-declined order %s must not appear in dead-letter queue", created.ID)
		}
	}
}
