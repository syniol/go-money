package money

import (
	"encoding/json"
	"errors"
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

	// Comparisons
	less, _ := m5.IsLessThan(m10)
	greater, _ := m10.IsGreaterThan(m5)
	equal, _ := m10.IsEqual(m10)
	if !less || !greater || !equal {
		t.Error("Comparison logic error")
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

	if m2.Minor() != 1050 || m2.currency.ISOCode != "USD" {
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
	f.Add("92233720368547758.07", "USD") // Boundary test (MaxInt64 for 2 decimals)
	f.Add("92233720368547758.08", "USD") // Overflow boundary test
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
		if cfg.ISOCode != code {
			t.Errorf("currency %s has mismatched ISOCode: %s", code, cfg.ISOCode)
		}
		if cfg.Decimals < 0 || cfg.Decimals > MaxSafeDecimals {
			t.Errorf("currency %s has invalid decimals: %d", code, cfg.Decimals)
		}
		if cfg.ISODigits < 0 {
			t.Errorf("currency %s has negative ISODigits: %d", code, cfg.ISODigits)
		}
		if cfg.Name == "" {
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
			if cfg.ISONum != tc.expectedNum {
				t.Errorf("%s ISONum = %d, want %d", tc.code, cfg.ISONum, tc.expectedNum)
			}
			if cfg.Decimals != tc.expectedDec {
				t.Errorf("%s Decimals = %d, want %d", tc.code, cfg.Decimals, tc.expectedDec)
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
	if gotEN != "$90,071,992,547,409.93" {
		t.Errorf("LocalisedString(en-US) = %q, want %q", gotEN, "$90,071,992,547,409.93")
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

	// Negative value
	neg := MustNew(-123456, "USD")
	gotNeg := neg.LocalisedString(language.AmericanEnglish)
	if gotNeg != "-$1,234.56" && gotNeg != "$-1,234.56" && gotNeg != "($1,234.56)" {
		t.Errorf("LocalisedString(neg USD) = %q", gotNeg)
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
