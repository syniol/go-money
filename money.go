// Package money provides an immutable monetary value type backed by int64
// minor units and ISO 4217 currency metadata.
//
// The library is designed for financial systems that cannot tolerate
// floating-point rounding, silent overflow, or currency mix-ups:
//
//   - Amounts are stored as int64 minor units (cents, pence, sen).
//   - Every arithmetic operation is guarded against overflow.
//   - Every arithmetic operation refuses to combine mismatched currencies.
//   - Money values are immutable and safe for concurrent use.
//
// Prefer New for integer inputs and NewFromString for user or wire input.
// FromDecimal is available for float64 sources but should be avoided in
// hot financial paths because float64 cannot represent every decimal exactly.
package money

//go:generate go run cmd/gen_currencies/main.go

// Money is an immutable monetary amount in a specific currency.
//
// Amounts are stored in the currency's minor units so all arithmetic stays
// in exact int64. Money values are safe to pass by value and to share across
// goroutines.
type Money struct {
	amount   int64
	currency *Currency
}

// New builds a Money value from an amount already expressed in the currency's
// minor units.
func New(minorAmount int64, currencyCode string) (Money, error) {
	if currencyCode == "" {
		return Money{}, &MoneyError{Op: "New", Err: ErrInvalidCurrency}
	}
	cfg, ok := currencyConfig[currencyCode]
	if !ok {
		return Money{}, &MoneyError{Op: "New", Currency: currencyCode, Err: ErrInvalidCurrency}
	}
	return Money{amount: minorAmount, currency: cfg}, nil
}

// MustNew is the panicking equivalent of New. Use it only for package-level
// constants where an invalid input represents a programmer error.
func MustNew(minorAmount int64, currencyCode string) Money {
	m, err := New(minorAmount, currencyCode)
	if err != nil {
		// The underlying MoneyError already carries the "money.New(...)"
		// prefix; do not add a second "money.MustNew:" on top.
		panic(err)
	}
	return m
}

// Minor returns the raw amount in the currency's minor unit.
func (m Money) Minor() int64 { return m.amount }

// Currency returns the ISO 4217 code, or the empty string for a zero-value Money.
func (m Money) Currency() string {
	if m.currency == nil {
		return ""
	}
	return m.currency.ISOCode
}

// Valid reports whether m carries a currency. A zero-value Money returned by
// a failing constructor is not valid and every method that requires a
// currency will refuse it. Prefer Valid over comparing Currency() to "".
func (m Money) Valid() bool { return m.currency != nil }

// IsZero reports whether the amount is exactly zero. Returns false for a
// zero-value Money that has no currency, because "zero" without a currency
// is not a valid amount.
func (m Money) IsZero() bool { return m.currency != nil && m.amount == 0 }

// IsPositive reports whether the amount is strictly greater than zero.
// Returns false for a zero-value Money without a currency.
func (m Money) IsPositive() bool { return m.currency != nil && m.amount > 0 }

// IsNegative reports whether the amount is strictly less than zero.
// Returns false for a zero-value Money without a currency.
func (m Money) IsNegative() bool { return m.currency != nil && m.amount < 0 }

// assertSameCurrency compares by ISOCode rather than pointer identity so that
// a test double or a caller who legitimately constructs a *Currency outside
// the package-level currencyConfig map is not treated as a mismatch. Pointer
// identity would be faster but depends on developer discipline that the
// exported Currency type cannot enforce.
func (m Money) assertSameCurrency(other Money) error {
	if m.currency == nil || other.currency == nil {
		return ErrCurrencyMismatch
	}
	if m.currency.ISOCode != other.currency.ISOCode {
		return ErrCurrencyMismatch
	}
	return nil
}
