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

func TestMultiplyAndAdd(t *testing.T) {
	unit, _ := ParseString("89.90")
	line1 := unit.Multiply(2)
	line2 := unit.Multiply(1)
	total := line1.Add(line2)
	if line1.Cents() != 17980 {
		t.Fatalf("line1: expected 17980, got %d", line1.Cents())
	}
	if line2.Cents() != 8990 {
		t.Fatalf("line2: expected 8990, got %d", line2.Cents())
	}
	if total.Cents() != 26970 {
		t.Fatalf("total: expected 26970, got %d", total.Cents())
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
