// Package money provides a decimal-safe monetary type for numeric(15,2) values.
// Amounts are stored as scaled integer cents; float64 is never used.
package money

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const scale int64 = 100

// ErrInvalid indicates the input could not be parsed as money.
var ErrInvalid = errors.New("invalid money amount")

// Money represents a fixed-scale decimal amount with two fractional digits.
type Money struct {
	cents int64
}

// FromCents constructs Money from an integer number of cents.
func FromCents(cents int64) Money {
	return Money{cents: cents}
}

// Cents returns the stored amount in cents.
func (m Money) Cents() int64 {
	return m.cents
}

// IsPositive reports whether the amount is strictly greater than zero.
func (m Money) IsPositive() bool {
	return m.cents > 0
}

// ParseString parses a decimal string such as "89.90" without using float64.
func ParseString(raw string) (Money, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Money{}, ErrInvalid
	}

	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}
	if s == "" {
		return Money{}, ErrInvalid
	}

	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return Money{}, ErrInvalid
	}

	dollars, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Money{}, ErrInvalid
	}

	var frac int64
	if len(parts) == 2 {
		fracPart := parts[1]
		if fracPart == "" {
			return Money{}, ErrInvalid
		}
		if len(fracPart) > 2 {
			return Money{}, ErrInvalid
		}
		if len(fracPart) == 1 {
			fracPart += "0"
		}
		frac, err = strconv.ParseInt(fracPart, 10, 64)
		if err != nil {
			return Money{}, ErrInvalid
		}
	}

	cents := dollars*scale + frac
	if negative {
		cents = -cents
	}
	return Money{cents: cents}, nil
}

// ParseNumber parses a json.Number without converting through float64.
func ParseNumber(n json.Number) (Money, error) {
	return ParseString(n.String())
}

// MarshalJSON encodes the amount as a JSON number with up to two decimal places.
func (m Money) MarshalJSON() ([]byte, error) {
	sign := ""
	c := m.cents
	if c < 0 {
		sign = "-"
		c = -c
	}
	dollars := c / scale
	frac := c % scale
	if frac == 0 {
		return []byte(fmt.Sprintf("%s%d", sign, dollars)), nil
	}
	return []byte(fmt.Sprintf("%s%d.%02d", sign, dollars, frac)), nil
}

// UnmarshalJSON decodes a JSON number or quoted decimal string.
func (m *Money) UnmarshalJSON(b []byte) error {
	var asNumber json.Number
	if err := json.Unmarshal(b, &asNumber); err == nil {
		parsed, err := ParseNumber(asNumber)
		if err != nil {
			return err
		}
		*m = parsed
		return nil
	}
	var asString string
	if err := json.Unmarshal(b, &asString); err != nil {
		return ErrInvalid
	}
	parsed, err := ParseString(asString)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}
