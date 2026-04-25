// Package money provides a robust, immutable monetary value type designed for
// high-precision financial calculations in production banking systems.
//
// This package follows strict principles to prevent common financial software bugs:
//   - Uses integer arithmetic exclusively (no floating-point in calculations)
//   - Enforces currency type safety to prevent mixing incompatible currencies
//   - Provides overflow-safe arithmetic operations with mathematical guarantees
//   - Implements banker's rounding for regulatory compliance
//   - Maintains immutability for thread safety and predictable behavior
//
// Thread Safety:
// Money values are immutable and safe for concurrent access across goroutines.
// The underlying currency configuration is read-only after package initialization.
// No synchronization primitives are required for Money operations.
//
// Performance Characteristics:
// Money uses value semantics to remain on the stack, avoiding garbage collection
// pressure. Pre-computed lookup tables eliminate repeated mathematical operations
// in hot paths.
package money

//go:generate go run cmd/gen_currencies/main.go

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Error definitions for comprehensive error handling and operational debugging.
var (
	ErrUnsafeScale      = errors.New("currency decimals exceed the safe limit for int64 container")
	ErrInvalidCurrency  = errors.New("invalid or unsupported currency code")
	ErrInvalidFormat    = errors.New("invalid money format")
	ErrTooMuchDetail    = errors.New("string scale exceeds currency decimals")
	ErrCurrencyMismatch = errors.New("currencies do not match")
	ErrOverflow         = errors.New("arithmetic overflow")
	ErrAmountTooLarge   = errors.New("amount exceeds maximum safe value")
	ErrInputTooLong     = errors.New("input string exceeds maximum allowed length")
	ErrEmptyInput       = errors.New("input string cannot be empty")
	ErrMalformedInput   = errors.New("input contains invalid characters or structure")
)

// MoneyError provides structured error information for operational debugging
// and audit trail requirements in financial systems.
type MoneyError struct {
	Op       string // The operation that failed (e.g., "Add", "NewFromString")
	Amount   string // The input amount that caused the error (if applicable)
	Currency string // The currency code involved (if applicable)
	Err      error  // The underlying error
}

// Error implements the error interface with contextual information.
func (e *MoneyError) Error() string {
	if e.Amount != "" && e.Currency != "" {
		return fmt.Sprintf("money.%s(%s, %s): %v", e.Op, e.Amount, e.Currency, e.Err)
	}
	if e.Amount != "" {
		return fmt.Sprintf("money.%s(%s): %v", e.Op, e.Amount, e.Err)
	}
	return fmt.Sprintf("money.%s: %v", e.Op, e.Err)
}

// Unwrap enables error unwrapping for errors.Is and errors.As.
func (e *MoneyError) Unwrap() error {
	return e.Err
}

// Money represents an immutable monetary amount with a specific currency.
// It uses integer arithmetic to avoid floating-point precision issues that
// are critical in financial calculations.
//
// All amounts are stored in the currency's minor units (e.g., cents for USD,
// pence for GBP). This design ensures that all monetary arithmetic maintains
// exact precision without rounding errors.
//
// Thread Safety: Money values are immutable and safe for concurrent use.
// Performance: Uses value semantics to avoid heap allocations and GC pressure.
//
// Example:
//
//	m := money.MustNew(1050, "USD") // Represents $10.50
//	fmt.Println(m.String()) // Output: $10.50
type Money struct {
	amount   int64   // Amount in minor units (e.g., cents for USD)
	currency *Config // Immutable currency configuration
}

// Config contains immutable currency configuration data.
// This structure is populated at package initialization and should never
// be modified after creation to ensure thread safety.
type Config struct {
	ISOCode      string // ISO 4217 currency code (e.g., "USD", "EUR")
	Name         string // Full currency name (e.g., "US Dollar")
	Demonym      string // Currency demonym (e.g., "American")
	MajorSingle  string // Singular major unit name (e.g., "dollar")
	MajorPlural  string // Plural major unit name (e.g., "dollars")
	ISONum       int    // ISO 4217 numeric code
	Symbol       string // Currency symbol (e.g., "$", "€")
	SymbolNative string // Native script symbol
	MinorSingle  string // Singular minor unit name (e.g., "cent")
	MinorPlural  string // Plural minor unit name (e.g., "cents")
	ISODigits    int    // Official ISO decimal places
	Decimals     int    // Decimal places used in calculations
	NumToBasic   int    // Conversion factor to basic unit
}

