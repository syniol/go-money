package money

import "testing"

func TestMoney_Valid(t *testing.T) {
	var zero Money
	if zero.Valid() {
		t.Error("zero-value Money.Valid() = true, want false")
	}
	if !MustNew(0, "USD").Valid() {
		t.Error("USD zero amount .Valid() = false, want true")
	}
}

func TestZeroValueMoney_HasNoTruthyPredicate(t *testing.T) {
	var m Money
	if m.IsZero() || m.IsPositive() || m.IsNegative() {
		t.Error("zero-value Money should not satisfy any predicate; currency is missing")
	}
}
