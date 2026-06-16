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

	"order-service-customer/internal/auth"
	"order-service-customer/internal/customer"
)

// fakeVerifier maps a known token to an identity; anything else fails. It lets
// the HTTP tests exercise auth without real JWT infrastructure.
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
	repo := customer.NewInMemoryRepository()
	svc := customer.NewService(repo)
	handler := customer.NewHandler(svc)
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

func createCustomer(t *testing.T, srv http.Handler, name, document string) customer.CustomerOutput {
	t.Helper()
	rec := do(t, srv, http.MethodPost, "/api/customers", "admin-token", customer.CreateCustomerInput{
		Name: name, Document: document,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %q: expected 201, got %d (%s)", name, rec.Code, rec.Body.String())
	}
	var out customer.CustomerOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode created customer: %v", err)
	}
	return out
}

func TestUnauthenticatedReturns401(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, http.MethodGet, "/api/customers", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED, got %s", code)
	}
}

func TestInvalidTokenReturns401(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, http.MethodGet, "/api/customers", "garbage", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestOperatorCanCreate(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, http.MethodPost, "/api/customers", "operator-token", customer.CreateCustomerInput{
		Name: "ACME", Document: "DOC-OP",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("operator create: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestCreateAndGet(t *testing.T) {
	srv := newTestServer()
	created := createCustomer(t, srv, "ACME Ltd", "12345678000199")
	rec := do(t, srv, http.MethodGet, "/api/customers/"+created.ID.String(), "admin-token", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", rec.Code)
	}
	var got customer.CustomerOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected id %s, got %s", created.ID, got.ID)
	}
}

func TestCreateMissingNameReturnsValidation(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, http.MethodPost, "/api/customers", "admin-token", customer.CreateCustomerInput{Document: "DOC-X"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestCreateDuplicateReturns409(t *testing.T) {
	srv := newTestServer()
	createCustomer(t, srv, "ACME", "DOC-DUP")
	rec := do(t, srv, http.MethodPost, "/api/customers", "admin-token", customer.CreateCustomerInput{
		Name: "Other", Document: "DOC-DUP",
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
	rec := do(t, srv, http.MethodGet, "/api/customers/"+uuid.New().String(), "admin-token", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "RESOURCE_NOT_FOUND" {
		t.Fatalf("expected RESOURCE_NOT_FOUND, got %s", code)
	}
}

func TestUpdateChangesFields(t *testing.T) {
	srv := newTestServer()
	created := createCustomer(t, srv, "Old", "DOC-UPD")
	rec := do(t, srv, http.MethodPut, "/api/customers/"+created.ID.String(), "admin-token",
		customer.UpdateCustomerInput{Name: "New", Document: "DOC-UPD", Email: "n@e.com"})
	if rec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var got customer.CustomerOutput
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Name != "New" || got.Email != "n@e.com" {
		t.Fatalf("fields not updated: %+v", got)
	}
}

func TestDeactivateThenStillRetrievable(t *testing.T) {
	srv := newTestServer()
	created := createCustomer(t, srv, "ACME", "DOC-DEACT")
	rec := do(t, srv, http.MethodPatch, "/api/customers/"+created.ID.String()+"/deactivate", "admin-token", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("deactivate: expected 204, got %d", rec.Code)
	}
	rec = do(t, srv, http.MethodGet, "/api/customers/"+created.ID.String(), "admin-token", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get after deactivate: expected 200, got %d", rec.Code)
	}
	var got customer.CustomerOutput
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Active {
		t.Fatal("expected active=false after deactivate")
	}
}

func TestListFiltersAndPaginates(t *testing.T) {
	srv := newTestServer()
	createCustomer(t, srv, "ACME Ltd", "AAA-1")
	createCustomer(t, srv, "Globex", "BBB-2")
	createCustomer(t, srv, "ACME Corp", "AAA-3")

	rec := do(t, srv, http.MethodGet, "/api/customers?name=acme&page=1&page_size=1", "admin-token", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec.Code)
	}
	var page customer.Page[customer.CustomerOutput]
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
	rec := do(t, srv, http.MethodGet, "/api/customers?page_size=99999", "admin-token", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestListNonNumericPageReturns400(t *testing.T) {
	srv := newTestServer()
	rec := do(t, srv, http.MethodGet, "/api/customers?page=abc", "admin-token", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
