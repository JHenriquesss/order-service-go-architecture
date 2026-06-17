package order

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"order-service-worker/internal/auth"
	apperrors "order-service-worker/internal/errors"
)

func TestCreateValidatesCustomerID(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		Items: []CreateOrderItemInput{{ProductID: env.productA, Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeValidation)
}

func TestCreateCustomerNotFound(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: uuid.New(),
		Items:      []CreateOrderItemInput{{ProductID: env.productA, Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestCreateInactiveCustomer(t *testing.T) {
	env := newTestEnv(t)
	inactive := uuid.New()
	env.customers.Customers[inactive] = Customer{ID: inactive, Active: false}
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: inactive,
		Items:      []CreateOrderItemInput{{ProductID: env.productA, Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeInactiveCustomer)
}

func TestCreateInactiveProduct(t *testing.T) {
	env := newTestEnv(t)
	inactive := uuid.New()
	env.products.Products[inactive] = Product{ID: inactive, Price: price8990(t), Active: false}
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: env.customer,
		Items:      []CreateOrderItemInput{{ProductID: inactive, Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeInactiveProduct)
}

func TestCreateProductNotFound(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: env.customer,
		Items:      []CreateOrderItemInput{{ProductID: uuid.New(), Quantity: 1}},
	})
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestCreateValidatesQuantity(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Create(context.Background(), uuid.New(), CreateOrderInput{
		CustomerID: env.customer,
		Items:      []CreateOrderItemInput{{ProductID: env.productA, Quantity: 0}},
	})
	assertCode(t, err, apperrors.CodeValidation)
}

func TestCancelNotFound(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Cancel(context.Background(), uuid.New())
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestShipNotFound(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.Ship(context.Background(), uuid.New())
	assertCode(t, err, apperrors.CodeNotFound)
}

func TestListPageSizeValidation(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.service.List(context.Background(), OrderFilter{Page: 1, PageSize: 101})
	assertCode(t, err, apperrors.CodeValidation)
}

func TestHandlerGetOrder(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)
	h := NewHandler(env.service)
	rec := do(t, h, http.MethodGet, "/api/orders/"+orderID.String(), "admin-token")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandlerListWithStatusFilter(t *testing.T) {
	env := newTestEnv(t)
	_ = seedCreatedOrder(t, env)
	h := NewHandler(env.service)
	rec := do(t, h, http.MethodGet, "/api/orders?status=CREATED&page=1&page_size=10", "admin-token")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerCreateSuccess(t *testing.T) {
	env := newTestEnv(t)
	h := NewHandler(env.service)
	body := bytes.NewBufferString(`{"customer_id":"` + env.customer.String() + `","items":[{"product_id":"` + env.productA.String() + `","quantity":1}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/orders", body)
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), auth.Identity{UserID: uuid.New(), Role: auth.RoleAdmin}))
	mux := http.NewServeMux()
	h.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestHandlerInvalidOrderID(t *testing.T) {
	env := newTestEnv(t)
	h := NewHandler(env.service)
	rec := do(t, h, http.MethodGet, "/api/orders/not-a-uuid", "admin-token")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandlerListInvalidStatusFilter(t *testing.T) {
	env := newTestEnv(t)
	h := NewHandler(env.service)
	rec := do(t, h, http.MethodGet, "/api/orders?status=BOGUS", "admin-token")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}
