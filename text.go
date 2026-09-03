package money

import "strings"

// MarshalText implements encoding.TextMarshaler, emitting "<decimal> <ISO>"
// (e.g. "10.50 USD"). Text codecs (YAML, TOML, URL params, form encoding,
// flag.TextVar) route through this without needing JSON-specific glue.
func (m Money) MarshalText() ([]byte, error) {
	if m.currency == nil {
		return nil, &MoneyError{Op: "MarshalText", Err: ErrInvalidCurrency}
	}
	amount := m.AsDecimalString()
	iso := m.currency.ISOCode
	buf := make([]byte, 0, len(amount)+1+len(iso))
	buf = append(buf, amount...)
	buf = append(buf, ' ')
	buf = append(buf, iso...)
	return buf, nil
}

// UnmarshalText implements encoding.TextUnmarshaler for "<decimal> <ISO>"
// input. Leading and trailing ASCII whitespace is tolerated.
func (m *Money) UnmarshalText(data []byte) error {
	s := strings.TrimSpace(string(data))
	if s == "" {
		return &MoneyError{Op: "UnmarshalText", Err: ErrEmptyInput}
	}
	amount, iso, ok := strings.Cut(s, " ")
	if !ok {
		return &MoneyError{Op: "UnmarshalText", Amount: s, Err: ErrMalformedInput}
	}
	v, err := NewFromString(amount, iso)
	if err != nil {
		return err
	}
	*m = v
	return nil
}
