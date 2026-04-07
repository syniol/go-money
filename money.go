package money

//go:generate go run cmd/gen_currencies/main.go

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	ErrUnsafeScale      = errors.New("currency decimals exceed the safe limit for int64 container")
	ErrInvalidCurrency  = errors.New("invalid or unsupported currency code")
	ErrInvalidFormat    = errors.New("invalid money format")
	ErrTooMuchDetail    = errors.New("string scale exceeds currency decimals")
	ErrCurrencyMismatch = errors.New("currencies do not match")
	ErrOverflow         = errors.New("arithmetic overflow")
)

// Money uses strict value semantics to remain on the stack.
// are kept on the stack rather than the heap, avoiding GC pressure.
// nolint:govet // We intentionally mix receivers here to mimic time.Time for JSON unmarshaling.
type Money struct {
	amount   int64
	currency *Config
}

type Config struct {
	ISOCode      string
	Name         string
	Demonym      string
	MajorSingle  string
	MajorPlural  string
	ISONum       int
	Symbol       string
	SymbolNative string
	MinorSingle  string
	MinorPlural  string
	ISODigits    int
	Decimals     int
	NumToBasic   int
}

type RoundingMode int

const (
	RoundHalfToEven       RoundingMode = iota // Banker's Rounding
	RoundHalfAwayFromZero                     // Common School Rounding
	RoundDown                                 // Floor
	RoundUp                                   // Ceiling
)

const MaxSafeDecimals = 12

// New creates a Money instance using int64 minor units.
// Example: "10.50" USD -> 1050
// It uses the pre-generated map (stored in currencies.gen.go)
func New(minorAmount int64, currencyCode string) (Money, error) {
	cfg, exists := currencyConfig[currencyCode]
	if !exists {
		return Money{}, ErrInvalidCurrency
	}

	// The "Safe Scale" check
	if err := validateScale(cfg); err != nil {
		return Money{}, err
	}

	return Money{
		amount:   minorAmount,
		currency: cfg,
	}, nil
}

// MustNew is for hardcoded values or global vars. It panics on error.
// Use this sparingly in production code; primarily for tests or constants.
func MustNew(minorAmount int64, currencyCode string) Money {
	m, err := New(minorAmount, currencyCode)
	if err != nil {
		panic(err)
	}
	return m
}

// NewFromString creates a Money instance using int64 minor units.
// Example: "10.50" USD -> 1050
func NewFromString(value string, currencyCode string) (Money, error) {
	cfg, exists := currencyConfig[currencyCode]
	if !exists {
		return Money{}, ErrInvalidCurrency
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return Money{}, ErrInvalidFormat
	}

	// 1. Check for negativity
	negative := false
	if value[0] == '-' {
		negative = true
		value = value[1:]
	}

	// 2. Split into whole and fractional parts
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return Money{}, ErrInvalidFormat
	}

	wholeStr := parts[0]
	fracStr := ""
	if len(parts) == 2 {
		fracStr = parts[1]
	}

	// 3. Strict decimal enforcement
	if len(fracStr) > cfg.Decimals {
		return Money{}, ErrTooMuchDetail
	}

	// 4. Pad the fraction (e.g., "5" becomes "50" for USD)
	for len(fracStr) < cfg.Decimals {
		fracStr += "0"
	}

	// 5. Build the combined string (e.g., "10" + "50" = "1050")
	combinedStr := wholeStr + fracStr
	if combinedStr == "" {
		combinedStr = "0"
	}

	// 6. Manual string-to-int64 conversion with overflow check
	// We avoid strconv.ParseInt because we need to be extremely careful with overflow
	var res int64
	for i := 0; i < len(combinedStr); i++ {
		digit := int64(combinedStr[i] - '0')
		if digit < 0 || digit > 9 {
			return Money{}, ErrInvalidFormat
		}

		// Overflow check: res * 10 + digit
		if res > (math.MaxInt64-digit)/10 {
			return Money{}, ErrOverflow
		}
		res = res*10 + digit
	}

	if negative {
		res = -res
	}

	return Money{
		amount:   res,
		currency: cfg,
	}, nil
}

// FromDecimal converts a high-precision value (represented as a float or decimal)
// into a final Money value using a specific rounding strategy.
// In finance, Banker's Rounding (Half-to-Even) is the gold standard because it reduces cumulative bias in large datasets.
func FromDecimal(value float64, currencyCode string, mode RoundingMode) (Money, error) {
	cfg, exists := currencyConfig[currencyCode]
	if !exists {
		return Money{}, ErrInvalidCurrency
	}

	// 1. Convert to the scale of the minor unit
	// e.g., $10.506 USD -> 1050.6 cents
	multiplier := math.Pow10(cfg.Decimals)
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
	}

	return Money{amount: rounded, currency: cfg}, nil
}

// Minor returns the raw underlying minor unit (e.g., 10050 for $100.50).
func (m Money) Minor() int64 {
	return m.amount
}

// Float returns the float representation ONLY for display or boundary transit.
// NEVER use this for internal math.
func (m Money) Float() float64 {
	if m.currency.Decimals == 0 {
		return float64(m.amount)
	}

	divisor := 1.0
	for i := 0; i < m.currency.Decimals; i++ {
		divisor *= 10.0
	}
	return float64(m.amount) / divisor
}

// IsEqual safely compares two Money objects. Zero allocations.
func (m Money) IsEqual(other Money) (bool, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return false, err
	}
	return m.amount == other.amount, nil
}

func (m Money) IsLess(other Money) (bool, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return false, err
	}
	return m.amount < other.amount, nil
}

func (m Money) IsGreat(other Money) (bool, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return false, err
	}
	return m.amount > other.amount, nil
}

