package money

import (
	"encoding/json"
	"errors"
	"testing"
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
// The goal is to ensure the parser NEVER panics, no matter what garbage the network sends it.
func FuzzNewFromString(f *testing.F) {
	// Provide seed corpus (examples of valid and tricky inputs)
	f.Add("10.50", "USD")
	f.Add("-100.99", "GBP")
	f.Add("0.00", "EUR")
	f.Add("9999999999999999.99", "USD") // Boundary test
	f.Add("NaN", "USD")
	f.Add("10.50.30", "USD")

	f.Fuzz(func(t *testing.T, val string, currencyCode string) {
		// We do not assert against specific errors here.
		// A fuzz test passes if the function returns an error gracefully WITHOUT panicking.
		_, err := NewFromString(val, currencyCode)

		// If you want to be extremely strict, you could assert that IF err == nil,
		// the output matches a secondary slower (but proven) implementation to check for logical bugs.
		_ = err
	})
}
