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

	"order-service-product/internal/auth"
	"order-service-product/internal/money"
	"order-service-product/internal/product"
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

func newTestServer() http.Handler {
	repo := product.NewInMemoryRepository()
	svc := product.NewService(repo)
	handler := product.NewHandler(svc)
	return New(fakeVerifier{}, handler)
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

func createProduct(t *testing.T, srv http.Handler, name, sku string, price money.Money) product.ProductOutput {
	t.Helper()
	rec := do(t, srv, http.MethodPost, "/api/products", "admin-token", product.CreateProductInput{
		Name: name, SKU: sku, Price: price,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %q: expected 201, got %d (%s)", name, rec.Code, rec.Body.String())
	}
	var out product.ProductOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode created product: %v", err)
	}
	return out
}

func price8990(t *testing.T) money.Money {
	t.Helper()
	p, err := money.ParseString("89.90")
	if err != nil {
		t.Fatalf("parse price: %v", err)
	}
	return p
}

func TestUnauthenticatedReturns401(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, http.MethodGet, "/api/products", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED, got %s", code)
	}
}

func TestInvalidTokenReturns401(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, http.MethodGet, "/api/products", "garbage", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestOperatorCanCreate(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, http.MethodPost, "/api/products", "operator-token", product.CreateProductInput{
		Name: "Mouse", SKU: "SKU-OP", Price: price8990(t),
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("operator create: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestCreateAndGet(t *testing.T) {
	srv := newTestServer()
	created := createProduct(t, srv, "Wireless Mouse", "MOUSE-001", price8990(t))
	rec := do(t, srv, http.MethodGet, "/api/products/"+created.ID.String(), "admin-token", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", rec.Code)
	}
	var got product.ProductOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected id %s, got %s", created.ID, got.ID)
	}
}

func TestCreateMissingNameReturnsValidation(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, http.MethodPost, "/api/products", "admin-token", product.CreateProductInput{
		SKU: "SKU-X", Price: price8990(t),
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestCreateNonPositivePriceReturnsValidation(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, http.MethodPost, "/api/products", "admin-token", product.CreateProductInput{
		Name: "Mouse", SKU: "SKU-ZERO", Price: money.FromCents(0),
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestCreateDuplicateReturns409(t *testing.T) {
	srv := newTestServer()
	createProduct(t, srv, "Mouse A", "SKU-DUP", price8990(t))
	rec := do(t, srv, http.MethodPost, "/api/products", "admin-token", product.CreateProductInput{
		Name: "Mouse B", SKU: "SKU-DUP", Price: price8990(t),
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "DUPLICATE_RESOURCE" {
		t.Fatalf("expected DUPLICATE_RESOURCE, got %s", code)
	}
}

func TestGetUnknownReturns404(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, http.MethodGet, "/api/products/"+uuid.New().String(), "admin-token", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "RESOURCE_NOT_FOUND" {
		t.Fatalf("expected RESOURCE_NOT_FOUND, got %s", code)
	}
}

func TestUpdateChangesFields(t *testing.T) {
	srv := newTestServer()
	created := createProduct(t, srv, "Old", "SKU-UPD", price8990(t))
	newPrice, _ := money.ParseString("99.99")
	rec := do(t, srv, http.MethodPut, "/api/products/"+created.ID.String(), "admin-token",
		product.UpdateProductInput{Name: "New", SKU: "SKU-UPD", Price: newPrice})
	if rec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var got product.ProductOutput
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Name != "New" || got.Price.Cents() != 9999 {
		t.Fatalf("fields not updated: %+v", got)
	}
}

func TestDeactivateThenStillRetrievable(t *testing.T) {
	srv := newTestServer()
	created := createProduct(t, srv, "Mouse", "SKU-DEACT", price8990(t))
	rec := do(t, srv, http.MethodPatch, "/api/products/"+created.ID.String()+"/deactivate", "admin-token", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("deactivate: expected 204, got %d", rec.Code)
	}
	rec = do(t, srv, http.MethodGet, "/api/products/"+created.ID.String(), "admin-token", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get after deactivate: expected 200, got %d", rec.Code)
	}
	var got product.ProductOutput
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Active {
		t.Fatal("expected active=false after deactivate")
	}
}

func TestListFiltersAndPaginates(t *testing.T) {
	srv := newTestServer()
	createProduct(t, srv, "Wireless Mouse", "MOUSE-1", price8990(t))
	createProduct(t, srv, "Keyboard", "KEY-2", price8990(t))
	createProduct(t, srv, "Wireless Headset", "HEAD-3", price8990(t))

	rec := do(t, srv, http.MethodGet, "/api/products?name=wireless&page=1&page_size=1", "admin-token", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec.Code)
	}
	var page product.Page[product.ProductOutput]
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if page.Total != 2 || page.TotalPages != 2 || len(page.Items) != 1 {
		t.Fatalf("filter+paginate wrong: total=%d totalPages=%d items=%d",
			page.Total, page.TotalPages, len(page.Items))
	}
}

func TestListInvalidPageSizeReturns400(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, http.MethodGet, "/api/products?page_size=99999", "admin-token", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestListNonNumericPageReturns400(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, http.MethodGet, "/api/products?page=abc", "admin-token", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
