package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"order-service-worker/internal/auth"
	"order-service-worker/internal/metrics"
	"order-service-worker/internal/money"
	"order-service-worker/internal/order"
)

type fakeVerifier struct{}

func (fakeVerifier) Verify(token string) (auth.Identity, error) {
	switch token {
	case "admin-token":
		return auth.Identity{UserID: uuid.New(), Role: auth.RoleAdmin}, nil
	case "operator-token":
		return auth.Identity{UserID: uuid.New(), Role: auth.RoleOperator}, nil
	default:
		return auth.Identity{}, errors.New("invalid token")
	}
}

func newTestServer() (http.Handler, *order.InMemoryRepository, *order.FakeProducer, uuid.UUID, uuid.UUID, uuid.UUID) {
	customerID := uuid.New()
	productA := uuid.New()
	productB := uuid.New()
	price, _ := money.ParseString("89.90")

	repo := order.NewInMemoryRepository()
	customers := &order.FakeCustomerLookup{Customers: map[uuid.UUID]order.Customer{
		customerID: {ID: customerID, Active: true},
	}}
	products := &order.FakeProductLookup{Products: map[uuid.UUID]order.Product{
		productA: {ID: productA, Price: price, Active: true},
		productB: {ID: productB, Price: price, Active: true},
	}}
	producer := &order.FakeProducer{}
	collector := metrics.NewCollector()
	svc := order.NewService(repo, customers, products, producer, nil, collector, nil)
	handler := order.NewHandler(svc)
	return New(fakeVerifier{}, handler, metrics.NewHandler(collector)), repo, producer, customerID, productA, productB
}

func do(t *testing.T, srv http.Handler, method, target, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequestWithContext(context.Background(), method, target, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func decodeErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	return resp.Error.Code
}

func TestCreateOrderReturns201(t *testing.T) {
	srv, repo, producer, customerID, productA, productB := newTestServer()
	rec := do(t, srv, http.MethodPost, "/api/orders", "admin-token", order.CreateOrderInput{
		CustomerID: customerID,
		Items: []order.CreateOrderItemInput{
			{ProductID: productA, Quantity: 2},
			{ProductID: productB, Quantity: 1},
		},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	if repo.CommitCount != 1 {
		t.Fatalf("expected atomic commit, got %d", repo.CommitCount)
	}
	msg, ok := producer.LastMessage()
	if !ok || msg.Event != order.OrderCreatedEvent {
		t.Fatalf("expected queue message")
	}
}

func TestGetOrderReturns200(t *testing.T) {
	srv, _, _, customerID, productA, _ := newTestServer()
	create := do(t, srv, http.MethodPost, "/api/orders", "admin-token", order.CreateOrderInput{
		CustomerID: customerID,
		Items:      []order.CreateOrderItemInput{{ProductID: productA, Quantity: 1}},
	})
	var created order.OrderOutput
	_ = json.Unmarshal(create.Body.Bytes(), &created)

	rec := do(t, srv, http.MethodGet, "/api/orders/"+created.ID.String(), "admin-token", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestListOrdersReturns200(t *testing.T) {
	srv, _, _, customerID, productA, _ := newTestServer()
	_ = do(t, srv, http.MethodPost, "/api/orders", "admin-token", order.CreateOrderInput{
		CustomerID: customerID,
		Items:      []order.CreateOrderItemInput{{ProductID: productA, Quantity: 1}},
	})
	rec := do(t, srv, http.MethodGet, "/api/orders", "admin-token", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestCreateOrderEmptyItemsReturnsValidationError(t *testing.T) {
	srv, _, _, customerID, _, _ := newTestServer()
	rec := do(t, srv, http.MethodPost, "/api/orders", "admin-token", order.CreateOrderInput{
		CustomerID: customerID,
		Items:      []order.CreateOrderItemInput{},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestUnauthenticatedReturns401(t *testing.T) {
	srv, _, _, customerID, productA, _ := newTestServer()
	rec := do(t, srv, http.MethodPost, "/api/orders", "", order.CreateOrderInput{
		CustomerID: customerID,
		Items:      []order.CreateOrderItemInput{{ProductID: productA, Quantity: 1}},
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
