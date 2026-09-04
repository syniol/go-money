package money

import "math"

// Add returns the sum of two Money values of the same currency. It refuses
// on currency mismatch or on int64 overflow. On error the receiver's
// currency is preserved in the returned Money so chained calls fail on the
// original error rather than on a spurious ErrCurrencyMismatch.
func (m Money) Add(other Money) (Money, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return Money{amount: 0, currency: m.currency}, &MoneyError{Op: "Add", Err: err}
	}
	if other.amount > 0 && m.amount > math.MaxInt64-other.amount {
		return Money{amount: 0, currency: m.currency}, &MoneyError{Op: "Add", Err: ErrOverflow}
	}
	if other.amount < 0 && m.amount < math.MinInt64-other.amount {
		return Money{amount: 0, currency: m.currency}, &MoneyError{Op: "Add", Err: ErrOverflow}
	}
	return Money{amount: m.amount + other.amount, currency: m.currency}, nil
}

// Sub returns the difference of two Money values of the same currency.
// On error the receiver's currency is preserved in the returned Money.
func (m Money) Sub(other Money) (Money, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return Money{amount: 0, currency: m.currency}, &MoneyError{Op: "Sub", Err: err}
	}
	if other.amount > 0 && m.amount < math.MinInt64+other.amount {
		return Money{amount: 0, currency: m.currency}, &MoneyError{Op: "Sub", Err: ErrOverflow}
	}
	if other.amount < 0 && m.amount > math.MaxInt64+other.amount {
		return Money{amount: 0, currency: m.currency}, &MoneyError{Op: "Sub", Err: ErrOverflow}
	}
	return Money{amount: m.amount - other.amount, currency: m.currency}, nil
}

// Mul scales the amount by an integer factor. Use it for "n items at price p".
// For percentages or interest, compute in a decimal library and re-enter via
// New so precision is explicit.
func (m Money) Mul(multiplier int64) (Money, error) {
	if multiplier == 0 || m.amount == 0 {
		return Money{amount: 0, currency: m.currency}, nil
	}
	if multiplier > 0 {
		if m.amount > 0 && m.amount > math.MaxInt64/multiplier {
			return Money{amount: 0, currency: m.currency}, &MoneyError{Op: "Mul", Err: ErrOverflow}
		}
		if m.amount < 0 && m.amount < math.MinInt64/multiplier {
			return Money{amount: 0, currency: m.currency}, &MoneyError{Op: "Mul", Err: ErrOverflow}
		}
	} else {
		if multiplier == -1 {
			if m.amount == math.MinInt64 {
				return Money{amount: 0, currency: m.currency}, &MoneyError{Op: "Mul", Err: ErrOverflow}
			}
		} else {
			if m.amount > 0 && m.amount > math.MinInt64/multiplier {
				return Money{amount: 0, currency: m.currency}, &MoneyError{Op: "Mul", Err: ErrOverflow}
			}
			if m.amount < 0 && m.amount < math.MaxInt64/multiplier {
				return Money{amount: 0, currency: m.currency}, &MoneyError{Op: "Mul", Err: ErrOverflow}
			}
		}
	}
	return Money{amount: m.amount * multiplier, currency: m.currency}, nil
}

// Split divides the amount into n parts of equal magnitude, distributing any
// remainder one minor unit at a time to the first parts so the sum equals the
// original amount exactly.
func (m Money) Split(n int) ([]Money, error) {
	if n <= 0 || n > MaxSplitParts {
		return nil, &MoneyError{Op: "Split", Err: ErrInvalidSplitCount}
	}
	quotient := m.amount / int64(n)
	remainder := m.amount % int64(n)
	if remainder < 0 {
		remainder = -remainder
	}
	results := make([]Money, n)
	for i := 0; i < n; i++ {
		val := quotient
		if int64(i) < remainder {
			if m.amount >= 0 {
				val++
			} else {
				val--
			}
		}
		results[i] = Money{amount: val, currency: m.currency}
	}
	return results, nil
}

// Cmp returns -1, 0, or 1 for less than, equal, greater than. It panics on
// currency mismatch, matching the shape of bytes.Compare and big.Int.Cmp so
// it can be used directly as a sort.Slice adapter. Callers that cannot
// guarantee same-currency inputs should use Compare instead.
//
// The panic value is the *MoneyError returned by Compare. Callers using
// recover should type-assert against the error interface (or *MoneyError
// directly), not against string.
func (m Money) Cmp(other Money) int {
	n, err := m.Compare(other)
	if err != nil {
		panic(err)
	}
	return n
}

// Compare returns -1, 0, or 1 for less than, equal, greater than. A
// zero-value Money always fails to compare. The int result is undefined
// when the returned error is non-nil.
func (m Money) Compare(other Money) (int, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return 0, &MoneyError{Op: "Compare", Err: err}
	}
	switch {
	case m.amount < other.amount:
		return -1, nil
	case m.amount > other.amount:
		return 1, nil
	default:
		return 0, nil
	}
}

// Equal reports whether m and other have the same currency and amount. It
// returns false rather than an error on mismatch, matching the shape of
// time.Time.Equal so it can be used ergonomically in conditionals. Two
// zero-value Money values are equal to each other, again matching
// time.Time.Equal.
func (m Money) Equal(other Money) bool {
	if m.currency == nil && other.currency == nil {
		return m.amount == other.amount
	}
	if m.currency == nil || other.currency == nil {
		return false
	}
	return m.currency.isoCode == other.currency.isoCode && m.amount == other.amount
}
