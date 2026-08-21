package domain

import (
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"
)

// Money keeps all grant calculations in fixed-scale decimal arithmetic.
type Money struct {
	value decimal.Decimal
}

var ZeroMoney = Money{value: decimal.Zero}

func ParseMoney(raw string) (Money, error) {
	d, err := decimal.NewFromString(raw)
	if err != nil {
		return Money{}, fmt.Errorf("parse money %q: %w", raw, err)
	}
	if !d.Equal(d.Round(2)) {
		return Money{}, fmt.Errorf("money %q has more than two decimal places", raw)
	}
	return Money{value: d.Round(2)}, nil
}

func MustMoney(raw string) Money {
	m, err := ParseMoney(raw)
	if err != nil {
		panic(err)
	}
	return m
}

func MoneyFromDecimal(value decimal.Decimal) Money {
	return Money{value: value.Round(2)}
}

func (m Money) Decimal() decimal.Decimal {
	return m.value
}

func (m Money) String() string {
	return m.value.StringFixed(2)
}

func (m Money) IsNegative() bool {
	return m.value.IsNegative()
}

func (m Money) IsZero() bool {
	return m.value.IsZero()
}

func (m Money) Positive() bool {
	return m.value.GreaterThan(decimal.Zero)
}

func (m Money) Add(other Money) Money {
	return MoneyFromDecimal(m.value.Add(other.value))
}

func (m Money) Sub(other Money) Money {
	return MoneyFromDecimal(m.value.Sub(other.value))
}

func (m Money) Mul(rate decimal.Decimal) Money {
	return MoneyFromDecimal(m.value.Mul(rate))
}

func (m Money) Min(other Money) Money {
	if m.value.LessThanOrEqual(other.value) {
		return m
	}
	return other
}

func (m Money) Max(other Money) Money {
	if m.value.GreaterThanOrEqual(other.value) {
		return m
	}
	return other
}

func (m Money) LessThan(other Money) bool {
	return m.value.LessThan(other.value)
}

func (m Money) LessThanOrEqual(other Money) bool {
	return m.value.LessThanOrEqual(other.value)
}

func (m Money) GreaterThan(other Money) bool {
	return m.value.GreaterThan(other.value)
}

func (m Money) Equal(other Money) bool {
	return m.value.Equal(other.value)
}

func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *Money) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("money must be a decimal string: %w", err)
	}
	parsed, err := ParseMoney(raw)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}
