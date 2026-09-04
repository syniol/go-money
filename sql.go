package money

import (
	"database/sql/driver"
	"fmt"
)

// Value implements database/sql/driver.Valuer, emitting the Money as
// "<decimal> <ISO>" (e.g. "10.50 USD") into a TEXT or VARCHAR column.
// The wire format matches MarshalText, so a Money value stored via
// database/sql round-trips identically through JSON, YAML, TOML and
// any encoding/text-based codec.
//
// Recommended column type: TEXT, or VARCHAR(24). The 24-byte cap covers
// the widest legitimate amount ("-9223372036854775808.5807 CLF" style),
// with headroom for leading zeros.
//
// For numeric-typed columns (e.g. Postgres NUMERIC(precision, scale)),
// use a driver-specific adapter; a pgx codec is planned as a follow-up.
func (m Money) Value() (driver.Value, error) {
	if m.currency == nil {
		return nil, &MoneyError{Op: "Value", Err: ErrInvalidCurrency}
	}
	txt, err := m.MarshalText()
	if err != nil {
		return nil, err
	}
	return string(txt), nil
}

// Scan implements database/sql.Scanner for the "<decimal> <ISO>" wire
// format. Accepts either string or []byte from the driver. Rejects nil
// with ErrEmptyInput; use NullMoney for nullable columns.
func (m *Money) Scan(src any) error {
	if src == nil {
		return &MoneyError{Op: "Scan", Err: ErrEmptyInput}
	}
	var data []byte
	switch v := src.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return &MoneyError{Op: "Scan", Err: fmt.Errorf("%w: unsupported source type %T", ErrMalformedInput, src)}
	}
	return m.UnmarshalText(data)
}

// NullMoney represents a Money that may be NULL in a database column,
// mirroring the shape of database/sql.NullString and friends. Valid
// distinguishes a real zero-amount Money (Valid == true, Money is a
// legitimate zero for its currency) from a NULL column (Valid == false).
type NullMoney struct {
	Money Money
	Valid bool
}

// Scan implements database/sql.Scanner for NullMoney. A NULL source
// leaves Valid == false and Money as its zero value; any other input
// delegates to Money.Scan.
func (nm *NullMoney) Scan(src any) error {
	if src == nil {
		nm.Money = Money{}
		nm.Valid = false
		return nil
	}
	if err := nm.Money.Scan(src); err != nil {
		nm.Valid = false
		return err
	}
	nm.Valid = true
	return nil
}

// Value implements database/sql/driver.Valuer for NullMoney, returning
// nil (SQL NULL) when Valid is false and delegating to Money.Value
// otherwise.
func (nm NullMoney) Value() (driver.Value, error) {
	if !nm.Valid {
		return nil, nil
	}
	return nm.Money.Value()
}
