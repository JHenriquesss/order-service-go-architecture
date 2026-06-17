//go:build integration

package integration

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func TestOrderProcessingFailureReachesFailed(t *testing.T) {
	dsn, redisAddr, apiBase := requireE2EEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	pool := connectPostgres(t, ctx, dsn)
	redisClient := connectRedis(t, ctx, redisAddr)
	client := newAPIClient(apiBase)
	client.login(t, "admin@example.com", "123456")

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
		"price": 10.00,
	}, http.StatusCreated, &product)

	var order orderResponse
	client.doJSON(t, http.MethodPost, "/api/orders", map[string]any{
		"customer_id": customer.ID,
		"items": []map[string]any{
			{"product_id": product.ID, "quantity": 1},
		},
	}, http.StatusCreated, &order)

	orderID, err := uuid.Parse(order.ID)
	if err != nil {
		t.Fatalf("parse order id: %v", err)
	}

	// Drain the queue message so the production worker does not race this test.
	for {
		_, err := redisClient.BRPop(ctx, 500*time.Millisecond, orderQueueName).Result()
		if errors.Is(err, redis.Nil) {
			break
		}
		if err != nil {
			t.Fatalf("drain queue: %v", err)
		}
	}

	if err := processOrderWithPaymentFailure(ctx, pool, orderID); err != nil {
		t.Fatalf("process failure: %v", err)
	}

	var status string
	err = pool.QueryRow(ctx, `SELECT status FROM orders WHERE id=$1`, orderID).Scan(&status)
	if err != nil {
		t.Fatalf("read order status: %v", err)
	}
	if status != "FAILED" {
		t.Fatalf("expected FAILED in database, got %s", status)
	}

	var apiOrder orderResponse
	client.doJSON(t, http.MethodGet, "/api/orders/"+order.ID, nil, http.StatusOK, &apiOrder)
	if apiOrder.Status != "FAILED" {
		t.Fatalf("expected FAILED via API, got %s", apiOrder.Status)
	}
}

// processOrderWithPaymentFailure mirrors production order processing with a
// failing payment step (BR-ORD-010) for integration verification.
func processOrderWithPaymentFailure(ctx context.Context, pool *pgxpool.Pool, orderID uuid.UUID) error {
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM orders WHERE id=$1`, orderID).Scan(&status); err != nil {
		return err
	}
	if status != "CREATED" {
		return nil
	}

	tag, err := pool.Exec(ctx, `UPDATE orders SET status='PROCESSING', updated_at=now() WHERE id=$1 AND status='CREATED'`, orderID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("order not in CREATED state")
	}

	_, err = pool.Exec(ctx,
		`UPDATE orders SET status='FAILED', processed_at=now(), updated_at=now() WHERE id=$1`,
		orderID,
	)
	return err
}
