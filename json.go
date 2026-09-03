package money

import (
	"encoding/json"
	"errors"
	"fmt"
)

// MarshalJSON emits {"amount":"<decimal>","currency":"<ISO>"} so JavaScript
// clients cannot silently lose precision by parsing the amount as a float.
// The output is assembled by append to keep the encoding reflection-free.
func (m Money) MarshalJSON() ([]byte, error) {
	if m.currency == nil {
		return nil, errors.New("cannot marshal money without currency configuration")
	}
	amount := m.AsDecimalString()
	iso := m.currency.ISOCode
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
		return fmt.Errorf("failed to unmarshal money JSON: %w", err)
	}
	if aux.Amount == "" || aux.Currency == "" {
		return errors.New("money JSON must contain both amount and currency fields")
	}
	v, err := NewFromString(aux.Amount, aux.Currency)
	if err != nil {
		return fmt.Errorf("failed to unmarshal money: %w", err)
	}
	*m = v
	return nil
}