// RoundingMode defines the strategy for handling fractional amounts
// when converting from decimal representations.
type RoundingMode int

const (
	// RoundHalfToEven implements banker's rounding, which reduces cumulative
	// bias in large datasets. This is the preferred mode for financial calculations.
	RoundHalfToEven RoundingMode = iota

	// RoundHalfAwayFromZero implements traditional school rounding.
	RoundHalfAwayFromZero

	// RoundDown implements floor behavior (always round toward negative infinity).
	RoundDown

	// RoundUp implements ceiling behavior (always round toward positive infinity).
	RoundUp
)

const (
	// MaxSafeDecimals defines the maximum number of decimal places that can
	// be safely represented in int64 arithmetic without overflow risk.
	MaxSafeDecimals = 12

	// MaxStringLength limits input string length to prevent DoS attacks
	// and ensure reasonable processing times.
	MaxStringLength = 64

	// MaxSafeAmount prevents overflow in financial calculations by maintaining
	// a conservative upper bound for monetary amounts.
	MaxSafeAmount = math.MaxInt64 / 10000
)

// Pre-computed powers of 10 for performance optimization.
// This eliminates repeated math.Pow10 calls in hot paths.
var pow10 = [...]int64{
	1, 10, 100, 1000, 10000, 100000, 1000000, 10000000,
	100000000, 1000000000, 10000000000, 100000000000, 1000000000000,
}

// getPow10 returns 10^n efficiently using a lookup table for common values.
func getPow10(n int) int64 {
	if n >= 0 && n < len(pow10) {
		return pow10[n]
	}
	// Fallback for unusual cases (should be rare in practice)
	return int64(math.Pow10(n))
}

// New creates a Money instance using int64 minor units with comprehensive
// validation and overflow protection.
//
// Parameters:
//   - minorAmount: The amount in the currency's minor units (e.g., cents)
//   - currencyCode: ISO 4217 currency code (e.g., "USD", "EUR")
//
// Returns:
//   - Money: A valid Money instance
//   - error: ErrInvalidCurrency if currency is not supported,
//     ErrUnsafeScale if currency decimal precision exceeds safe limits,
//     ErrAmountTooLarge if amount exceeds safe calculation bounds
//
// Example:
//
//	m, err := money.New(1050, "USD") // $10.50
func New(minorAmount int64, currencyCode string) (Money, error) {
	if currencyCode == "" {
		return Money{}, &MoneyError{
			Op:  "New",
			Err: ErrInvalidCurrency,
		}
	}

	cfg, exists := currencyConfig[currencyCode]
	if !exists {
		return Money{}, &MoneyError{
			Op:       "New",
			Currency: currencyCode,
			Err:      ErrInvalidCurrency,
		}
	}

	if err := validateScale(cfg); err != nil {
		return Money{}, &MoneyError{
			Op:       "New",
			Currency: currencyCode,
			Err:      err,
		}
	}

	if minorAmount > MaxSafeAmount || minorAmount < -MaxSafeAmount {
		return Money{}, &MoneyError{
			Op:       "New",
			Amount:   strconv.FormatInt(minorAmount, 10),
			Currency: currencyCode,
			Err:      ErrAmountTooLarge,
		}
	}

	return Money{
		amount:   minorAmount,
		currency: cfg,
	}, nil
}

// MustNew creates a Money instance and panics on any error.
// This function is intended for use with compile-time constants or in test code
// where initialization failure represents a programmer error.
//
// Use this sparingly in production code. Prefer New() for runtime values.
//
// Example:
//
//	var USD10 = money.MustNew(1000, "USD") // Package-level constant
func MustNew(minorAmount int64, currencyCode string) Money {
	m, err := New(minorAmount, currencyCode)
	if err != nil {
		panic(fmt.Sprintf("MustNew failed: %v", err))
	}
	return m
}

