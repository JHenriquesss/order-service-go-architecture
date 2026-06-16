package errors

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func decode(t *testing.T, rec *httptest.ResponseRecorder) errorResponse {
	t.Helper()
	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v (%q)", err, rec.Body.String())
	}
	return body
}

func TestWriteRendersAppError(t *testing.T) {
	rec := httptest.NewRecorder()

	Write(rec, InvalidOrderStatus("Only PAID orders can be shipped."))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}
	body := decode(t, rec)
	if body.Error.Code != string(CodeInvalidOrderStatus) {
		t.Errorf("code = %q, want %q", body.Error.Code, CodeInvalidOrderStatus)
	}
	if body.Error.Message != "Only PAID orders can be shipped." {
		t.Errorf("message = %q", body.Error.Message)
	}
}

func TestWriteFallsBackForUnmappedError(t *testing.T) {
	rec := httptest.NewRecorder()

	Write(rec, errors.New("connection reset by peer: secret-host:5432"))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	body := decode(t, rec)
	if body.Error.Code != string(CodeInternal) {
		t.Errorf("code = %q, want %q", body.Error.Code, CodeInternal)
	}
	if body.Error.Message != "Internal server error" {
		t.Errorf("message leaked internals: %q", body.Error.Message)
	}
}

func TestConstructorsSetCodeAndStatus(t *testing.T) {
	cases := []struct {
		name   string
		err    *AppError
		code   Code
		status int
	}{
		{"validation", Validation("bad"), CodeValidation, http.StatusBadRequest},
		{"not found", NotFound("missing"), CodeNotFound, http.StatusNotFound},
		{"duplicate", Duplicate("conflict"), CodeDuplicate, http.StatusConflict},
		{"invalid status", InvalidOrderStatus("nope"), CodeInvalidOrderStatus, http.StatusBadRequest},
		{"internal", Internal("boom", nil), CodeInternal, http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.err.Code != c.code {
				t.Errorf("code = %q, want %q", c.err.Code, c.code)
			}
			if c.err.HTTPStatus != c.status {
				t.Errorf("status = %d, want %d", c.err.HTTPStatus, c.status)
			}
		})
	}
}

func TestErrorStringWithAndWithoutCause(t *testing.T) {
	noCause := Validation("bad request")
	if noCause.Error() != "VALIDATION_ERROR: bad request" {
		t.Errorf("unexpected Error(): %q", noCause.Error())
	}
	withCause := Internal("boom", errors.New("root"))
	if withCause.Error() != "INTERNAL_ERROR: boom: root" {
		t.Errorf("unexpected Error(): %q", withCause.Error())
	}
}

func TestAppErrorUnwrap(t *testing.T) {
	cause := errors.New("boom")
	err := Internal("Internal server error", cause)
	if !errors.Is(err, cause) {
		t.Error("expected wrapped cause to be unwrappable")
	}
}
