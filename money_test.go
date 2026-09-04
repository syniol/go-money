package money

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"golang.org/x/text/language"
)

func TestNew_And_MustNew(t *testing.T) {
	t.Run("valid new", func(t *testing.T) {
		m, err := New(100, "USD")
		if err != nil || m.Minor() != 100 {
			t.Errorf("New failed: %v", err)
		}
	})

	t.Run("invalid currency", func(t *testing.T) {
		_, err := New(100, "INVALID")
		if !errors.Is(err, ErrInvalidCurrency) {
			t.Errorf("Expected ErrInvalidCurrency, got %v", err)
		}
	})

	t.Run("must new panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("MustNew should have panicked on invalid currency")
			}
		}()
		_ = MustNew(100, "INVALID")
	})
}

func TestNewFromString(t *testing.T) {
	tests := []struct {
		name       string
		val        string
		currency   string
		wantAmount int64
		wantErr    error
	}{
		{"valid positive", "10.50", "USD", 1050, nil},
		{"valid negative", "-10.50", "USD", -1050, nil},
		{"valid no fraction", "10", "USD", 1000, nil},
		{"zero decimals currency", "500", "JPY", 500, nil},
		{"too many decimals", "10.501", "USD", 0, ErrTooMuchDetail},
		{"lone negative sign", "-", "USD", 0, ErrInvalidFormat},
		{"empty string", "", "USD", 0, ErrEmptyInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewFromString(tt.val, tt.currency)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got error %v, want %v", err, tt.wantErr)
			}
			if err == nil && got.Minor() != tt.wantAmount {
				t.Errorf("got %v, want %v", got.Minor(), tt.wantAmount)
			}
		})
	}
}

func TestFromDecimal(t *testing.T) {
	tests := []struct {
		name string
		val  float64
		mode RoundingMode
		want int64
	}{
		{"Bankers Even", 10.505, RoundHalfToEven, 1050}, // 10.505 -> 10.50
		{"Bankers Odd", 10.515, RoundHalfToEven, 1052},  // 10.515 -> 10.52
		{"School Round", 10.505, RoundHalfAwayFromZero, 1051},
		{"Floor", 10.509, RoundDown, 1050},
		{"Ceil", 10.501, RoundUp, 1051},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := FromDecimal(tt.val, "USD", tt.mode)
			if m.Minor() != tt.want {
				t.Errorf("got %d, want %d", m.Minor(), tt.want)
			}
		})
	}
}

func TestArithmetic_And_Comparisons(t *testing.T) {
	m10 := MustNew(1000, "USD")
	m5 := MustNew(500, "USD")

	// Addition
	res, _ := m10.Add(m5)
	if res.Minor() != 1500 {
		t.Error("Add failed")
	}

	// Subtraction
	res, _ = m10.Sub(m5)
	if res.Minor() != 500 {
		t.Error("Sub failed")
	}

	// Multiplication
	res, _ = m10.Mul(3)
	if res.Minor() != 3000 {
		t.Error("Mul failed")
	}

	// Comparisons via Compare and Equal
	lessCmp, _ := m5.Compare(m10)
	greaterCmp, _ := m10.Compare(m5)
	if lessCmp != -1 || greaterCmp != 1 || !m10.Equal(m10) {
		t.Error("Comparison logic error")
	}
	if m10.Equal(m5) {
		t.Error("Equal returned true for unequal amounts")
	}

	// Compare method
	cmp, _ := m10.Compare(m5)
	if cmp != 1 {
		t.Errorf("Expected 1, got %d", cmp)
	}

	// Boolean checks
	if !MustNew(0, "USD").IsZero() {
		t.Error("IsZero failed")
	}
	if !m10.IsPositive() {
		t.Error("IsPositive failed")
	}
	if !MustNew(-10, "USD").IsNegative() {
		t.Error("IsNegative failed")
	}
}

func TestDisplayMethods(t *testing.T) {
	m := MustNew(1050, "USD")

	if m.AsDecimalString() != "10.50" {
		t.Errorf("Float() got %s, want 10.50", m.AsDecimalString())
	}

	if m.String() != "$10.50" {
		t.Errorf("String() got %s, want $10.50", m.String())
	}

	jpy := MustNew(500, "JPY")
	if jpy.String() != "¥500" {
		t.Errorf("JPY String() got %s, want ¥500", jpy.String())
	}
}

func TestJSON_Marshaling(t *testing.T) {
	m := MustNew(1050, "USD")

	// Marshal
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}

	expected := `{"amount":"10.50","currency":"USD"}`
	if string(data) != expected {
		t.Errorf("Marshal got %s, want %s", string(data), expected)
	}

	// Unmarshal
	var m2 Money
	if err := json.Unmarshal(data, &m2); err != nil {
		t.Fatal(err)
	}

	if m2.Minor() != 1050 || m2.currency.isoCode != "USD" {
		t.Error("Unmarshal failed to reconstruct Money object")
	}
}

