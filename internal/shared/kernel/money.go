package kernel

import (
	"errors"
	"fmt"
)

// Money is an immutable value object representing amount in kopecks (1/100 of ruble).
type Money struct {
	amount   int64  // stored in minor units (kopecks)
	currency string // ISO 4217, e.g. "RUB"
}

var ErrNegativeMoney = errors.New("money amount cannot be negative")
var ErrCurrencyMismatch = errors.New("currency mismatch")

func NewMoney(amount int64, currency string) (Money, error) {
	if amount < 0 {
		return Money{}, ErrNegativeMoney
	}
	return Money{amount: amount, currency: currency}, nil
}

func MustMoney(amount int64, currency string) Money {
	m, err := NewMoney(amount, currency)
	if err != nil {
		panic(err)
	}
	return m
}

func (m Money) Amount() int64    { return m.amount }
func (m Money) Currency() string { return m.currency }
func (m Money) IsZero() bool     { return m.amount == 0 }

func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}
	return Money{amount: m.amount + other.amount, currency: m.currency}, nil
}

func (m Money) Sub(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}
	result := m.amount - other.amount
	if result < 0 {
		return Money{}, ErrNegativeMoney
	}
	return Money{amount: result, currency: m.currency}, nil
}

func (m Money) Multiply(factor int64) Money {
	return Money{amount: m.amount * factor, currency: m.currency}
}

func (m Money) String() string {
	return fmt.Sprintf("%d.%02d %s", m.amount/100, m.amount%100, m.currency)
}
