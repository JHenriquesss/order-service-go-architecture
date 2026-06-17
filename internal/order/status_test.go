package order

import (
	"testing"
)

func TestParseOrderStatusValid(t *testing.T) {
	cases := []string{"CREATED", "created", " PROCESSING ", "paid", "SHIPPED", "CANCELED", "FAILED"}
	for _, raw := range cases {
		if _, err := ParseOrderStatus(raw); err != nil {
			t.Fatalf("ParseOrderStatus(%q) unexpected error: %v", raw, err)
		}
	}
}

func TestParseOrderStatusInvalid(t *testing.T) {
	_, err := ParseOrderStatus("UNKNOWN")
	if err == nil {
		t.Fatal("expected error for unknown status")
	}
}

func validTransitions() []struct{ from, to OrderStatus } {
	return []struct{ from, to OrderStatus }{
		{StatusCreated, StatusProcessing},
		{StatusCreated, StatusCanceled},
		{StatusProcessing, StatusPaid},
		{StatusProcessing, StatusFailed},
		{StatusPaid, StatusShipped},
	}
}

// CanTransition allows every valid Â§7 transition.
func TestCanTransitionAllowsValidTransitions(t *testing.T) {
	for _, tc := range validTransitions() {
		if !CanTransition(tc.from, tc.to) {
			t.Fatalf("expected %s -> %s to be allowed", tc.from, tc.to)
		}
	}
}

// CanTransition rejects every invalid transition.
func TestCanTransitionRejectsInvalidTransitions(t *testing.T) {
	valid := make(map[OrderStatus]map[OrderStatus]bool)
	for _, tc := range validTransitions() {
		if valid[tc.from] == nil {
			valid[tc.from] = make(map[OrderStatus]bool)
		}
		valid[tc.from][tc.to] = true
	}

	for _, from := range AllStatuses() {
		for _, to := range AllStatuses() {
			if from == to {
				continue
			}
			if valid[from][to] {
				continue
			}
			if CanTransition(from, to) {
				t.Fatalf("expected %s -> %s to be rejected", from, to)
			}
		}
	}
}

func TestCanTransitionSameStatusRejected(t *testing.T) {
	for _, s := range AllStatuses() {
		if CanTransition(s, s) {
			t.Fatalf("self-transition %s should be rejected", s)
		}
	}
}
