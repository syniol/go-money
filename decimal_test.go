package money

import (
	"errors"
	"math"
	"testing"
)

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

func TestFromDecimal_BoundaryExactMax(t *testing.T) {
	// float64(math.MaxInt64) rounds up past MaxInt64. Feeding that back into
	// int64() is implementation-defined by the Go spec. The bounds check
	// must reject rather than accept.
	if _, err := FromDecimal(float64(math.MaxInt64)/100.0*100.0, "JPY", RoundHalfToEven); !errors.Is(err, ErrAmountTooLarge) {
		t.Errorf("FromDecimal at MaxInt64 boundary err = %v, want ErrAmountTooLarge", err)
	}
}

func TestFromDecimal_UnknownMode(t *testing.T) {
	_, err := FromDecimal(1.0, "USD", RoundingMode(99))
	if !errors.Is(err, ErrInvalidRoundingMode) {
		t.Errorf("err = %v, want ErrInvalidRoundingMode", err)
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