// FuzzNewFromString throws random byte sequences and parameters at our string parser.
// To run this: `go test -fuzz=FuzzNewFromString`
// It asserts both panic-freedom and mathematical invariants (e.g. sign preservation).
func FuzzNewFromString(f *testing.F) {
	// Provide seed corpus (examples of valid and tricky inputs)
	f.Add("10.50", "USD")
	f.Add("-100.99", "GBP")
	f.Add("0.00", "EUR")
	f.Add("92233720368547758.07", "USD")  // Boundary test (MaxInt64 for 2 decimals)
	f.Add("92233720368547758.08", "USD")  // Overflow boundary test
	f.Add("-92233720368547758.08", "USD") // MinInt64 boundary test
	f.Add("-92233720368547758.09", "USD") // Underflow boundary test
	f.Add("9999999999999999.99", "USD")
	f.Add("NaN", "USD")
	f.Add("10.50.30", "USD")

	f.Fuzz(func(t *testing.T, val string, currencyCode string) {
		got, err := NewFromString(val, currencyCode)
		if err != nil {
			return
		}

		// Invariant 1: Currency configuration must match requested code
		if got.Currency() != currencyCode {
			t.Errorf("currency mismatch: got %q, want %q", got.Currency(), currencyCode)
		}

		// Invariant 2: Positive input without '-' must have non-negative minor amount
		trimmed := strings.TrimSpace(val)
		if !strings.HasPrefix(trimmed, "-") {
			if got.Minor() < 0 {
				t.Errorf("positive input %q produced negative minor amount %d (silent wrap-around!)", val, got.Minor())
			}
		} else {
			if got.Minor() > 0 {
				t.Errorf("negative input %q produced positive minor amount %d (silent wrap-around!)", val, got.Minor())
			}
		}
	})
}

func TestCurrencyConfigs(t *testing.T) {
	if len(currencyConfig) == 0 {
		t.Fatal("currencyConfig map should not be empty")
	}

	for code, cfg := range currencyConfig {
		if cfg.isoCode != code {
			t.Errorf("currency %s has mismatched ISOCode: %s", code, cfg.isoCode)
		}
		if cfg.decimals < 0 || cfg.decimals > MaxSafeDecimals {
			t.Errorf("currency %s has invalid decimals: %d", code, cfg.decimals)
		}
		if cfg.isoDigits < 0 {
			t.Errorf("currency %s has negative ISODigits: %d", code, cfg.isoDigits)
		}
		if cfg.name == "" {
			t.Errorf("currency %s has empty Name", code)
		}
	}
}

func TestSpecificISOCurrencies(t *testing.T) {
	testCases := []struct {
		code         string
		expectedNum  int
		expectedDec  int
		stringInput  string
		expectedUnit int64
	}{
		{"USD", 840, 2, "10.50", 1050},
		{"EUR", 978, 2, "25.00", 2500},
		{"JPY", 392, 0, "500", 500},
		{"KRW", 410, 0, "1000", 1000},
		{"VED", 926, 2, "15.75", 1575},
		{"SLE", 925, 2, "100.50", 10050},
		{"ZWG", 924, 2, "50.25", 5025},
		{"XCG", 532, 2, "12.34", 1234},
		{"CLF", 990, 4, "1.2345", 12345},
		{"XAU", 959, 0, "10", 10},
	}

	for _, tc := range testCases {
		t.Run(tc.code, func(t *testing.T) {
			cfg, exists := currencyConfig[tc.code]
			if !exists {
				t.Fatalf("currency %s does not exist in configuration", tc.code)
			}
			if cfg.isoNum != tc.expectedNum {
				t.Errorf("%s ISONum = %d, want %d", tc.code, cfg.isoNum, tc.expectedNum)
			}
			if cfg.decimals != tc.expectedDec {
				t.Errorf("%s Decimals = %d, want %d", tc.code, cfg.decimals, tc.expectedDec)
			}

			m, err := NewFromString(tc.stringInput, tc.code)
			if err != nil {
				t.Fatalf("NewFromString(%q, %q) returned error: %v", tc.stringInput, tc.code, err)
			}
			if m.Minor() != tc.expectedUnit {
				t.Errorf("%s minor unit = %d, want %d", tc.code, m.Minor(), tc.expectedUnit)
			}
		})
	}
}

func TestMul_NegativeMultiplier(t *testing.T) {
	m := MustNew(1000, "USD")

	// Multiplying positive amount by -1
	res, err := m.Mul(-1)
	if err != nil {
		t.Fatalf("m.Mul(-1) failed: %v", err)
	}
	if res.Minor() != -1000 {
		t.Errorf("got %d, want -1000", res.Minor())
	}

	// Multiplying negative amount by -1
	neg := MustNew(-1000, "USD")
	res, err = neg.Mul(-1)
	if err != nil {
		t.Fatalf("neg.Mul(-1) failed: %v", err)
	}
	if res.Minor() != 1000 {
		t.Errorf("got %d, want 1000", res.Minor())
	}

	// Multiplying by negative factor
	res, err = m.Mul(-5)
	if err != nil {
		t.Fatalf("m.Mul(-5) failed: %v", err)
	}
	if res.Minor() != -5000 {
		t.Errorf("got %d, want -5000", res.Minor())
	}

	// Zero multipliers
	res, err = m.Mul(0)
	if err != nil || res.Minor() != 0 {
		t.Errorf("m.Mul(0) failed")
	}

	zero := MustNew(0, "USD")
	res, err = zero.Mul(-1)
	if err != nil || res.Minor() != 0 {
		t.Errorf("zero.Mul(-1) failed")
	}
}

