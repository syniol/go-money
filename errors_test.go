package money

import (
	"errors"
	"testing"
)

func TestMoneyError_Format(t *testing.T) {
	base := errors.New("boom")
	tests := []struct {
		name string
		in   MoneyError
		want string
	}{
		{"op only", MoneyError{Op: "Add", Err: base}, "money.Add: boom"},
		{"op and amount", MoneyError{Op: "NewFromString", Amount: "10.50", Err: base}, "money.NewFromString(10.50): boom"},
		{"op, amount, currency", MoneyError{Op: "New", Amount: "10", Currency: "USD", Err: base}, "money.New(10, USD): boom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMoneyError_NilReceiver(t *testing.T) {
	var e *MoneyError
	if got := e.Error(); got != "<nil>" {
		t.Errorf("nil.Error() = %q, want %q", got, "<nil>")
	}
	if e.Unwrap() != nil {
		t.Error("nil.Unwrap() should return nil")
	}
}

func TestMoneyError_EmptyOpDefault(t *testing.T) {
	e := &MoneyError{Err: errors.New("boom")}
	if got := e.Error(); got != "money.?: boom" {
		t.Errorf("Error() with empty Op = %q, want %q", got, "money.?: boom")
	}
}
