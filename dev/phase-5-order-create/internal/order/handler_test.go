package order

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"order-service-order/internal/auth"
	"order-service-order/internal/money"
)

func newHandlerTestEnv(t *testing.T) (*Handler, uuid.UUID, uuid.UUID) {
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
	return NewHandler(svc), customerID, productID
}

func TestHandlerCreateReturns201(t *testing.T) {
	h, customerID, productID := newHandlerTestEnv(t)
	body, _ := json.Marshal(CreateOrderInput{
		CustomerID: customerID,
		Items:      []CreateOrderItemInput{{ProductID: productID, Quantity: 1}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewReader(body))
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), auth.Identity{
		UserID: uuid.New(),
		Role:   auth.RoleAdmin,
	}))
	rec := httptest.NewRecorder()
	h.Create(rec, req)
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
	createReq := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewReader(body))
	createReq = createReq.WithContext(auth.ContextWithIdentity(createReq.Context(), auth.Identity{
		UserID: uuid.New(), Role: auth.RoleAdmin,
	}))
	createRec := httptest.NewRecorder()
	h.Create(createRec, createReq)

	var created OrderOutput
	_ = json.Unmarshal(createRec.Body.Bytes(), &created)

	getReq := httptest.NewRequest(http.MethodGet, "/api/orders/"+created.ID.String(), nil)
	getReq.SetPathValue("id", created.ID.String())
	getRec := httptest.NewRecorder()
	h.Get(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("Get expected 200, got %d", getRec.Code)
	}

	listRec := httptest.NewRecorder()
	h.List(listRec, httptest.NewRequest(http.MethodGet, "/api/orders", nil))
	if listRec.Code != http.StatusOK {
		t.Fatalf("List expected 200, got %d", listRec.Code)
	}
}

func TestHandlerParseFilterInvalidStatus(t *testing.T) {
	h, _, _ := newHandlerTestEnv(t)
	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/api/orders?status=INVALID", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerParseFilterInvalidCustomerID(t *testing.T) {
	h, _, _ := newHandlerTestEnv(t)
	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, "/api/orders?customer_id=not-a-uuid", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerGetInvalidID(t *testing.T) {
	h, _, _ := newHandlerTestEnv(t)
	req := httptest.NewRequest(http.MethodGet, "/api/orders/bad-id", nil)
	req.SetPathValue("id", "bad-id")
	rec := httptest.NewRecorder()
	h.Get(rec, req)
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
	h.Create(rec, httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewReader(body)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandlerCreateInvalidJSON(t *testing.T) {
	h, _, _ := newHandlerTestEnv(t)
	req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewReader([]byte("{")))
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), auth.Identity{
		UserID: uuid.New(), Role: auth.RoleAdmin,
	}))
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
