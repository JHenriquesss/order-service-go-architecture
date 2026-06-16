// Package errors defines the application's error model: a typed AppError with
// the canonical error codes (architecture §24) and helper constructors.
package errors

import "net/http"

// Code is an application error code (architecture §24).
type Code string

const (
	CodeValidation         Code = "VALIDATION_ERROR"
	CodeUnauthorized       Code = "UNAUTHORIZED"
	CodeForbidden          Code = "FORBIDDEN"
	CodeNotFound           Code = "RESOURCE_NOT_FOUND"
	CodeDuplicate          Code = "DUPLICATE_RESOURCE"
	CodeInvalidOrderStatus Code = "INVALID_ORDER_STATUS"
	CodeInactiveCustomer   Code = "INACTIVE_CUSTOMER"
	CodeInactiveProduct    Code = "INACTIVE_PRODUCT"
	CodeInternal           Code = "INTERNAL_ERROR"
)

// AppError is a domain-aware error carrying an API code, a client-safe
// message, the HTTP status to respond with, and an optional wrapped cause.
type AppError struct {
	Code       Code
	Message    string
	HTTPStatus int
	cause      error
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return string(e.Code) + ": " + e.Message + ": " + e.cause.Error()
	}
	return string(e.Code) + ": " + e.Message
}

// Unwrap exposes the wrapped cause for errors.Is / errors.As.
func (e *AppError) Unwrap() error { return e.cause }

// New builds an AppError with an explicit code, message, status and cause.
func New(code Code, message string, status int, cause error) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: status, cause: cause}
}

// Validation reports an invalid request (400).
func Validation(message string) *AppError {
	return New(CodeValidation, message, http.StatusBadRequest, nil)
}

// NotFound reports a missing resource (404).
func NotFound(message string) *AppError {
	return New(CodeNotFound, message, http.StatusNotFound, nil)
}

// Duplicate reports a uniqueness conflict (409).
func Duplicate(message string) *AppError {
	return New(CodeDuplicate, message, http.StatusConflict, nil)
}

// InvalidOrderStatus reports an illegal order status transition (400).
func InvalidOrderStatus(message string) *AppError {
	return New(CodeInvalidOrderStatus, message, http.StatusBadRequest, nil)
}

// Internal reports an unexpected server error (500). The cause is wrapped for
// logging but never exposed to clients by the writer.
func Internal(message string, cause error) *AppError {
	return New(CodeInternal, message, http.StatusInternalServerError, cause)
}