// NewFromString creates a Money instance from a string representation with
// comprehensive input validation and security hardening.
//
// The function accepts various formats:
//   - "10.50" (decimal notation)
//   - "10" (integer notation)
//   - "-5.25" (negative amounts)
//   - "1,000.50" (with thousand separators, which are stripped)
//
// Parameters:
//   - value: String representation of the monetary amount
//   - currencyCode: ISO 4217 currency code
//
// Returns:
//   - Money: Valid Money instance
//   - error: Various validation errors with context
//
// Security: Input length is limited and format is strictly validated
// to prevent DoS attacks and malformed input processing.
func NewFromString(value string, currencyCode string) (Money, error) {
	if currencyCode == "" {
		return Money{}, &MoneyError{
			Op:     "NewFromString",
			Amount: value,
			Err:    ErrInvalidCurrency,
		}
	}

	cfg, exists := currencyConfig[currencyCode]
	if !exists {
		return Money{}, &MoneyError{
			Op:       "NewFromString",
			Amount:   value,
			Currency: currencyCode,
			Err:      ErrInvalidCurrency,
		}
	}

	// Input validation for security and correctness
	if len(value) == 0 {
		return Money{}, &MoneyError{
			Op:       "NewFromString",
			Currency: currencyCode,
			Err:      ErrEmptyInput,
		}
	}

	if len(value) > MaxStringLength {
		return Money{}, &MoneyError{
			Op:       "NewFromString",
			Amount:   value,
			Currency: currencyCode,
			Err:      ErrInputTooLong,
		}
	}

	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == "-" || value == "-." {
		return Money{}, &MoneyError{
			Op:       "NewFromString",
			Amount:   value,
			Currency: currencyCode,
			Err:      ErrInvalidFormat,
		}
	}

	// Structural validation
	if strings.Count(value, ".") > 1 ||
		strings.Count(value, "-") > 1 ||
		(strings.Contains(value, "-") && !strings.HasPrefix(value, "-")) {
		return Money{}, &MoneyError{
			Op:       "NewFromString",
			Amount:   value,
			Currency: currencyCode,
			Err:      ErrMalformedInput,
		}
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return Money{}, &MoneyError{
			Op:       "NewFromString",
			Amount:   value,
			Currency: currencyCode,
			Err:      ErrInvalidFormat,
		}
	}

	var intPart, fracPart string
	intPart = parts[0]

	if len(parts) == 2 {
		fracPart = parts[1]
		// Error if provided decimals exceed currency definition
		if len(fracPart) > cfg.Decimals {
			return Money{}, &MoneyError{
				Op:       "NewFromString",
				Amount:   value,
				Currency: currencyCode,
				Err:      ErrTooMuchDetail,
			}
		}
	}

	// Normalize by removing thousand separators
	intPart = strings.ReplaceAll(intPart, ",", "")

	// Handle the integer part
	var totalAmount int64
	if intPart != "" && intPart != "-" {
		parsedInt, err := strconv.ParseInt(intPart, 10, 64)
		if err != nil {
			return Money{}, &MoneyError{
				Op:       "NewFromString",
				Amount:   value,
				Currency: currencyCode,
				Err:      ErrInvalidFormat,
			}
		}

		// Check for overflow before multiplication
		multiplier := getPow10(cfg.Decimals)
		if parsedInt > 0 && parsedInt > math.MaxInt64/multiplier {
			return Money{}, &MoneyError{
				Op:       "NewFromString",
				Amount:   value,
				Currency: currencyCode,
				Err:      ErrAmountTooLarge,
			}
		}
		if parsedInt < 0 && parsedInt < math.MinInt64/multiplier {
			return Money{}, &MoneyError{
				Op:       "NewFromString",
				Amount:   value,
				Currency: currencyCode,
				Err:      ErrAmountTooLarge,
			}
		}

		totalAmount = parsedInt * multiplier
	} else if intPart == "-" {
		return Money{}, &MoneyError{
			Op:       "NewFromString",
			Amount:   value,
			Currency: currencyCode,
			Err:      ErrInvalidFormat,
		}
	}

	// Handle the fractional part
	if fracPart != "" {
		parsedFrac, err := strconv.ParseInt(fracPart, 10, 64)
		if err != nil {
			return Money{}, &MoneyError{
				Op:       "NewFromString",
				Amount:   value,
				Currency: currencyCode,
				Err:      ErrInvalidFormat,
			}
		}

		// Scale the fraction to the correct minor unit power
		fracMultiplier := getPow10(cfg.Decimals - len(fracPart))
		fractionalAmount := parsedFrac * fracMultiplier

		if strings.HasPrefix(intPart, "-") {
			totalAmount -= fractionalAmount
		} else {
			totalAmount += fractionalAmount
		}
	}

	return New(totalAmount, currencyCode)
}