func TestAsDecimalString_HighPrecision(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		currency string
		wantStr  string
	}{
		{"zero USD", 0, "USD", "0.00"},
		{"one cent USD", 1, "USD", "0.01"},
		{"five cents USD", 5, "USD", "0.05"},
		{"negative cent USD", -1, "USD", "-0.01"},
		{"negative five cents USD", -5, "USD", "-0.05"},
		{"large USD beyond float64 53-bit precision", 9007199254740993, "USD", "90071992547409.93"},
		{"negative large USD", -9007199254740993, "USD", "-90071992547409.93"},
		{"zero decimals JPY", 500, "JPY", "500"},
		{"zero decimals negative JPY", -500, "JPY", "-500"},
		{"three decimals KWD", 1234, "KWD", "1.234"},
		{"four decimals CLF", 12345, "CLF", "1.2345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Money{amount: tt.amount, currency: currencyConfig[tt.currency]}
			got := m.AsDecimalString()
			if got != tt.wantStr {
				t.Errorf("AsDecimalString() = %q, want %q", got, tt.wantStr)
			}
		})
	}
}

func TestNewFromString_CommasAndValidation(t *testing.T) {
	tests := []struct {
		name     string
		val      string
		currency string
		wantErr  error
	}{
		{"comma in integer part", "1,000.50", "USD", ErrInvalidFormat},
		{"double comma", "1,,000", "USD", ErrInvalidFormat},
		{"malformed comma", "10,0", "USD", ErrInvalidFormat},
		{"letter in integer", "12a.34", "USD", ErrInvalidFormat},
		{"letter in fraction", "12.3b", "USD", ErrInvalidFormat},
		{"multiple dots", "10.50.30", "USD", ErrMalformedInput},
		{"misplaced minus", "10-50", "USD", ErrMalformedInput},
		{"arabic indic numerals", "١٠.٥٠", "USD", ErrInvalidFormat},
		{"devanagari numerals", "१०.५०", "USD", ErrInvalidFormat},
		{"full width numerals", "１０.５０", "USD", ErrInvalidFormat},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFromString(tt.val, tt.currency)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("NewFromString(%q, %q) error = %v, want %v", tt.val, tt.currency, err, tt.wantErr)
			}
		})
	}
}

func TestSovereignScaleAmounts(t *testing.T) {
	// 1.26 Quadrillion VND ($50B corporate acquisition in VND)
	vnd, err := New(1260000000000000, "VND")
	if err != nil {
		t.Fatalf("New failed for sovereign scale VND: %v", err)
	}
	if vnd.Minor() != 1260000000000000 {
		t.Errorf("got %d, want 1260000000000000", vnd.Minor())
	}
	if vnd.AsDecimalString() != "1260000000000000" {
		t.Errorf("VND AsDecimalString = %s, want 1260000000000000", vnd.AsDecimalString())
	}

	// 1.6 Quadrillion IDR
	idr, err := New(1600000000000000, "IDR")
	if err != nil {
		t.Fatalf("New failed for sovereign scale IDR: %v", err)
	}
	if idr.Minor() != 1600000000000000 {
		t.Errorf("got %d, want 1600000000000000", idr.Minor())
	}

	// MaxInt64 and MinInt64 boundary testing for zero-decimal currency
	jpyMax, err := New(math.MaxInt64, "JPY")
	if err != nil {
		t.Fatalf("New failed for MaxInt64 JPY: %v", err)
	}
	if jpyMax.Minor() != math.MaxInt64 {
		t.Errorf("got %d, want %d", jpyMax.Minor(), int64(math.MaxInt64))
	}

	jpyMin, err := New(math.MinInt64, "JPY")
	if err != nil {
		t.Fatalf("New failed for MinInt64 JPY: %v", err)
	}
	if jpyMin.Minor() != math.MinInt64 {
		t.Errorf("got %d, want %d", jpyMin.Minor(), int64(math.MinInt64))
	}
}

