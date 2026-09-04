package money

import (
	"errors"
	"math"
	"strings"
	"testing"
)

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

func BenchmarkNewFromString(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = NewFromString("1234.56", "USD")
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

func TestNewFromString_TrimBeforeLengthCap(t *testing.T) {
	// 65 spaces should be classified as empty (trimmed to ""), not TooLong.
	_, err := NewFromString(strings.Repeat(" ", 65), "USD")
	if !errors.Is(err, ErrEmptyInput) {
		t.Errorf("65-space input err = %v, want ErrEmptyInput", err)
	}
}
