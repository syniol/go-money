package money

import (
	"errors"
	"fmt"
)

var (
	ErrUnsafeScale       = errors.New("currency decimals exceed the safe limit for int64 container")
	ErrInvalidCurrency   = errors.New("invalid or unsupported currency code")
	ErrInvalidFormat     = errors.New("invalid money format")
	ErrTooMuchDetail     = errors.New("string scale exceeds currency decimals")
	ErrCurrencyMismatch  = errors.New("currencies do not match")
	ErrOverflow          = errors.New("arithmetic overflow")
	ErrAmountTooLarge    = errors.New("amount exceeds maximum safe value")
	ErrInputTooLong      = errors.New("input string exceeds maximum allowed length")
	ErrEmptyInput        = errors.New("input string cannot be empty")
	ErrMalformedInput    = errors.New("input contains invalid characters or structure")
	ErrInvalidSplitCount = fmt.Errorf("split count must be between 1 and %d", MaxSplitParts)
	ErrInvalidRoundingMode = errors.New("invalid rounding mode")
)

// MoneyError wraps a sentinel error with contextual information about the
// failed operation. Compare against sentinels with errors.Is rather than
// pattern-matching the message string.
type MoneyError struct {
	Op       string
	Amount   string
	Currency string
	Err      error
}

func (e *MoneyError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Amount != "" && e.Currency != "" {
		return fmt.Sprintf("money.%s(%s, %s): %v", e.Op, e.Amount, e.Currency, e.Err)
	}
	if e.Amount != "" {
		return fmt.Sprintf("money.%s(%s): %v", e.Op, e.Amount, e.Err)
	}
	return fmt.Sprintf("money.%s: %v", e.Op, e.Err)
}

func (e *MoneyError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