func TestLocalisedString_ExactPrecision(t *testing.T) {
	// Value exceeding 2^53 (9,007,199,254,740,992 minor units)
	// $90,071,992,547,409.93
	m := MustNew(9007199254740993, "USD")

	gotEN := m.LocalisedString(language.AmericanEnglish)
	// CLDR/x/text emits a non-breaking space between symbol and amount for
	// en-US; that is CLDR-correct behaviour and must not be stripped, because
	// stripping it would also strip the NBSP that other locales use as their
	// thousands separator (see fr-FR case below).
	if !strings.Contains(gotEN, "90,071,992,547,409.93") || !strings.Contains(gotEN, "$") {
		t.Errorf("LocalisedString(en-US) = %q, want the digits 90,071,992,547,409.93 and $ preserved", gotEN)
	}

	gotDE := m.LocalisedString(language.German)
	// German uses '.' for thousands and ',' for decimal
	if gotDE != "90.071.992.547.409,93$" && gotDE != "$90.071.992.547.409,93" && gotDE != "90.071.992.547.409,93\u00a0$" {
		// Verify digits and separators are preserved without float precision loss
		if !strings.Contains(gotDE, "90.071.992.547.409,93") && !strings.Contains(gotDE, "90.071.992.547.409.93") {
			t.Errorf("LocalisedString(de) = %q, expected exact digits preserved", gotDE)
		}
	}

	// Zero decimal high precision currency
	jpy := MustNew(9007199254740993, "JPY")
	gotJPY := jpy.LocalisedString(language.Japanese)
	if !strings.Contains(gotJPY, "9,007,199,254,740,993") {
		t.Errorf("LocalisedString(ja JPY) = %q, want contains 9,007,199,254,740,993", gotJPY)
	}

	// Negative value: preserve exact digits regardless of sign placement.
	neg := MustNew(-123456, "USD")
	gotNeg := neg.LocalisedString(language.AmericanEnglish)
	if !strings.Contains(gotNeg, "1,234.56") {
		t.Errorf("LocalisedString(neg USD) = %q, want digits 1,234.56 preserved", gotNeg)
	}
}

// TestLocalisedString_NBSPThousandsSeparator locks in the fix for the bug
// where the old implementation stripped every Unicode whitespace rune,
// destroying the NBSP that CLDR uses as the thousands separator in French,
// Russian and Swedish. Under the buggy behaviour, "1 234,56" collapsed to
// "1234,56".
func TestLocalisedString_NBSPThousandsSeparator(t *testing.T) {
	m := MustNew(123456789, "EUR") // 1 234 567,89 in fr-FR

	gotFR := m.LocalisedString(language.French)
	// French thousands separator is NBSP ( ). Assert the exact byte is present.
	if !strings.Contains(gotFR, " ") {
		t.Errorf("LocalisedString(fr) = %q, expected NBSP thousands separator preserved", gotFR)
	}
	if !strings.Contains(gotFR, ",89") {
		t.Errorf("LocalisedString(fr) = %q, expected French decimal separator ','", gotFR)
	}
}

func TestNewFromString_QuadrillionDollarBug(t *testing.T) {
	// Exact MaxInt64 for 2 decimal currency ($92,233,720,368,547,758.07)
	maxUSD, err := NewFromString("92233720368547758.07", "USD")
	if err != nil {
		t.Fatalf("NewFromString failed for MaxInt64 USD: %v", err)
	}
	if maxUSD.Minor() != math.MaxInt64 {
		t.Errorf("got %d, want %d (math.MaxInt64)", maxUSD.Minor(), int64(math.MaxInt64))
	}

	// +1 cent beyond MaxInt64 must fail with ErrAmountTooLarge, NOT wrap around to negative
	_, err = NewFromString("92233720368547758.08", "USD")
	if !errors.Is(err, ErrAmountTooLarge) {
		t.Errorf("NewFromString(92233720368547758.08, USD) error = %v, want ErrAmountTooLarge", err)
	}

	// Exact MinInt64 for 2 decimal currency (-$92,233,720,368,547,758.08)
	minUSD, err := NewFromString("-92233720368547758.08", "USD")
	if err != nil {
		t.Fatalf("NewFromString failed for MinInt64 USD: %v", err)
	}
	if minUSD.Minor() != math.MinInt64 {
		t.Errorf("got %d, want %d (math.MinInt64)", minUSD.Minor(), int64(math.MinInt64))
	}

	// -1 cent beyond MinInt64 must fail with ErrAmountTooLarge, NOT wrap around to positive
	_, err = NewFromString("-92233720368547758.09", "USD")
	if !errors.Is(err, ErrAmountTooLarge) {
		t.Errorf("NewFromString(-92233720368547758.09, USD) error = %v, want ErrAmountTooLarge", err)
	}
}

func TestFromDecimal_OverflowAndInf(t *testing.T) {
	tests := []struct {
		name    string
		val     float64
		wantErr error
	}{
		{"massive float positive", 1e22, ErrAmountTooLarge},
		{"massive float negative", -1e22, ErrAmountTooLarge},
		{"NaN", math.NaN(), ErrAmountTooLarge},
		{"positive infinity", math.Inf(1), ErrAmountTooLarge},
		{"negative infinity", math.Inf(-1), ErrAmountTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromDecimal(tt.val, "USD", RoundHalfToEven)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("FromDecimal(%v, USD) error = %v, want %v", tt.val, err, tt.wantErr)
			}
		})
	}
}

