package customer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// newTestHandler returns the customer routes backed by an in-memory repository.
// Auth is applied by the server, so these tests exercise the handler directly.
func newTestHandler() http.Handler {
	return NewHandler(NewService(NewInMemoryRepository())).Routes()
}

func do(t *testing.T, srv http.Handler, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
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

func createCustomer(t *testing.T, srv http.Handler, name, document string) CustomerOutput {
	t.Helper()
	rec := do(t, srv, http.MethodPost, "/", CreateCustomerInput{Name: name, Document: document})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %q: expected 201, got %d (%s)", name, rec.Code, rec.Body.String())
	}
	var out CustomerOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode created customer: %v", err)
	}
	return out
}

func TestHandlerCreateAndGet(t *testing.T) {
	srv := newTestHandler()
	created := createCustomer(t, srv, "ACME Ltd", "12345678000199")
	rec := do(t, srv, http.MethodGet, "/"+created.ID.String(), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", rec.Code)
	}
	var got CustomerOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected id %s, got %s", created.ID, got.ID)
	}
}

func TestHandlerCreateMissingNameReturnsValidation(t *testing.T) {
	srv := newTestHandler()
	rec := do(t, srv, http.MethodPost, "/", CreateCustomerInput{Document: "DOC-X"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestHandlerCreateDuplicateReturns409(t *testing.T) {
	srv := newTestHandler()
	createCustomer(t, srv, "ACME", "DOC-DUP")
	rec := do(t, srv, http.MethodPost, "/", CreateCustomerInput{Name: "Other", Document: "DOC-DUP"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "DUPLICATE_RESOURCE" {
		t.Fatalf("expected DUPLICATE_RESOURCE, got %s", code)
	}
}

func TestHandlerGetUnknownReturns404(t *testing.T) {
	srv := newTestHandler()
	rec := do(t, srv, http.MethodGet, "/"+uuid.New().String(), nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "RESOURCE_NOT_FOUND" {
		t.Fatalf("expected RESOURCE_NOT_FOUND, got %s", code)
	}
}

func TestHandlerInvalidIDReturns400(t *testing.T) {
	srv := newTestHandler()
	rec := do(t, srv, http.MethodGet, "/not-a-uuid", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerUpdateChangesFields(t *testing.T) {
	srv := newTestHandler()
	created := createCustomer(t, srv, "Old", "DOC-UPD")
	rec := do(t, srv, http.MethodPut, "/"+created.ID.String(),
		UpdateCustomerInput{Name: "New", Document: "DOC-UPD", Email: "n@e.com"})
	if rec.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var got CustomerOutput
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Name != "New" || got.Email != "n@e.com" {
		t.Fatalf("fields not updated: %+v", got)
	}
}

func TestHandlerDeactivateThenStillRetrievable(t *testing.T) {
	srv := newTestHandler()
	created := createCustomer(t, srv, "ACME", "DOC-DEACT")
	rec := do(t, srv, http.MethodPatch, "/"+created.ID.String()+"/deactivate", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("deactivate: expected 204, got %d", rec.Code)
	}
	rec = do(t, srv, http.MethodGet, "/"+created.ID.String(), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get after deactivate: expected 200, got %d", rec.Code)
	}
	var got CustomerOutput
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Active {
		t.Fatal("expected active=false after deactivate")
	}
}

func TestHandlerListFiltersAndPaginates(t *testing.T) {
	srv := newTestHandler()
	createCustomer(t, srv, "ACME Ltd", "AAA-1")
	createCustomer(t, srv, "Globex", "BBB-2")
	createCustomer(t, srv, "ACME Corp", "AAA-3")

	rec := do(t, srv, http.MethodGet, "/?name=acme&page=1&page_size=1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec.Code)
	}
	var page Page[CustomerOutput]
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
	rec := do(t, srv, http.MethodGet, "/?page_size=99999", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if code := decodeErrorCode(t, rec); code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %s", code)
	}
}

func TestHandlerListNonNumericPageReturns400(t *testing.T) {
	srv := newTestHandler()
	rec := do(t, srv, http.MethodGet, "/?page=abc", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandlerListActiveFilterRejectsNonBool(t *testing.T) {
	srv := newTestHandler()
	rec := do(t, srv, http.MethodGet, "/?active=maybe", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