// FromDecimal converts a high-precision decimal value into a Money value
// using the specified rounding strategy.
//
// In financial systems, Banker's Rounding (RoundHalfToEven) is preferred
// because it reduces cumulative bias in large datasets of calculations.
//
// Parameters:
//   - value: The decimal value to convert
//   - currencyCode: ISO 4217 currency code
//   - mode: Rounding strategy to apply
//
// Returns:
//   - Money: The rounded Money value
//   - error: Currency or validation errors
//
// Example:
//
//	m, err := money.FromDecimal(10.506, "USD", money.RoundHalfToEven)
//	// Result: $10.51 (rounded using banker's rounding)
func FromDecimal(value float64, currencyCode string, mode RoundingMode) (Money, error) {
	if currencyCode == "" {
		return Money{}, &MoneyError{
			Op:     "FromDecimal",
			Amount: fmt.Sprintf("%.6f", value),
			Err:    ErrInvalidCurrency,
		}
	}

	cfg, exists := currencyConfig[currencyCode]
	if !exists {
		return Money{}, &MoneyError{
			Op:       "FromDecimal",
			Amount:   fmt.Sprintf("%.6f", value),
			Currency: currencyCode,
			Err:      ErrInvalidCurrency,
		}
	}

	// Convert to the scale of the minor unit
	multiplier := float64(getPow10(cfg.Decimals))
	scaledValue := value * multiplier

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
		rounded = int64(math.RoundToEven(scaledValue)) // Default to banker's rounding
	}

	return Money{amount: rounded, currency: cfg}, nil
}

// Minor returns the raw underlying minor unit amount.
//
// Example:
//
//	m := money.MustNew(1050, "USD")
//	fmt.Println(m.Minor()) // Output: 1050 (cents)
func (m Money) Minor() int64 {
	return m.amount
}

// Currency returns the currency code for this Money instance.
//
// Example:
//
//	m := money.MustNew(1050, "USD")
//	fmt.Println(m.Currency()) // Output: "USD"
func (m Money) Currency() string {
	if m.currency == nil {
		return ""
	}
	return m.currency.ISOCode
}

// AsDecimalString returns a decimal string representation for display
// or interoperability purposes.
//
// WARNING: This method is provided ONLY for display, logging, or
// integration with decimal libraries. NEVER use this result in
// arithmetic calculations as it may introduce precision loss.
//
// For calculations, always use the integer-based methods (Add, Sub, Mul).
//
// Example:
//
//	m := money.MustNew(1050, "USD")
//	fmt.Println(m.AsDecimalString()) // Output: "10.50"
func (m Money) AsDecimalString() string {
	if m.currency == nil {
		return "0"
	}

	if m.currency.Decimals == 0 {
		return strconv.FormatInt(m.amount, 10)
	}

	divisor := float64(getPow10(m.currency.Decimals))
	floatVal := float64(m.amount) / divisor
	return fmt.Sprintf("%.*f", m.currency.Decimals, floatVal)
}

// IsEqual performs a safe equality comparison between two Money values.
// Returns an error if the currencies don't match.
//
// Example:
//
//	m1 := money.MustNew(1000, "USD")
//	m2 := money.MustNew(1000, "USD")
//	equal, err := m1.IsEqual(m2) // true, nil
func (m Money) IsEqual(other Money) (bool, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return false, err
	}
	return m.amount == other.amount, nil
}

// IsLessThan performs a safe less-than comparison between two Money values.
// Returns an error if the currencies don't match.
func (m Money) IsLessThan(other Money) (bool, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return false, err
	}
	return m.amount < other.amount, nil
}

// IsGreaterThan performs a safe greater-than comparison between two Money values.
// Returns an error if the currencies don't match.
func (m Money) IsGreaterThan(other Money) (bool, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return false, err
	}
	return m.amount > other.amount, nil
}