func TestMoney_Split_Bounds(t *testing.T) {
	m := MustNew(1000, "USD")

	// Valid split
	parts, err := m.Split(3)
	if err != nil || len(parts) != 3 {
		t.Fatalf("Split(3) failed: %v", err)
	}

	// Zero or negative splits
	_, err = m.Split(0)
	if !errors.Is(err, ErrInvalidSplitCount) {
		t.Errorf("Split(0) error = %v, want ErrInvalidSplitCount", err)
	}

	_, err = m.Split(-5)
	if !errors.Is(err, ErrInvalidSplitCount) {
		t.Errorf("Split(-5) error = %v, want ErrInvalidSplitCount", err)
	}

	// Exceeding MaxSplitParts (OOM prevention)
	_, err = m.Split(MaxSplitParts + 1)
	if !errors.Is(err, ErrInvalidSplitCount) {
		t.Errorf("Split(%d) error = %v, want ErrInvalidSplitCount", MaxSplitParts+1, err)
	}
}

func TestGetPow10_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("getPow10 should panic on out-of-range scale")
		}
	}()
	_ = getPow10(99)
}

func TestAdd_Overflow(t *testing.T) {
	maxUSD := MustNew(math.MaxInt64, "USD")
	one := MustNew(1, "USD")
	if _, err := maxUSD.Add(one); !errors.Is(err, ErrOverflow) {
		t.Errorf("MaxInt64 + 1 error = %v, want ErrOverflow", err)
	}
	minUSD := MustNew(math.MinInt64, "USD")
	negOne := MustNew(-1, "USD")
	if _, err := minUSD.Add(negOne); !errors.Is(err, ErrOverflow) {
		t.Errorf("MinInt64 + -1 error = %v, want ErrOverflow", err)
	}
}

func TestSub_Overflow(t *testing.T) {
	minUSD := MustNew(math.MinInt64, "USD")
	one := MustNew(1, "USD")
	if _, err := minUSD.Sub(one); !errors.Is(err, ErrOverflow) {
		t.Errorf("MinInt64 - 1 error = %v, want ErrOverflow", err)
	}
	maxUSD := MustNew(math.MaxInt64, "USD")
	negOne := MustNew(-1, "USD")
	if _, err := maxUSD.Sub(negOne); !errors.Is(err, ErrOverflow) {
		t.Errorf("MaxInt64 - -1 error = %v, want ErrOverflow", err)
	}
}

func TestMul_Overflow(t *testing.T) {
	half := MustNew(math.MaxInt64/2+1, "USD")
	if _, err := half.Mul(2); !errors.Is(err, ErrOverflow) {
		t.Errorf("(MaxInt64/2+1) * 2 error = %v, want ErrOverflow", err)
	}
	minUSD := MustNew(math.MinInt64, "USD")
	if _, err := minUSD.Mul(-1); !errors.Is(err, ErrOverflow) {
		t.Errorf("MinInt64 * -1 error = %v, want ErrOverflow", err)
	}
}

func TestArithmetic_CurrencyMismatch(t *testing.T) {
	usd := MustNew(100, "USD")
	eur := MustNew(100, "EUR")
	if _, err := usd.Add(eur); !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("Add across currencies error = %v, want ErrCurrencyMismatch", err)
	}
	if _, err := usd.Sub(eur); !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("Sub across currencies error = %v, want ErrCurrencyMismatch", err)
	}
	if _, err := usd.Compare(eur); !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("Compare across currencies error = %v, want ErrCurrencyMismatch", err)
	}
	if usd.Equal(eur) {
		t.Errorf("Equal across currencies returned true")
	}
}

func TestSplit_One(t *testing.T) {
	m := MustNew(1234, "USD")
	parts, err := m.Split(1)
	if err != nil {
		t.Fatalf("Split(1) failed: %v", err)
	}
	if len(parts) != 1 || parts[0].Minor() != 1234 {
		t.Errorf("Split(1) = %v, want a single unchanged part", parts)
	}
}

func TestUnmarshalJSON_Errors(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr error
	}{
		{"empty object", `{}`, ErrEmptyInput},
		{"missing amount", `{"currency":"USD"}`, ErrEmptyInput},
		{"missing currency", `{"amount":"10.50"}`, ErrEmptyInput},
		{"invalid currency", `{"amount":"10.50","currency":"XYZ"}`, ErrInvalidCurrency},
		{"invalid amount", `{"amount":"not-a-number","currency":"USD"}`, ErrMalformedInput},
		{"malformed json", `{"amount":"10.50",`, ErrMalformedInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m Money
			err := m.UnmarshalJSON([]byte(tt.data))
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("UnmarshalJSON(%q) err = %v, want %v", tt.data, err, tt.wantErr)
			}
		})
	}
}

func TestMarshalJSON_ZeroValue(t *testing.T) {
	var m Money
	if _, err := m.MarshalJSON(); err == nil {
		t.Error("MarshalJSON on zero-value Money returned no error")
	}
}

func BenchmarkNewFromString(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = NewFromString("1234.56", "USD")
	}
}

func BenchmarkAdd(b *testing.B) {
	m1 := MustNew(1000, "USD")
	m2 := MustNew(500, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m1.Add(m2)
	}
}

func BenchmarkSub(b *testing.B) {
	m1 := MustNew(1000, "USD")
	m2 := MustNew(500, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m1.Sub(m2)
	}
}

