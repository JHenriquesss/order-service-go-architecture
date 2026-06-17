package order

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
	apperrors "order-service-worker/internal/errors"
	"order-service-worker/internal/metrics"
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

func do(t *testing.T, h *Handler, method, target, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), method, target, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	mux := http.NewServeMux()
	h.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
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

func TestHandlerCancelCreatedOrder(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)
	h := NewHandler(env.service)

	rec := do(t, h, http.MethodPatch, "/api/orders/"+orderID.String()+"/cancel", "admin-token")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var out OrderOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Status != StatusCanceled {
		t.Fatalf("status %s", out.Status)
	}
}

func TestHandlerCancelPaidOrderReturnsInvalidStatus(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)
	_ = env.repo.UpdateStatus(context.Background(), orderID, StatusPaid)
	h := NewHandler(env.service)

	rec := do(t, h, http.MethodPatch, "/api/orders/"+orderID.String()+"/cancel", "admin-token")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
	if got := decodeErrorCode(t, rec); got != string(apperrors.CodeInvalidOrderStatus) {
		t.Fatalf("code %s", got)
	}
}

func TestHandlerShipPaidOrder(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)
	_ = env.repo.UpdateStatus(context.Background(), orderID, StatusPaid)
	h := NewHandler(env.service)

	rec := do(t, h, http.MethodPatch, "/api/orders/"+orderID.String()+"/ship", "operator-token")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var out OrderOutput
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Status != StatusShipped {
		t.Fatalf("status %s", out.Status)
	}
}

func TestHandlerShipCreatedOrderReturnsInvalidStatus(t *testing.T) {
	env := newTestEnv(t)
	orderID := seedCreatedOrder(t, env)
	h := NewHandler(env.service)

	rec := do(t, h, http.MethodPatch, "/api/orders/"+orderID.String()+"/ship", "admin-token")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
	if got := decodeErrorCode(t, rec); got != string(apperrors.CodeInvalidOrderStatus) {
		t.Fatalf("code %s", got)
	}
}

func TestHandlerCreateRequiresAuth(t *testing.T) {
	env := newTestEnv(t)
	h := NewHandler(env.service)
	body := bytes.NewBufferString(`{"customer_id":"` + env.customer.String() + `","items":[{"product_id":"` + env.productA.String() + `","quantity":1}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/orders", body)
	mux := http.NewServeMux()
	h.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestHandlerMetricsNotInOrderHandler(t *testing.T) {
	c := metrics.NewCollector()
	h := NewHandler(NewService(NewInMemoryRepository(), nil, nil, nil, nil, c, nil))
	mux := http.NewServeMux()
	h.Register(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
}
