package middleware

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func TestRequestIDGeneratesWhenAbsent(t *testing.T) {
	var captured string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = RequestIDFromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if captured == "" {
		t.Fatal("expected generated request id in context")
	}
	if got := rec.Header().Get(HeaderRequestID); got != captured {
		t.Errorf("response header %q != context id %q", got, captured)
	}
}

func TestRequestIDPropagatesInbound(t *testing.T) {
	var captured string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = RequestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderRequestID, "inbound-id")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if captured != "inbound-id" {
		t.Errorf("expected inbound-id propagated, got %q", captured)
	}
}

func TestRecoveryConvertsPanicTo500(t *testing.T) {
	h := Recovery(discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("recovery body not JSON: %v (%q)", err, rec.Body.String())
	}
	if body["error"]["code"] != "INTERNAL_ERROR" {
		t.Errorf("code = %q, want INTERNAL_ERROR", body["error"]["code"])
	}
}

func TestLoggingCapturesStatus(t *testing.T) {
	h := Logging(discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want 418", rec.Code)
	}
}