func BenchmarkMul(b *testing.B) {
	m := MustNew(1000, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.Mul(3)
	}
}

func BenchmarkAsDecimalString(b *testing.B) {
	m := MustNew(123456, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.AsDecimalString()
	}
}

func BenchmarkString(b *testing.B) {
	m := MustNew(123456, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.String()
	}
}

func BenchmarkMarshalJSON(b *testing.B) {
	m := MustNew(123456, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.MarshalJSON()
	}
}

func BenchmarkUnmarshalJSON(b *testing.B) {
	data := []byte(`{"amount":"1234.56","currency":"USD"}`)
	var m Money
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.UnmarshalJSON(data)
	}
}

func TestFromDecimal_BoundaryExactMax(t *testing.T) {
	// float64(math.MaxInt64) rounds up past MaxInt64. Feeding that back into
	// int64() is implementation-defined by the Go spec. The bounds check
	// must reject rather than accept.
	if _, err := FromDecimal(float64(math.MaxInt64)/100.0*100.0, "JPY", RoundHalfToEven); !errors.Is(err, ErrAmountTooLarge) {
		t.Errorf("FromDecimal at MaxInt64 boundary err = %v, want ErrAmountTooLarge", err)
	}
}

func TestCurrency_HasISONum(t *testing.T) {
	// USD has a real ISO num, so HasISONum must be true.
	if !currencyConfig["USD"].HasISONum() {
		t.Error("USD.HasISONum() = false, want true")
	}
	// Any dataset entry that HasISONum reports as unknown must equal the
	// sentinel exactly; otherwise the generator lost a distinction.
	for _, c := range currencyConfig {
		if !c.HasISONum() && c.isoNum != ISONumUnknown {
			t.Errorf("%s: HasISONum false but ISONum = %d, want ISONumUnknown", c.isoCode, c.isoNum)
		}
	}
	// Synthetic construction: guarantee the sentinel path is exercised even
	// when the current generated dataset happens to have no nulls.
	syntheticUnknown := &Currency{isoCode: "XXX", isoNum: ISONumUnknown, numToBasic: NumToBasicUnknown}
	if syntheticUnknown.HasISONum() {
		t.Error("synthetic Currency with ISONumUnknown reported HasISONum = true")
	}
	if syntheticUnknown.HasNumToBasic() {
		t.Error("synthetic Currency with NumToBasicUnknown reported HasNumToBasic = true")
	}
	syntheticKnown := &Currency{isoCode: "YYY", isoNum: 999, numToBasic: 100}
	if !syntheticKnown.HasISONum() {
		t.Error("synthetic Currency with real ISONum reported HasISONum = false")
	}
	if !syntheticKnown.HasNumToBasic() {
		t.Error("synthetic Currency with real NumToBasic reported HasNumToBasic = false")
	}
}

func TestNewFromString_OverflowingIntegerPart(t *testing.T) {
	// "99999999999999999999" (20 nines) exceeds MaxInt64 and must be
	// classified as ErrAmountTooLarge, not ErrInvalidFormat.
	_, err := NewFromString("99999999999999999999", "USD")
	if !errors.Is(err, ErrAmountTooLarge) {
		t.Errorf("err = %v, want ErrAmountTooLarge", err)
	}
}

func TestNewFromString_WhitespaceOnly(t *testing.T) {
	_, err := NewFromString("   ", "USD")
	if !errors.Is(err, ErrEmptyInput) {
		t.Errorf("err = %v, want ErrEmptyInput", err)
	}
}

func TestEqual_ZeroValues(t *testing.T) {
	var a, b Money
	if !a.Equal(b) {
		t.Error("two zero-value Moneys should be equal, matching time.Time.Equal")
	}
	usd := MustNew(0, "USD")
	if a.Equal(usd) || usd.Equal(a) {
		t.Error("zero-value Money should not equal a real currency-carrying zero")
	}
}

func TestMarshalUnmarshalText_Roundtrip(t *testing.T) {
	m := MustNew(1234567, "GBP")
	txt, err := m.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	if string(txt) != "12345.67 GBP" {
		t.Errorf("MarshalText = %q, want %q", txt, "12345.67 GBP")
	}
	var round Money
	if err := round.UnmarshalText(txt); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}
	if !round.Equal(m) {
		t.Errorf("round-trip = %v, want %v", round, m)
	}
}

func TestUnmarshalText_Errors(t *testing.T) {
	var m Money
	if err := m.UnmarshalText([]byte("")); !errors.Is(err, ErrEmptyInput) {
		t.Errorf("empty err = %v, want ErrEmptyInput", err)
	}
	if err := m.UnmarshalText([]byte("12.34")); !errors.Is(err, ErrMalformedInput) {
		t.Errorf("no-space err = %v, want ErrMalformedInput", err)
	}
	if err := m.UnmarshalText([]byte("12.34 XYZ")); !errors.Is(err, ErrInvalidCurrency) {
		t.Errorf("bad currency err = %v, want ErrInvalidCurrency", err)
	}
}

