package money

import (
	"errors"
	"testing"
)

func TestNew_And_MustNew(t *testing.T) {
	t.Run("valid new", func(t *testing.T) {
		m, err := New(100, "USD")
		if err != nil || m.Minor() != 100 {
			t.Errorf("New failed: %v", err)
		}
	})

	t.Run("invalid currency", func(t *testing.T) {
		_, err := New(100, "INVALID")
		if !errors.Is(err, ErrInvalidCurrency) {
			t.Errorf("Expected ErrInvalidCurrency, got %v", err)
		}
	})

	t.Run("must new panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("MustNew should have panicked on invalid currency")
			}
		}()
		_ = MustNew(100, "INVALID")
	})
}

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
