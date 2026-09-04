package money

import (
	"database/sql/driver"
	"errors"
	"testing"
)

func TestValue_ProducesTextWireFormat(t *testing.T) {
	m := MustNew(1050, "USD")
	v, err := m.Value()
	if err != nil {
		t.Fatalf("Value err = %v", err)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("Value returned %T, want string", v)
	}
	if s != "10.50 USD" {
		t.Errorf("Value = %q, want %q", s, "10.50 USD")
	}
}

func TestValue_ZeroValueReturnsErrInvalidCurrency(t *testing.T) {
	var m Money
	if _, err := m.Value(); !errors.Is(err, ErrInvalidCurrency) {
		t.Errorf("zero-value Value err = %v, want ErrInvalidCurrency", err)
	}
}

func TestScan_FromString(t *testing.T) {
	var m Money
	if err := m.Scan("15.99 GBP"); err != nil {
		t.Fatalf("Scan err = %v", err)
	}
	if m.Minor() != 1599 || m.Currency() != "GBP" {
		t.Errorf("Scan produced %v, want 1599 GBP", m)
	}
}

func TestScan_FromBytes(t *testing.T) {
	var m Money
	if err := m.Scan([]byte("42.00 EUR")); err != nil {
		t.Fatalf("Scan err = %v", err)
	}
	if m.Minor() != 4200 || m.Currency() != "EUR" {
		t.Errorf("Scan produced %v, want 4200 EUR", m)
	}
}

func TestScan_NilReturnsErrEmptyInput(t *testing.T) {
	var m Money
	if err := m.Scan(nil); !errors.Is(err, ErrEmptyInput) {
		t.Errorf("nil Scan err = %v, want ErrEmptyInput; use NullMoney for nullable columns", err)
	}
}

func TestScan_UnsupportedTypeReturnsErrMalformedInput(t *testing.T) {
	var m Money
	if err := m.Scan(42); !errors.Is(err, ErrMalformedInput) {
		t.Errorf("int Scan err = %v, want ErrMalformedInput", err)
	}
	if err := m.Scan(3.14); !errors.Is(err, ErrMalformedInput) {
		t.Errorf("float64 Scan err = %v, want ErrMalformedInput", err)
	}
	if err := m.Scan(true); !errors.Is(err, ErrMalformedInput) {
		t.Errorf("bool Scan err = %v, want ErrMalformedInput", err)
	}
}

func TestScan_MalformedTextReturnsError(t *testing.T) {
	var m Money
	if err := m.Scan("garbage"); err == nil {
		t.Error("garbage Scan returned nil error")
	}
	if err := m.Scan("10.50"); !errors.Is(err, ErrMalformedInput) {
		t.Errorf("no-space Scan err = %v, want ErrMalformedInput", err)
	}
}

func TestValueScan_RoundTrip(t *testing.T) {
	cases := []Money{
		MustNew(1050, "USD"),
		MustNew(-1050, "USD"),
		MustNew(0, "USD"),
		MustNew(500, "JPY"),
		MustNew(12345, "CLF"),
	}
	for _, orig := range cases {
		t.Run(orig.String(), func(t *testing.T) {
			v, err := orig.Value()
			if err != nil {
				t.Fatalf("Value err = %v", err)
			}
			var round Money
			if err := round.Scan(v); err != nil {
				t.Fatalf("Scan err = %v", err)
			}
			if !round.Equal(orig) {
				t.Errorf("round-trip = %v, want %v", round, orig)
			}
		})
	}
}

func TestNullMoney_ScanNil(t *testing.T) {
	var nm NullMoney
	nm.Valid = true
	nm.Money = MustNew(999, "USD")
	if err := nm.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) err = %v", err)
	}
	if nm.Valid {
		t.Error("NullMoney.Valid should be false after Scan(nil)")
	}
	if nm.Money.Valid() {
		t.Error("NullMoney.Money should be zero after Scan(nil)")
	}
}

func TestNullMoney_ScanValid(t *testing.T) {
	var nm NullMoney
	if err := nm.Scan("100.00 EUR"); err != nil {
		t.Fatalf("Scan err = %v", err)
	}
	if !nm.Valid {
		t.Error("NullMoney.Valid should be true after successful Scan")
	}
	if nm.Money.Minor() != 10000 || nm.Money.Currency() != "EUR" {
		t.Errorf("NullMoney.Money = %v, want 10000 EUR", nm.Money)
	}
}

func TestNullMoney_ScanInvalidClearsValid(t *testing.T) {
	nm := NullMoney{Money: MustNew(1, "USD"), Valid: true}
	if err := nm.Scan("garbage"); err == nil {
		t.Error("garbage Scan returned nil error")
	}
	if nm.Valid {
		t.Error("NullMoney.Valid should be false after failed Scan")
	}
}

func TestNullMoney_ValueRoundTrip(t *testing.T) {
	// Valid NullMoney round-trips through Value/Scan.
	nm := NullMoney{Money: MustNew(1234, "GBP"), Valid: true}
	v, err := nm.Value()
	if err != nil {
		t.Fatalf("Value err = %v", err)
	}
	var round NullMoney
	if err := round.Scan(v); err != nil {
		t.Fatalf("Scan err = %v", err)
	}
	if !round.Valid || !round.Money.Equal(nm.Money) {
		t.Errorf("round-trip = %+v, want %+v", round, nm)
	}

	// Invalid NullMoney returns SQL NULL from Value.
	nm = NullMoney{}
	v, err = nm.Value()
	if err != nil {
		t.Fatalf("Value err = %v", err)
	}
	if v != nil {
		t.Errorf("invalid NullMoney.Value = %v, want nil", v)
	}
}

// Compile-time assertions that Money and NullMoney satisfy the
// database/sql/driver interfaces.
var (
	_ driver.Valuer = Money{}
	_ driver.Valuer = NullMoney{}
	_               = (&Money{}).Scan
	_               = (&NullMoney{}).Scan
)
