package order

import (
	"fmt"
	"strings"
)

// OrderStatus is the defined type for order lifecycle states (architecture §7).
type OrderStatus string

const (
	StatusCreated    OrderStatus = "CREATED"
	StatusProcessing OrderStatus = "PROCESSING"
	StatusPaid       OrderStatus = "PAID"
	StatusShipped    OrderStatus = "SHIPPED"
	StatusCanceled   OrderStatus = "CANCELED"
	StatusFailed     OrderStatus = "FAILED"
)

// allStatuses lists every valid status for exhaustive transition tests.
var allStatuses = []OrderStatus{
	StatusCreated,
	StatusProcessing,
	StatusPaid,
	StatusShipped,
	StatusCanceled,
	StatusFailed,
}

// ParseOrderStatus validates and returns an OrderStatus from a string.
func ParseOrderStatus(raw string) (OrderStatus, error) {
	s := OrderStatus(strings.TrimSpace(strings.ToUpper(raw)))
	switch s {
	case StatusCreated, StatusProcessing, StatusPaid, StatusShipped, StatusCanceled, StatusFailed:
		return s, nil
	default:
		return "", fmt.Errorf("invalid order status: %q", raw)
	}
}

// CanTransition reports whether moving from one status to another is allowed
// per the transition table (architecture §7).
func CanTransition(from, to OrderStatus) bool {
	switch from {
	case StatusCreated:
		return to == StatusProcessing || to == StatusCanceled
	case StatusProcessing:
		return to == StatusPaid || to == StatusFailed
	case StatusPaid:
		return to == StatusShipped
	default:
		return false
	}
}

// AllStatuses returns every defined status (for exhaustive tests).
func AllStatuses() []OrderStatus {
	out := make([]OrderStatus, len(allStatuses))
	copy(out, allStatuses)
	return out
}