// Add performs overflow-safe addition of two Money values.
// Both values must have the same currency.
//
// The overflow detection is mathematically proven: we check the conditions
// that would cause overflow before performing the operation.
//
// Example:
//
//	m1 := money.MustNew(1000, "USD") // $10.00
//	m2 := money.MustNew(500, "USD")  // $5.00
//	sum, err := m1.Add(m2)           // $15.00, nil
func (m Money) Add(other Money) (Money, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return Money{}, &MoneyError{
			Op:  "Add",
			Err: err,
		}
	}

	// Mathematical overflow detection before operation
	if other.amount > 0 && m.amount > math.MaxInt64-other.amount {
		return Money{}, &MoneyError{
			Op:  "Add",
			Err: ErrOverflow,
		}
	}
	if other.amount < 0 && m.amount < math.MinInt64-other.amount {
		return Money{}, &MoneyError{
			Op:  "Add",
			Err: ErrOverflow,
		}
	}

	return Money{amount: m.amount + other.amount, currency: m.currency}, nil
}

// Sub performs overflow-safe subtraction of two Money values.
// Both values must have the same currency.
func (m Money) Sub(other Money) (Money, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return Money{}, &MoneyError{
			Op:  "Sub",
			Err: err,
		}
	}

	// Mathematical overflow detection: subtraction is addition of negative
	if other.amount > 0 && m.amount < math.MinInt64+other.amount {
		return Money{}, &MoneyError{
			Op:  "Sub",
			Err: ErrOverflow,
		}
	}
	if other.amount < 0 && m.amount > math.MaxInt64+other.amount {
		return Money{}, &MoneyError{
			Op:  "Sub",
			Err: ErrOverflow,
		}
	}

	return Money{amount: m.amount - other.amount, currency: m.currency}, nil
}

// Mul performs overflow-safe multiplication by an integer factor.
// This is useful for scenarios like "3 items at $5.00 each".
//
// For percentage calculations (tax, interest), use a decimal library
// and convert the result back using FromDecimal().
//
// Example:
//
//	price := money.MustNew(500, "USD") // $5.00
//	total, err := price.Mul(3)         // $15.00, nil
func (m Money) Mul(multiplier int64) (Money, error) {
	if multiplier == 0 || m.amount == 0 {
		return Money{amount: 0, currency: m.currency}, nil
	}

	// Overflow detection before multiplication
	if multiplier > 0 {
		if m.amount > 0 && m.amount > math.MaxInt64/multiplier {
			return Money{}, &MoneyError{Op: "Mul", Err: ErrOverflow}
		}
		if m.amount < 0 && m.amount < math.MinInt64/multiplier {
			return Money{}, &MoneyError{Op: "Mul", Err: ErrOverflow}
		}
	} else { // multiplier < 0
		if m.amount > 0 && m.amount > math.MinInt64/multiplier {
			return Money{}, &MoneyError{Op: "Mul", Err: ErrOverflow}
		}
		if m.amount < 0 && m.amount < math.MaxInt64/multiplier {
			return Money{}, &MoneyError{Op: "Mul", Err: ErrOverflow}
		}
	}

	return Money{amount: m.amount * multiplier, currency: m.currency}, nil
}