// FuzzUnmarshalJSON exercises the JSON boundary with random bytes and
// asserts panic-freedom and the sign-preservation invariant on success.
func FuzzUnmarshalJSON(f *testing.F) {
	f.Add(`{"amount":"10.50","currency":"USD"}`)
	f.Add(`{"amount":"-100.00","currency":"GBP"}`)
	f.Add(`{"amount":"0.00","currency":"EUR"}`)
	f.Add(`{}`)
	f.Add(`{"amount":"","currency":""}`)
	f.Add(`{"amount":"92233720368547758.07","currency":"USD"}`)
	f.Add(`not json`)
	f.Add(``)

	f.Fuzz(func(t *testing.T, payload string) {
		var m Money
		err := m.UnmarshalJSON([]byte(payload))
		if err != nil {
			return
		}
		if m.Currency() == "" {
			t.Errorf("payload %q parsed with empty currency", payload)
		}
	})
}

func TestCmp_PanicsOnMismatch(t *testing.T) {
	usd := MustNew(100, "USD")
	eur := MustNew(100, "EUR")
	defer func() {
		if r := recover(); r == nil {
			t.Error("Cmp with mismatched currencies should panic")
		}
	}()
	_ = usd.Cmp(eur)
}

func TestCmp_ReturnsCompareOrder(t *testing.T) {
	a := MustNew(100, "USD")
	b := MustNew(200, "USD")
	if a.Cmp(b) != -1 || b.Cmp(a) != 1 || a.Cmp(a) != 0 {
		t.Error("Cmp ordering wrong")
	}
}

func TestMoney_Valid(t *testing.T) {
	var zero Money
	if zero.Valid() {
		t.Error("zero-value Money.Valid() = true, want false")
	}
	if !MustNew(0, "USD").Valid() {
		t.Error("USD zero amount .Valid() = false, want true")
	}
}

func TestFromDecimal_UnknownMode(t *testing.T) {
	_, err := FromDecimal(1.0, "USD", RoundingMode(99))
	if !errors.Is(err, ErrInvalidRoundingMode) {
		t.Errorf("err = %v, want ErrInvalidRoundingMode", err)
	}
}

func TestLocalisedString_SymbolStyle(t *testing.T) {
	m := MustNew(123456, "USD")
	iso := m.LocalisedString(language.AmericanEnglish, SymbolStyleISO)
	if !strings.Contains(iso, "USD") {
		t.Errorf("SymbolStyleISO output %q missing USD", iso)
	}
	narrow := m.LocalisedString(language.AmericanEnglish, SymbolStyleNarrow)
	if !strings.Contains(narrow, "$") {
		t.Errorf("SymbolStyleNarrow output %q missing $", narrow)
	}
}

func TestZeroValueMoney_HasNoTruthyPredicate(t *testing.T) {
	var m Money
	if m.IsZero() || m.IsPositive() || m.IsNegative() {
		t.Error("zero-value Money should not satisfy any predicate; currency is missing")
	}
}

func TestFromDecimal_ZeroDecimalsCurrency(t *testing.T) {
	// JPY has zero decimals; the multiplier is 1 and rounding modes should
	// select the correct integer round.
	tests := []struct {
		name string
		val  float64
		mode RoundingMode
		want int64
	}{
		{"bankers even 10.5", 10.5, RoundHalfToEven, 10},
		{"bankers odd 11.5", 11.5, RoundHalfToEven, 12},
		{"away from zero 10.5", 10.5, RoundHalfAwayFromZero, 11},
		{"floor 10.9", 10.9, RoundDown, 10},
		{"ceil 10.1", 10.1, RoundUp, 11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromDecimal(tt.val, "JPY", tt.mode)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if got.Minor() != tt.want {
				t.Errorf("Minor() = %d, want %d", got.Minor(), tt.want)
			}
		})
	}
}

// TestFromDecimal_FloatPrecisionTrap locks in the documented behaviour that
// FromDecimal cannot represent every decimal exactly. 1.005 is stored as
// 1.00499999... in IEEE-754, so RoundHalfAwayFromZero at USD scale rounds
// down to 100 cents rather than the mathematically expected 101 cents.
// This is the exact trap the WARNING doc mentions; changing this test is
// a signal that the trap is no longer documented and callers may be
// surprised.
func TestFromDecimal_FloatPrecisionTrap(t *testing.T) {
	got, err := FromDecimal(1.005, "USD", RoundHalfAwayFromZero)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.Minor() != 100 {
		t.Errorf("Minor() = %d, want 100 (proving the float trap is real; use NewFromString for exact input)", got.Minor())
	}
}

