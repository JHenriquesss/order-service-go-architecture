package order

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"order-service-go/internal/middleware"
	"order-service-go/internal/money"
)

func newHandlerTestEnv(t *testing.T) (http.Handler, uuid.UUID, uuid.UUID) {
	t.Helper()
	customerID := uuid.New()
	productID := uuid.New()
	price, _ := money.ParseString("89.90")
	repo := NewInMemoryRepository()
	customers := &FakeCustomerLookup{Customers: map[uuid.UUID]Customer{
		customerID: {ID: customerID, Active: true},
	}}
	products := &FakeProductLookup{Products: map[uuid.UUID]Product{
		productID: {ID: productID, Price: price, Active: true},
	}}
	svc := NewService(repo, customers, products, &FakeProducer{}, nil)
	svc.now = func() time.Time { return time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC) }
	return NewHandler(svc).Routes(), customerID, productID
}

// authed wraps a request with an authenticated user id, the way the server's
// Authenticator middleware would.
func authed(req *http.Request) *http.Request {
	return req.WithContext(middleware.ContextWithUserID(req.Context(), uuid.New().String()))
}

func TestHandlerCreateReturns201(t *testing.T) {
	h, customerID, productID := newHandlerTestEnv(t)
	body, _ := json.Marshal(CreateOrderInput{
		CustomerID: customerID,
		Items:      []CreateOrderItemInput{{ProductID: productID, Quantity: 1}},
	})
	req := authed(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestHandlerGetAndList(t *testing.T) {
	h, customerID, productID := newHandlerTestEnv(t)
	body, _ := json.Marshal(CreateOrderInput{
		CustomerID: customerID,
		Items:      []CreateOrderItemInput{{ProductID: productID, Quantity: 1}},
	})
	createRec := httptest.NewRecorder()
	h.ServeHTTP(createRec, authed(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))))
	var created OrderOutput
	_ = json.Unmarshal(createRec.Body.Bytes(), &created)

	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, httptest.NewRequest(http.MethodGet, "/"+created.ID.String(), nil))
	if getRec.Code != http.StatusOK {
		t.Fatalf("Get expected 200, got %d", getRec.Code)
	}

	listRec := httptest.NewRecorder()
	h.ServeHTTP(listRec, httptest.NewRequest(http.MethodGet, "/", nil))
	if listRec.Code != http.StatusOK {
		t.Fatalf("List expected 200, got %d", listRec.Code)
	}
}

func TestHandlerParseFilterInvalidStatus(t *testing.T) {
	h, _, _ := newHandlerTestEnv(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?status=INVALID", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerParseFilterInvalidCustomerID(t *testing.T) {
	h, _, _ := newHandlerTestEnv(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/?customer_id=not-a-uuid", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerGetInvalidID(t *testing.T) {
	h, _, _ := newHandlerTestEnv(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/bad-id", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerCreateUnauthorizedWithoutIdentity(t *testing.T) {
	h, customerID, productID := newHandlerTestEnv(t)
	body, _ := json.Marshal(CreateOrderInput{
		CustomerID: customerID,
		Items:      []CreateOrderItemInput{{ProductID: productID, Quantity: 1}},
	})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandlerCreateInvalidJSON(t *testing.T) {
	h, _, _ := newHandlerTestEnv(t)
	req := authed(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("{"))))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
