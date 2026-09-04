package money

import (
	"encoding/json"
	"errors"
	"testing"
)

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