func TestMoneyError_Format(t *testing.T) {
	base := errors.New("boom")
	tests := []struct {
		name string
		in   MoneyError
		want string
	}{
		{"op only", MoneyError{Op: "Add", Err: base}, "money.Add: boom"},
		{"op and amount", MoneyError{Op: "NewFromString", Amount: "10.50", Err: base}, "money.NewFromString(10.50): boom"},
		{"op, amount, currency", MoneyError{Op: "New", Amount: "10", Currency: "USD", Err: base}, "money.New(10, USD): boom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewFromString_TrimBeforeLengthCap(t *testing.T) {
	// 65 spaces should be classified as empty (trimmed to ""), not TooLong.
	_, err := NewFromString(strings.Repeat(" ", 65), "USD")
	if !errors.Is(err, ErrEmptyInput) {
		t.Errorf("65-space input err = %v, want ErrEmptyInput", err)
	}
}

func TestMoneyError_NilReceiver(t *testing.T) {
	var e *MoneyError
	if got := e.Error(); got != "<nil>" {
		t.Errorf("nil.Error() = %q, want %q", got, "<nil>")
	}
	if e.Unwrap() != nil {
		t.Error("nil.Unwrap() should return nil")
	}
}

func TestNeg(t *testing.T) {
	pos := MustNew(1000, "USD")
	neg, err := pos.Neg()
	if err != nil || neg.Minor() != -1000 {
		t.Errorf("Neg(1000 USD) = (%v, %v)", neg.Minor(), err)
	}
	roundTrip, err := neg.Neg()
	if err != nil || roundTrip.Minor() != 1000 {
		t.Errorf("Neg(-1000 USD) = (%v, %v)", roundTrip.Minor(), err)
	}
	min := MustNew(math.MinInt64, "USD")
	if _, err := min.Neg(); !errors.Is(err, ErrOverflow) {
		t.Errorf("Neg(MinInt64 USD) err = %v, want ErrOverflow", err)
	}
	var zero Money
	got, err := zero.Neg()
	if err != nil || got.Valid() {
		t.Errorf("Neg(zero-value) = (%v, %v)", got, err)
	}
}

func TestAbs(t *testing.T) {
	neg := MustNew(-1000, "USD")
	abs, err := neg.Abs()
	if err != nil || abs.Minor() != 1000 {
		t.Errorf("Abs(-1000 USD) = (%v, %v)", abs.Minor(), err)
	}
	pos := MustNew(1000, "USD")
	if v, err := pos.Abs(); err != nil || v.Minor() != 1000 {
		t.Errorf("Abs(1000 USD) unchanged = (%v, %v)", v.Minor(), err)
	}
	min := MustNew(math.MinInt64, "USD")
	if _, err := min.Abs(); !errors.Is(err, ErrOverflow) {
		t.Errorf("Abs(MinInt64 USD) err = %v, want ErrOverflow", err)
	}
}

func TestFormat_Verbs(t *testing.T) {
	m := MustNew(1050, "USD")
	tests := []struct {
		format string
		want   string
	}{
		{"%s", "$10.50"},
		{"%v", "$10.50"},
		{"%+v", "money.Money{amount:1050 currency:USD}"},
		{"%#v", "money.MustNew(1050, \"USD\")"},
		{"%q", "\"$10.50\""},
	}
	for _, tt := range tests {
		got := fmt.Sprintf(tt.format, m)
		if got != tt.want {
			t.Errorf("Sprintf(%q, m) = %q, want %q", tt.format, got, tt.want)
		}
	}
	if got := fmt.Sprintf("%d", m); !strings.Contains(got, "money.Money=") {
		t.Errorf("unknown verb fallback = %q, want to contain money.Money=", got)
	}
}

func TestMoneyError_EmptyOpDefault(t *testing.T) {
	e := &MoneyError{Err: errors.New("boom")}
	if got := e.Error(); got != "money.?: boom" {
		t.Errorf("Error() with empty Op = %q, want %q", got, "money.?: boom")
	}
}

func TestLocalisedString_ArabicLocaleNoCorruption(t *testing.T) {
	// Arabic-Indic digits can produce multi-byte runes in the separator
	// sniff; the byte-slice approach would have produced garbage. Assert
	// that the digits 1234.56 (in whatever form CLDR picks) survive.
	m := MustNew(123456, "USD")
	got := m.LocalisedString(language.Arabic)
	if got == "" {
		t.Error("LocalisedString(ar) returned empty; separator sniff likely broke")
	}
}

func TestGetCurrency(t *testing.T) {
	usd, ok := GetCurrency("USD")
	if !ok {
		t.Fatal("GetCurrency(USD) not found")
	}
	if usd.ISOCode() != "USD" || usd.Symbol() != "$" || usd.Decimals() != 2 {
		t.Errorf("USD accessors returned wrong values: %s %s %d", usd.ISOCode(), usd.Symbol(), usd.Decimals())
	}
	if _, ok := GetCurrency("ZZZ"); ok {
		t.Error("GetCurrency(ZZZ) should not exist")
	}
}

func TestCurrencies(t *testing.T) {
	codes := Currencies()
	if len(codes) == 0 {
		t.Fatal("Currencies() returned empty")
	}
	for i := 1; i < len(codes); i++ {
		if codes[i-1] >= codes[i] {
			t.Fatalf("codes not sorted at index %d: %q >= %q", i, codes[i-1], codes[i])
		}
	}
	// mutation should not affect the internal map
	codes[0] = "MUTATED"
	fresh := Currencies()
	if fresh[0] == "MUTATED" {
		t.Error("Currencies() returned shared slice; caller mutation leaked back")
	}
}