// Add performs a thread-safe, overflow-checked addition.
func (m Money) Add(other Money) (Money, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return Money{}, err
	}

	// Overflow check: if both are positive and result is smaller than either,
	// or both are negative and result is larger than either.
	res := m.amount + other.amount
	if (other.amount > 0 && res < m.amount) || (other.amount < 0 && res > m.amount) {
		return Money{}, ErrOverflow
	}

	return Money{amount: res, currency: m.currency}, nil
}

// Sub performs overflow-checked subtraction.
func (m Money) Sub(other Money) (Money, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return Money{}, err
	}

	res := m.amount - other.amount
	// Check for overflow: subtraction is essentially adding a negative.
	if (other.amount > 0 && res > m.amount) || (other.amount < 0 && res < m.amount) {
		return Money{}, ErrOverflow
	}

	return Money{amount: res, currency: m.currency}, nil
}

// Mul handles integer multiplication (e.g., "3 of these items").
// For percentages (tax/interest), we'd use a different approach (Decimal).
func (m Money) Mul(multiplier int64) (Money, error) {
	if multiplier == 0 || m.amount == 0 {
		return Money{amount: 0, currency: m.currency}, nil
	}

	res := m.amount * multiplier
	if res/multiplier != m.amount {
		return Money{}, ErrOverflow
	}

	return Money{amount: res, currency: m.currency}, nil
}

// Split divides the money into N parts, distributing remainders fairly. This is critical for reconciliation.
// It distributes the remainder penny-by-penny across the participants.
func (m Money) Split(n int) ([]Money, error) {
	if n <= 0 {
		return nil, errors.New("split count must be positive")
	}

	quotient := m.amount / int64(n)
	remainder := m.amount % int64(n)

	results := make([]Money, n)
	for i := 0; i < n; i++ {
		val := quotient
		// Distribute the remainder penny-by-penny
		if int64(i) < remainder {
			val++
		}
		results[i] = Money{amount: val, currency: m.currency}
	}

	return results, nil
}

// Compare used for sorting ledgers or validating balances against credit limits.
// We want this to be readable and allocation-free.
func (m Money) Compare(other Money) (int, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return 0, err
	}
	if m.amount < other.amount {
		return -1, nil
	}
	if m.amount > other.amount {
		return 1, nil
	}
	return 0, nil
}

// IsZero Helper methods for readability
func (m Money) IsZero() bool { return m.amount == 0 }

// IsPositive Helper methods for readability
func (m Money) IsPositive() bool { return m.amount > 0 }

// IsNegative Helper methods for readability
func (m Money) IsNegative() bool { return m.amount < 0 }

// String returns a localized string representation.
func (m Money) String() string {
	if m.currency.Decimals == 0 {
		return fmt.Sprintf("%s%d", m.currency.Symbol, m.amount)
	}

	divisor := math.Pow10(m.currency.Decimals)
	floatVal := float64(m.amount) / divisor

	format := fmt.Sprintf("%%s%%.%df", m.currency.Decimals)
	return fmt.Sprintf(format, m.currency.Symbol, floatVal)
}

// LocalisedString uses a locale/CLDR provider.
// In a high-tier system, the Money package provides the data, but a LocaleProvider provides the context.
// The Common Locale Data Repository (CLDR) is the industry-standard database maintained by Unicode. It contains the rules for every culture on earth regarding:
// Decimal Separators: Is it 10.50 (US) or 10,50 (Germany)?
// Grouping Separators: Is it 1,000 (UK) or 1.000 (Italy) or 1 000 (France)?
// Symbol Placement: Does the symbol go before ($100) or after (100 €)?
// Currency Spacing: Is there a space between the symbol and the number? (100 руб vs $100).
func (m Money) LocalisedString(tag language.Tag) string {
	// message.NewPrinter is the "Provider" that consults CLDR data
	p := message.NewPrinter(tag)

	// currency.ISO handles the formatting rules for the specific code
	cur, _ := currency.ParseISO(m.currency.ISOCode)

	// The printer automatically handles decimal points,
	// grouping separators, and symbol placement based on the language tag.
	return p.Sprint(currency.NarrowSymbol(cur.Amount(m.Float())))
}

// MarshalJSON implements the json.Marshaler interface.
// It renders as {"amount":"10.50","currency":"USD"} to prevent
// client-side floating-point precision loss.
func (m Money) MarshalJSON() ([]byte, error) {
	if m.currency == nil {
		return nil, errors.New("cannot marshal money without currency configuration")
	}

	divisor := math.Pow10(m.currency.Decimals)
	// We use %.*f to ensure we always output the correct number of decimal places
	amountStr := fmt.Sprintf("%.*f", m.currency.Decimals, float64(m.amount)/divisor)

	// Manual byte construction is faster than json.Marshal(struct{...})
	// for high-throughput services as it avoids reflection.
	return []byte(fmt.Sprintf(`{"amount":"%s","currency":"%s"}`, amountStr, m.currency.ISOCode)), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It expects the same {"amount":"10.50","currency":"USD"} format.
func (m *Money) UnmarshalJSON(data []byte) error {
	var aux struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Reuse our hardened string parser to ensure consistency
	money, err := NewFromString(aux.Amount, aux.Currency)
	if err != nil {
		return fmt.Errorf("failed to unmarshal money: %w", err)
	}

	*m = money
	return nil
}

// validateScale ensures the currency is actually safe to use with int64 math.
func validateScale(cfg *Config) error {
	if cfg.Decimals > MaxSafeDecimals {
		return ErrUnsafeScale
	}
	return nil
}

// assertSameCurrency ensures same currency used for comparison helper methods
func (m Money) assertSameCurrency(other Money) error {
	if m.currency == nil || other.currency == nil || m.currency.ISOCode != other.currency.ISOCode {
		return ErrCurrencyMismatch
	}
	return nil
}
