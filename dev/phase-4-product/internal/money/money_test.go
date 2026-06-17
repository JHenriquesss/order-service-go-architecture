package money

import (
	"encoding/json"
	"testing"
)

func TestParseStringPositiveAmount(t *testing.T) {
	m, err := ParseString("89.90")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Cents() != 8990 {
		t.Fatalf("expected 8990 cents, got %d", m.Cents())
	}
}

func TestParseStringRejectsMoreThanTwoDecimals(t *testing.T) {
	_, err := ParseString("1.999")
	if err == nil {
		t.Fatal("expected error for too many decimal places")
	}
}

func TestMoneyMarshalJSONWithoutFloat64(t *testing.T) {
	m := FromCents(8990)
	b, err := m.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "89.90" {
		t.Fatalf("expected 89.90, got %s", b)
	}
}

func TestMoneyUnmarshalJSONNumber(t *testing.T) {
	var m Money
	if err := json.Unmarshal([]byte(`89.90`), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Cents() != 8990 {
		t.Fatalf("expected 8990 cents, got %d", m.Cents())
	}
}

func TestMoneyIsPositive(t *testing.T) {
	if !FromCents(1).IsPositive() {
		t.Fatal("1 cent must be positive")
	}
	if FromCents(0).IsPositive() {
		t.Fatal("zero must not be positive")
	}
	if FromCents(-100).IsPositive() {
		t.Fatal("negative must not be positive")
	}
}
