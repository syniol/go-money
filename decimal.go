package money

import (
	"fmt"
	"math"
)

// RoundingMode selects how FromDecimal maps a fractional float to minor units.
type RoundingMode int

const (
	// RoundHalfToEven implements banker's rounding, minimising cumulative bias.
	RoundHalfToEven RoundingMode = iota
	// RoundHalfAwayFromZero is traditional school rounding.
	RoundHalfAwayFromZero
	// RoundDown floors toward negative infinity.
	RoundDown
	// RoundUp ceilings toward positive infinity.
	RoundUp
)

// FromDecimal converts a float64 to Money using the given rounding mode.
//
// WARNING: float64 cannot represent every decimal fraction exactly (for
// example, 1.005 is stored as 1.00499999...). For amounts that must
// round-trip exactly, prefer NewFromString or New. FromDecimal is provided
// for callers migrating from float-based systems and for interoperating with
// libraries that hand you a float.
func FromDecimal(value float64, currencyCode string, mode RoundingMode) (Money, error) {
	if currencyCode == "" {
		return Money{}, &MoneyError{Op: "FromDecimal", Amount: fmt.Sprintf("%.6f", value), Err: ErrInvalidCurrency}
	}
	cfg, ok := currencyConfig[currencyCode]
	if !ok {
		return Money{}, &MoneyError{Op: "FromDecimal", Amount: fmt.Sprintf("%.6f", value), Currency: currencyCode, Err: ErrInvalidCurrency}
	}
	multiplier := float64(getPow10(cfg.Decimals))
	scaledValue := value * multiplier
	// float64(math.MaxInt64) rounds up to 9.223372036854776e18 (one past
	// MaxInt64) because MaxInt64 needs 63 bits and float64 only has 52 bits
	// of mantissa. Use >= so scaledValue == float64(math.MaxInt64) is
	// rejected rather than fed to a spec-undefined int64 conversion.
	if math.IsNaN(scaledValue) || math.IsInf(scaledValue, 0) || scaledValue >= float64(math.MaxInt64) || scaledValue <= float64(math.MinInt64) {
		return Money{amount: 0, currency: cfg}, &MoneyError{Op: "FromDecimal", Amount: fmt.Sprintf("%.6f", value), Currency: currencyCode, Err: ErrAmountTooLarge}
	}
	var rounded int64
	switch mode {
	case RoundHalfToEven:
		rounded = int64(math.RoundToEven(scaledValue))
	case RoundHalfAwayFromZero:
		rounded = int64(math.Round(scaledValue))
	case RoundDown:
		rounded = int64(math.Floor(scaledValue))
	case RoundUp:
		rounded = int64(math.Ceil(scaledValue))
	default:
		return Money{amount: 0, currency: cfg}, &MoneyError{Op: "FromDecimal", Amount: fmt.Sprintf("%.6f", value), Currency: currencyCode, Err: ErrInvalidRoundingMode}
	}
	return Money{amount: rounded, currency: cfg}, nil
}
