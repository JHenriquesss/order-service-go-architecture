package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"order-service-go/internal/auth"
	"order-service-go/internal/customer"
	"order-service-go/internal/order"
	"order-service-go/internal/product"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// newTestServer wires the full stack with an in-memory repository, the way main
// would, and returns the httptest server plus the token manager.
func newTestServer(t *testing.T) (*httptest.Server, *auth.TokenManager) {
	t.Helper()
	tm, err := auth.NewTokenManager("integration-secret", 120*time.Minute)
	if err != nil {
		t.Fatalf("NewTokenManager: %v", err)
	}
	svc := auth.NewService(auth.NewInMemoryUserRepository(), tm)
	customerHandler := customer.NewHandler(customer.NewService(customer.NewInMemoryRepository()))
	productHandler := product.NewHandler(product.NewService(product.NewInMemoryRepository()))
	orderSvc := order.NewService(
		order.NewInMemoryRepository(),
		&order.FakeCustomerLookup{Customers: nil},
		&order.FakeProductLookup{Products: nil},
		&order.FakeProducer{},
		nil,
	)
	orderHandler := order.NewHandler(orderSvc)
	srv := httptest.NewServer(New(discardLogger(), auth.NewHandler(svc), customerHandler, productHandler, orderHandler, tm))
	t.Cleanup(srv.Close)
	return srv, tm
}

func post(t *testing.T, url, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func TestHealthReturns200(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("health body not JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %q, want ok", body["status"])
	}
}

func TestRegisterThenLoginFlow(t *testing.T) {
	srv, _ := newTestServer(t)

	reg := post(t, srv.URL+"/api/auth/register",
		`{"name":"Admin","email":"admin@example.com","password":"123456","role":"ADMIN"}`)
	if reg.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d, want 201", reg.StatusCode)
	}
	regBody, _ := io.ReadAll(reg.Body)
	if strings.Contains(string(regBody), "password") || strings.Contains(string(regBody), "hash") {
		t.Fatalf("register response leaked password/hash: %s", regBody)
	}

	login := post(t, srv.URL+"/api/auth/login",
		`{"email":"admin@example.com","password":"123456"}`)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", login.StatusCode)
	}
	var tok auth.TokenResponse
	if err := json.NewDecoder(login.Body).Decode(&tok); err != nil {
		t.Fatalf("decode token: %v", err)
	}
	if tok.AccessToken == "" || tok.TokenType != "Bearer" || tok.ExpiresIn != 7200 {
		t.Fatalf("unexpected token response: %+v", tok)
	}
}

func TestLoginWrongPasswordIsGeneric401(t *testing.T) {
	srv, _ := newTestServer(t)
	post(t, srv.URL+"/api/auth/register",
		`{"name":"U","email":"u@example.com","password":"correctpass","role":"OPERATOR"}`)

	resp := post(t, srv.URL+"/api/auth/login", `{"email":"u@example.com","password":"wrong"}`)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(strings.ToLower(string(body)), "password") ||
		strings.Contains(strings.ToLower(string(body)), "email") {
		t.Fatalf("login error must be generic, leaked detail: %s", body)
	}
}

func protectedGet(t *testing.T, url, token string) int {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func TestProtectedRouteEnforcesAuthAndRole(t *testing.T) {
	srv, tm := newTestServer(t)
	adminTok, _, _ := tm.Issue("admin-id", auth.RoleAdmin)
	operTok, _, _ := tm.Issue("oper-id", auth.RoleOperator)

	if code := protectedGet(t, srv.URL+"/api/users", ""); code != http.StatusUnauthorized {
		t.Fatalf("no token: status = %d, want 401", code)
	}
	if code := protectedGet(t, srv.URL+"/api/users", "garbage.token.value"); code != http.StatusUnauthorized {
		t.Fatalf("bad token: status = %d, want 401", code)
	}
	if code := protectedGet(t, srv.URL+"/api/users", operTok); code != http.StatusForbidden {
		t.Fatalf("OPERATOR on ADMIN-only: status = %d, want 403", code)
	}
	if code := protectedGet(t, srv.URL+"/api/users", adminTok); code != http.StatusOK {
		t.Fatalf("ADMIN on ADMIN-only: status = %d, want 200", code)
	}
}

func TestCustomerRoutesEnforceAuthAndCreate(t *testing.T) {
	srv, tm := newTestServer(t)
	adminTok, _, _ := tm.Issue("admin-id", auth.RoleAdmin)

	// Unauthenticated create is rejected.
	if code := protectedGet(t, srv.URL+"/api/customers", ""); code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated list: status = %d, want 401", code)
	}

	// Authenticated create with a valid ADMIN token succeeds.
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/customers",
		strings.NewReader(`{"name":"ACME","document":"123","email":"a@acme.com"}`))
	req.Header.Set("Authorization", "Bearer "+adminTok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", resp.StatusCode)
	}
}

func TestRegisterRejectsBadJSON(t *testing.T) {
	srv, _ := newTestServer(t)
	resp, err := http.Post(srv.URL+"/api/auth/register", "application/json", bytes.NewReader([]byte("{not json")))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
