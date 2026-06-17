package product

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"order-service-go/internal/money"
	"order-service-go/internal/pagination"
)

// newTestHandler returns the product routes backed by an in-memory repository.
// Auth is applied by the server, so these tests exercise the handler directly.
func newTestHandler() http.Handler {
	return NewHandler(NewService(NewInMemoryRepository())).Routes()
}

func do(t *testing.T, srv http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != "" {
		buf.WriteString(body)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, target, &buf)
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

func createProduct(t *testing.T, srv http.Handler, name, sku, price string) ProductOutput {
	t.Helper()
	body := `{"name":"` + name + `","sku":"` + sku + `","price":` + price + `}`
	rec := do(t, srv, http.MethodPost, "/", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %q: expected 201, got %d (%s)", name, rec.Code, rec.Body.String())
	}
	var out ProductOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode created product: %v", err)
	}
	return out
}

func TestHandlerCreateAndGet(t *testing.T) {
	srv := newTestHandler()
	created := createProduct(t, srv, "Wireless Mouse", "MOUSE-001", "89.90")
	if created.Price.Cents() != 8990 {
		t.Fatalf("price not preserved: got %d cents", created.Price.Cents())
	}
	rec := do(t, srv, http.MethodGet, "/"+created.ID.String(), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", rec.Code)
	}
	var got ProductOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != created.ID || got.Price != money.FromCents(8990) {
		t.Fatalf("unexpected product: %+v", got)
	}
}

func TestHandlerCreateMissingNameReturnsValidation(t *testing.T) {
	srv := newTestHandler()
	rec := do(t, srv, http.MethodPost, "/", `{"sku":"X","price":10}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestHandlerCreateNonPositivePriceReturns400(t *testing.T) {
	srv := newTestHandler()
	rec := do(t, srv, http.MethodPost, "/", `{"name":"X","sku":"X1","price":0}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestHandlerCreateDuplicateSKUReturns409(t *testing.T) {
	srv := newTestHandler()
	createProduct(t, srv, "A", "DUP-SKU", "10.00")
	rec := do(t, srv, http.MethodPost, "/", `{"name":"B","sku":"DUP-SKU","price":12.00}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "DUPLICATE_RESOURCE" {
		t.Fatalf("expected DUPLICATE_RESOURCE, got %s", code)
	}
}

func TestHandlerGetUnknownReturns404(t *testing.T) {
	srv := newTestHandler()
	rec := do(t, srv, http.MethodGet, "/"+uuid.New().String(), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "RESOURCE_NOT_FOUND" {
		t.Fatalf("expected RESOURCE_NOT_FOUND, got %s", code)
	}
}

func TestHandlerInvalidIDReturns400(t *testing.T) {
	srv := newTestHandler()
	rec := do(t, srv, http.MethodGet, "/not-a-uuid", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerUpdateChangesFields(t *testing.T) {
	srv := newTestHandler()
	created := createProduct(t, srv, "Old", "UPD-SKU", "10.00")
	rec := do(t, srv, http.MethodPut, "/"+created.ID.String(),
		`{"name":"New","sku":"UPD-SKU","price":25.50}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var got ProductOutput
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Name != "New" || got.Price.Cents() != 2550 {
		t.Fatalf("fields not updated: %+v", got)
	}
}

func TestHandlerDeactivateThenStillRetrievable(t *testing.T) {
	srv := newTestHandler()
	created := createProduct(t, srv, "ACME", "DEACT-SKU", "10.00")
	rec := do(t, srv, http.MethodPatch, "/"+created.ID.String()+"/deactivate", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("deactivate: expected 204, got %d", rec.Code)
	}
	rec = do(t, srv, http.MethodGet, "/"+created.ID.String(), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get after deactivate: expected 200, got %d", rec.Code)
	}
	var got ProductOutput
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Active {
		t.Fatal("expected active=false after deactivate")
	}
}

func TestHandlerListFiltersAndPaginates(t *testing.T) {
	srv := newTestHandler()
	createProduct(t, srv, "Wireless Mouse", "MOUSE-1", "10.00")
	createProduct(t, srv, "Keyboard", "KEY-1", "20.00")
	createProduct(t, srv, "Wireless Pad", "PAD-1", "5.00")

	rec := do(t, srv, http.MethodGet, "/?name=wireless&page=1&page_size=1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec.Code)
	}
	var page pagination.Page[ProductOutput]
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if page.Total != 2 || page.TotalPages != 2 || len(page.Items) != 1 {
		t.Fatalf("filter+paginate wrong: total=%d totalPages=%d items=%d",
			page.Total, page.TotalPages, len(page.Items))
	}
}

func TestHandlerListInvalidPageSizeReturns400(t *testing.T) {
	srv := newTestHandler()
	rec := do(t, srv, http.MethodGet, "/?page_size=99999", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerListNonNumericPageReturns400(t *testing.T) {
	srv := newTestHandler()
	rec := do(t, srv, http.MethodGet, "/?page=abc", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
