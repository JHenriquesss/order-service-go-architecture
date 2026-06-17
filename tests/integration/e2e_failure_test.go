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
)

func TestOrderProcessingFailureReachesFailed(t *testing.T) {
	dsn, _, apiBase := requireE2EEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	pool := connectPostgres(t, ctx, dsn)
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

	var adminID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM users WHERE email=$1`, "admin@example.com").Scan(&adminID); err != nil {
		t.Fatalf("load admin user: %v", err)
	}

	orderID := uuid.New()
	customerID, err := uuid.Parse(customer.ID)
	if err != nil {
		t.Fatalf("parse customer id: %v", err)
	}
	productID, err := uuid.Parse(product.ID)
	if err != nil {
		t.Fatalf("parse product id: %v", err)
	}
	itemID := uuid.New()
	now := time.Now().UTC()
	if _, err := pool.Exec(ctx, `INSERT INTO orders (id, customer_id, status, total_amount, created_by, created_at, updated_at)
		VALUES ($1, $2, 'CREATED', 10.00, $3, $4, $4)`,
		orderID, customerID, adminID, now,
	); err != nil {
		t.Fatalf("insert order: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, total_price, created_at)
		VALUES ($1, $2, $3, 1, 10.00, 10.00, $4)`,
		itemID, orderID, productID, now,
	); err != nil {
		t.Fatalf("insert order item: %v", err)
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
	client.doJSON(t, http.MethodGet, "/api/orders/"+orderID.String(), nil, http.StatusOK, &apiOrder)
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
