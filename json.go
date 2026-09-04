package money

import (
	"encoding/json"
	"fmt"
)

// MarshalJSON emits {"amount":"<decimal>","currency":"<ISO>"} so JavaScript
// clients cannot silently lose precision by parsing the amount as a float.
// The output is assembled by append to keep the encoding reflection-free.
func (m Money) MarshalJSON() ([]byte, error) {
	if m.currency == nil {
		return nil, &MoneyError{Op: "MarshalJSON", Err: ErrInvalidCurrency}
	}
	amount := m.AsDecimalString()
	iso := m.currency.isoCode
	buf := make([]byte, 0, len(amount)+len(iso)+26)
	buf = append(buf, `{"amount":"`...)
	buf = append(buf, amount...)
	buf = append(buf, `","currency":"`...)
	buf = append(buf, iso...)
	buf = append(buf, `"}`...)
	return buf, nil
}

// UnmarshalJSON accepts {"amount":"<decimal>","currency":"<ISO>"} and reuses
// NewFromString so parsing rules match on the wire and off it.
func (m *Money) UnmarshalJSON(data []byte) error {
	var aux struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		// Preserve the underlying json error so operators debugging bad
		// payloads see the actual parse failure, while callers can still
		// match ErrMalformedInput via errors.Is.
		return &MoneyError{Op: "UnmarshalJSON", Err: fmt.Errorf("%w: %w", ErrMalformedInput, err)}
	}
	if aux.Amount == "" || aux.Currency == "" {
		return &MoneyError{Op: "UnmarshalJSON", Amount: aux.Amount, Currency: aux.Currency, Err: ErrEmptyInput}
	}
	v, err := NewFromString(aux.Amount, aux.Currency)
	if err != nil {
		return err
	}
	*m = v
	return nil
}
