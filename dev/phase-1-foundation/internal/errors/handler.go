package errors

import (
	"encoding/json"
	"errors"
	"net/http"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

// Write renders err as the canonical JSON error response (architecture §24):
//
//	{ "error": { "code", "message" } }
//
// A non-AppError (or unmapped error) falls back to INTERNAL_ERROR / 500 with a
// generic message so internal details never leak to clients.
func Write(w http.ResponseWriter, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = Internal("Internal server error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HTTPStatus)
	_ = json.NewEncoder(w).Encode(errorResponse{
		Error: errorBody{Code: string(appErr.Code), Message: appErr.Message},
	})
}