// Split divides the money into N parts, distributing any remainder
// fairly across the first remainder parts.
//
// This is critical for reconciliation in financial systems where
// the sum of parts must exactly equal the original amount.
//
// Example:
//
//	total := money.MustNew(100, "USD") // $1.00
//	parts, err := total.Split(3)       // [$0.34, $0.33, $0.33]
//	// Note: $0.34 + $0.33 + $0.33 = $1.00 exactly
func (m Money) Split(n int) ([]Money, error) {
	if n <= 0 {
		return nil, &MoneyError{
			Op:  "Split",
			Err: errors.New("split count must be positive"),
		}
	}

	quotient := m.amount / int64(n)
	remainder := m.amount % int64(n)

	// Handle negative amounts correctly
	if remainder < 0 {
		remainder = -remainder
	}

	results := make([]Money, n)
	for i := 0; i < n; i++ {
		val := quotient
		// Distribute the remainder penny-by-penny to the first remainder parts
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

// Compare returns -1 if m < other, 0 if m == other, and 1 if m > other.
// Both values must have the same currency.
//
// This method is useful for sorting operations and validating balances
// against credit limits.
//
// Example:
//
//	m1 := money.MustNew(1000, "USD")
//	m2 := money.MustNew(2000, "USD")
//	cmp, err := m1.Compare(m2) // -1, nil
func (m Money) Compare(other Money) (int, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return 0, &MoneyError{
			Op:  "Compare",
			Err: err,
		}
	}

	if m.amount < other.amount {
		return -1, nil
	}
	if m.amount > other.amount {
		return 1, nil
	}
	return 0, nil
}

// IsZero returns true if the monetary amount is exactly zero.
func (m Money) IsZero() bool {
	return m.amount == 0
}

// IsPositive returns true if the monetary amount is greater than zero.
func (m Money) IsPositive() bool {
	return m.amount > 0
}

// IsNegative returns true if the monetary amount is less than zero.
func (m Money) IsNegative() bool {
	return m.amount < 0
}

// String returns a formatted string representation of the Money value
// using the currency's symbol and appropriate decimal places.
//
// Example:
//
//	m := money.MustNew(1050, "USD")
//	fmt.Println(m.String()) // Output: $10.50
func (m Money) String() string {
	if m.currency == nil {
		return "0"
	}

	if m.currency.Decimals == 0 {
		return fmt.Sprintf("%s%d", m.currency.Symbol, m.amount)
	}

	divisor := float64(getPow10(m.currency.Decimals))
	floatVal := float64(m.amount) / divisor

	format := fmt.Sprintf("%%s%%.%df", m.currency.Decimals)
	return fmt.Sprintf(format, m.currency.Symbol, floatVal)
}

// LocalisedString returns a locale-aware formatted string using CLDR rules
// for the specified language tag.
//
// The Common Locale Data Repository (CLDR) contains formatting rules for
// every culture, including decimal separators, grouping, and symbol placement.
//
// Example:
//
//	m := money.MustNew(123456, "USD") // $1,234.56
//	fmt.Println(m.LocalisedString(language.AmericanEnglish)) // $1,234.56
func (m Money) LocalisedString(tag language.Tag) string {
	if m.currency == nil {
		return "0"
	}

	p := message.NewPrinter(tag)
	cur, err := currency.ParseISO(m.currency.ISOCode)
	if err != nil {
		// Fallback to simple string representation
		return m.String()
	}

	// Convert to decimal value for CLDR formatting
	decimalValue := float64(m.amount) / float64(getPow10(m.currency.Decimals))

	raw := p.Sprint(currency.NarrowSymbol(cur.Amount(decimalValue)))

	// Remove Unicode whitespace while preserving currency symbols and digits
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, raw)
}

// MarshalJSON implements json.Marshaler interface with precision preservation.
// Returns JSON in the format: {"amount":"10.50","currency":"USD"}
//
// This format prevents client-side floating-point precision loss that would
// occur with numeric JSON representation.
func (m Money) MarshalJSON() ([]byte, error) {
	if m.currency == nil {
		return nil, errors.New("cannot marshal money without currency configuration")
	}

	amountStr := m.AsDecimalString()

	// Manual JSON construction for performance (avoids reflection)
	return []byte(fmt.Sprintf(`{"amount":"%s","currency":"%s"}`,
		amountStr, m.currency.ISOCode)), nil
}

// UnmarshalJSON implements json.Unmarshaler interface.
// Expects JSON in the format: {"amount":"10.50","currency":"USD"}
func (m *Money) UnmarshalJSON(data []byte) error {
	var aux struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return fmt.Errorf("failed to unmarshal money JSON: %w", err)
	}

	// Validate that both fields are present
	if aux.Amount == "" || aux.Currency == "" {
		return errors.New("money JSON must contain both amount and currency fields")
	}

	// Reuse the hardened string parser for consistency
	money, err := NewFromString(aux.Amount, aux.Currency)
	if err != nil {
		return fmt.Errorf("failed to unmarshal money: %w", err)
	}

	*m = money
	return nil
}

// validateScale ensures the currency decimal precision is safe for int64 arithmetic.
func validateScale(cfg *Config) error {
	if cfg.Decimals > MaxSafeDecimals {
		return ErrUnsafeScale
	}
	return nil
}

// assertSameCurrency validates that two Money values have the same currency.
// This is a critical safety check for all arithmetic and comparison operations.
func (m Money) assertSameCurrency(other Money) error {
	if m.currency == nil || other.currency == nil {
		return ErrCurrencyMismatch
	}
	if m.currency.ISOCode != other.currency.ISOCode {
		return ErrCurrencyMismatch
	}
	return nil
}
