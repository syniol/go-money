// Package sqltest exercises Money.Value, Money.Scan and NullMoney against
// a real SQL driver (modernc.org/sqlite, pure Go, no CGO) so the codec is
// verified end to end without leaking a driver dependency into the root
// module. See ../sql.go for the implementation.
package sqltest

import (
	"database/sql"
	"errors"
	"testing"

	money "github.com/syniol/go-money"
	_ "modernc.org/sqlite"
)

func open(t *testing.T) *sql.DB {
	t.Helper()
	// Driver name "sqlite" is specific to modernc.org/sqlite. If you swap
	// in github.com/mattn/go-sqlite3 (CGO), the name is "sqlite3".
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE orders (id INTEGER PRIMARY KEY, total TEXT NOT NULL, refund TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func TestSQLite_MoneyRoundTrip(t *testing.T) {
	db := open(t)
	orig := money.MustNew(1050, "USD")

	if _, err := db.Exec(`INSERT INTO orders (id, total) VALUES (?, ?)`, 1, orig); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var got money.Money
	if err := db.QueryRow(`SELECT total FROM orders WHERE id = ?`, 1).Scan(&got); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !got.Equal(orig) {
		t.Errorf("round-trip = %v, want %v", got, orig)
	}
}

func TestSQLite_NullMoney_RoundTrip(t *testing.T) {
	db := open(t)

	// Row with NULL refund.
	orig := money.MustNew(2500, "GBP")
	nullRefund := money.NullMoney{}
	if _, err := db.Exec(`INSERT INTO orders (id, total, refund) VALUES (?, ?, ?)`, 1, orig, nullRefund); err != nil {
		t.Fatalf("insert null: %v", err)
	}

	// Row with a real refund. Avoid the identifier `real`, which shadows
	// the Go built-in that returns the real part of a complex number.
	filledRefund := money.NullMoney{Money: money.MustNew(500, "GBP"), Valid: true}
	if _, err := db.Exec(`INSERT INTO orders (id, total, refund) VALUES (?, ?, ?)`, 2, orig, filledRefund); err != nil {
		t.Fatalf("insert filled: %v", err)
	}

	var (
		total money.Money
		rf    money.NullMoney
	)
	if err := db.QueryRow(`SELECT total, refund FROM orders WHERE id = ?`, 1).Scan(&total, &rf); err != nil {
		t.Fatalf("scan null row: %v", err)
	}
	if !total.Equal(orig) {
		t.Errorf("total = %v, want %v", total, orig)
	}
	if rf.Valid {
		t.Errorf("refund.Valid = true, want false for NULL column")
	}

	if err := db.QueryRow(`SELECT total, refund FROM orders WHERE id = ?`, 2).Scan(&total, &rf); err != nil {
		t.Fatalf("scan filled row: %v", err)
	}
	if !rf.Valid || !rf.Money.Equal(filledRefund.Money) {
		t.Errorf("refund = %+v, want %+v", rf, filledRefund)
	}
}

func TestSQLite_MalformedColumn_Errors(t *testing.T) {
	db := open(t)
	if _, err := db.Exec(`INSERT INTO orders (id, total) VALUES (?, ?)`, 1, "not a money value"); err != nil {
		t.Fatalf("insert raw: %v", err)
	}
	var got money.Money
	err := db.QueryRow(`SELECT total FROM orders WHERE id = ?`, 1).Scan(&got)
	if err == nil {
		t.Fatalf("Scan returned nil on malformed column value")
	}
	// "not a money value" cuts on the first space to ("not", "a money
	// value"); the second half is not a valid ISO code, so we expect
	// ErrInvalidCurrency, not a generic parse error.
	if !errors.Is(err, money.ErrInvalidCurrency) {
		t.Errorf("Scan err = %v, want ErrInvalidCurrency", err)
	}
}
