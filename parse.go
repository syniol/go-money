package money

import (
	"math"
	"strconv"
	"strings"
)

// NewFromString parses an amount written as an ASCII decimal string. The input
// must use only ASCII digits, at most one leading '-', and at most one '.'.
// Commas, whitespace inside the number, and non-ASCII numeral scripts are
// rejected to prevent homoglyph attacks and localisation ambiguity. Upstream
// layers must canonicalise input before calling.
func NewFromString(value string, currencyCode string) (Money, error) {
	if currencyCode == "" {
		return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Err: ErrInvalidCurrency}
	}
	cfg, ok := currencyConfig[currencyCode]
	if !ok {
		return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrInvalidCurrency}
	}

	if len(value) == 0 {
		return Money{}, &MoneyError{Op: "NewFromString", Currency: currencyCode, Err: ErrEmptyInput}
	}

	value = strings.TrimSpace(value)
	if value == "" {
		// An all-whitespace input became empty after trimming; classify it
		// as empty rather than too long or bad format.
		return Money{}, &MoneyError{Op: "NewFromString", Currency: currencyCode, Err: ErrEmptyInput}
	}
	if len(value) > MaxStringLength {
		// Enforce the length cap on the trimmed value so a padded 65-space
		// input is classified as empty, not too long.
		return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrInputTooLong}
	}
	switch value {
	case ".", "-", "-.":
		return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrInvalidFormat}
	}

	if strings.Contains(value, ",") {
		return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrInvalidFormat}
	}

	if strings.Count(value, ".") > 1 ||
		strings.Count(value, "-") > 1 ||
		(strings.Contains(value, "-") && !strings.HasPrefix(value, "-")) {
		return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrMalformedInput}
	}

	intPart, fracPart, _ := strings.Cut(value, ".")

	digits := strings.TrimPrefix(intPart, "-")
	for _, r := range digits {
		if r < '0' || r > '9' {
			return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrInvalidFormat}
		}
	}
	for _, r := range fracPart {
		if r < '0' || r > '9' {
			return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrInvalidFormat}
		}
	}

	if len(fracPart) > cfg.decimals {
		return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrTooMuchDetail}
	}

	var totalAmount int64
	if intPart != "" && intPart != "-" {
		parsedInt, err := strconv.ParseInt(intPart, 10, 64)
		if err != nil {
			// ParseInt already validated that digits are ASCII (we did that
			// above); the only remaining failure is int64 overflow, which is
			// an amount problem, not a format problem.
			if numErr, ok := err.(*strconv.NumError); ok && numErr.Err == strconv.ErrRange {
				return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrAmountTooLarge}
			}
			return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrInvalidFormat}
		}
		multiplier := getPow10(cfg.decimals)
		if parsedInt > 0 && parsedInt > math.MaxInt64/multiplier {
			return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrAmountTooLarge}
		}
		if parsedInt < 0 && parsedInt < math.MinInt64/multiplier {
			return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrAmountTooLarge}
		}
		totalAmount = parsedInt * multiplier
	} else if intPart == "-" {
		return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrInvalidFormat}
	}

	if fracPart != "" {
		// ParseInt cannot fail here: the loop above rejected any non-digit
		// rune, and len(fracPart) <= cfg.decimals <= MaxSafeDecimals = 12,
		// which always fits in int64.
		parsedFrac, _ := strconv.ParseInt(fracPart, 10, 64)
		fracMultiplier := getPow10(cfg.decimals - len(fracPart))
		fractionalAmount := parsedFrac * fracMultiplier
		if strings.HasPrefix(intPart, "-") {
			if totalAmount < math.MinInt64+fractionalAmount {
				return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrAmountTooLarge}
			}
			totalAmount -= fractionalAmount
		} else {
			if totalAmount > math.MaxInt64-fractionalAmount {
				return Money{}, &MoneyError{Op: "NewFromString", Amount: value, Currency: currencyCode, Err: ErrAmountTooLarge}
			}
			totalAmount += fractionalAmount
		}
	}

	return Money{amount: totalAmount, currency: cfg}, nil
}
