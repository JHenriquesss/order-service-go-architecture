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

func TestParseStringValidForms(t *testing.T) {
	cases := map[string]int64{
		"0":       0,
		"5":       500,
		"5.5":     550,
		"5.05":    505,
		"-2.50":   -250,
		"1234.56": 123456,
	}
	for in, want := range cases {
		m, err := ParseString(in)
		if err != nil {
			t.Fatalf("ParseString(%q): unexpected error %v", in, err)
		}
		if m.Cents() != want {
			t.Fatalf("ParseString(%q) = %d cents, want %d", in, m.Cents(), want)
		}
	}
}

func TestParseStringRejectsInvalid(t *testing.T) {
	for _, in := range []string{"", "   ", "-", "abc", "1.2.3", "1.", "1.ab", "x.5"} {
		if _, err := ParseString(in); err == nil {
			t.Fatalf("ParseString(%q): expected error, got nil", in)
		}
	}
}

func TestMoneyMarshalIntegerAndNegative(t *testing.T) {
	if b, _ := FromCents(500).MarshalJSON(); string(b) != "5" {
		t.Fatalf("integer amount: got %s, want 5", b)
	}
	if b, _ := FromCents(-250).MarshalJSON(); string(b) != "-2.50" {
		t.Fatalf("negative amount: got %s, want -2.50", b)
	}
}

func TestMoneyUnmarshalStringFormAndInvalid(t *testing.T) {
	var m Money
	if err := json.Unmarshal([]byte(`"12.34"`), &m); err != nil {
		t.Fatalf("unmarshal quoted: %v", err)
	}
	if m.Cents() != 1234 {
		t.Fatalf("got %d cents, want 1234", m.Cents())
	}
	if err := json.Unmarshal([]byte(`"nope"`), &m); err == nil {
		t.Fatal("expected error for invalid quoted amount")
	}
	if err := json.Unmarshal([]byte(`true`), &m); err == nil {
		t.Fatal("expected error for non-number/non-string")
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
