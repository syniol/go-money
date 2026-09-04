package money

import (
	"errors"
	"testing"
)

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
