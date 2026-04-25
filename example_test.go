package money_test

import (
	"encoding/json"
	"fmt"

	"github.com/syniol/go-money"
	"golang.org/x/text/language"
)

func ExampleNew() {
	moneyExample, err := money.New(8881, "GBP")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(moneyExample.AsDecimalString())
	fmt.Println(moneyExample.Minor())
	fmt.Println(moneyExample.String())

	// Output:
	// 88.81
	// 8881
	// £88.81
}

func ExampleMustNew() {
	// MustNew is ideal for package-level constants or tests where
	// a failure to initialize represents a fatal programmer error.
	m := money.MustNew(10050, "USD")

	fmt.Println(m.String())

	// Output:
	// $100.50
}

func ExampleNewFromString() {
	m, err := money.NewFromString("123.45", "EUR")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(m.Minor())
	fmt.Println(m.String())

	// Output:
	// 12345
	// €123.45
}

func ExampleFromDecimal() {
	// Using Banker's Rounding (RoundHalfToEven) to reduce cumulative bias
	m, err := money.FromDecimal(10.506, "USD", money.RoundHalfToEven)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(m.String())

	// Output:
	// $10.51
}

func ExampleMoney_IsEqual() {
	moneyExample, err := money.New(8881, "GBP")
	if err != nil {
		fmt.Println(err)
	}

	moneyNotEqual, err := money.New(8882, "GBP")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(moneyExample.IsEqual(moneyExample))
	fmt.Println(moneyExample.IsEqual(moneyNotEqual))

	// Output:
	// true <nil>
	// false <nil>
}

func ExampleMoney_IsLessThan() {
	moneyExample, err := money.New(8881, "GBP")
	if err != nil {
		fmt.Println(err)
	}

	moneyGreater, err := money.New(8882, "GBP")
	if err != nil {
		fmt.Println(err)
	}

	moneyLess, err := money.New(8880, "GBP")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(moneyExample.IsLessThan(moneyExample))
	fmt.Println(moneyExample.IsLessThan(moneyGreater))
	fmt.Println(moneyExample.IsLessThan(moneyLess))

	// Output:
	// false <nil>
	// true <nil>
	// false <nil>
}

func ExampleMoney_IsGreaterThan() {
	moneyExample, err := money.New(8881, "GBP")
	if err != nil {
		fmt.Println(err)
	}

	moneyGreater, err := money.New(8882, "GBP")
	if err != nil {
		fmt.Println(err)
	}

	moneyLess, err := money.New(8880, "GBP")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(moneyExample.IsGreaterThan(moneyExample))
	fmt.Println(moneyExample.IsGreaterThan(moneyGreater))
	fmt.Println(moneyExample.IsGreaterThan(moneyLess))

	// Output:
	// false <nil>
	// false <nil>
	// true <nil>
}

func ExampleMoney_Compare() {
	m1 := money.MustNew(1000, "USD")
	m2 := money.MustNew(2000, "USD")
	m3 := money.MustNew(1000, "USD")

	cmp1, _ := m1.Compare(m2)
	cmp2, _ := m2.Compare(m1)
	cmp3, _ := m1.Compare(m3)

	fmt.Println(cmp1)
	fmt.Println(cmp2)
	fmt.Println(cmp3)

	// Output:
	// -1
	// 1
	// 0
}

func ExampleMoney_Add() {
	base := money.MustNew(1000, "USD") // $10.00
	tip := money.MustNew(250, "USD")   // $2.50

	total, err := base.Add(tip)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(total.String())

	// Output:
	// $12.50
}

func ExampleMoney_Sub() {
	wallet := money.MustNew(5000, "USD") // $50.00
	cost := money.MustNew(1550, "USD")   // $15.50

	balance, err := wallet.Sub(cost)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(balance.String())

	// Output:
	// $34.50
}

func ExampleMoney_Mul() {
	unitPrice := money.MustNew(1500, "USD") // $15.00
	quantity := int64(3)

	total, err := unitPrice.Mul(quantity)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(total.String())

	// Output:
	// $45.00
}

func ExampleMoney_Split() {
	total := money.MustNew(100, "USD") // $1.00

	// Splitting $1.00 among 3 parties perfectly distributes the remainder
	parts, err := total.Split(3)
	if err != nil {
		fmt.Println(err)
	}

	for _, part := range parts {
		fmt.Println(part.String())
	}

	// Output:
	// $0.34
	// $0.33
	// $0.33
}

func ExampleMoney_IsZero() {
	m1 := money.MustNew(0, "USD")
	m2 := money.MustNew(500, "USD")

	fmt.Println(m1.IsZero())
	fmt.Println(m2.IsZero())

	// Output:
	// true
	// false
}

func ExampleMoney_IsPositive() {
	m1 := money.MustNew(100, "USD")
	m2 := money.MustNew(-100, "USD")
	m3 := money.MustNew(0, "USD")

	fmt.Println(m1.IsPositive())
	fmt.Println(m2.IsPositive())
	fmt.Println(m3.IsPositive())

	// Output:
	// true
	// false
	// false
}

func ExampleMoney_IsNegative() {
	m1 := money.MustNew(100, "USD")
	m2 := money.MustNew(-100, "USD")
	m3 := money.MustNew(0, "USD")

	fmt.Println(m1.IsNegative())
	fmt.Println(m2.IsNegative())
	fmt.Println(m3.IsNegative())

	// Output:
	// false
	// true
	// false
}

func ExampleMoney_LocalisedString() {
	m := money.MustNew(123456, "USD") // $1,234.56

	// After our sanitization in money.go, this will consistently
	// return the symbol flush against the digits.
	fmt.Println(m.LocalisedString(language.AmericanEnglish))

	// Output:
	// $1,234.56
}

func ExampleMoney_MarshalJSON() {
	m := money.MustNew(1050, "USD")

	data, err := json.Marshal(m)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(string(data))

	// Output:
	// {"amount":"10.50","currency":"USD"}
}

func ExampleMoney_UnmarshalJSON() {
	payload := []byte(`{"amount":"15.99","currency":"GBP"}`)

	var m money.Money
	if err := json.Unmarshal(payload, &m); err != nil {
		fmt.Println(err)
	}

	fmt.Println(m.Minor())
	fmt.Println(m.String())

	// Output:
	// 1599
	// £15.99
}
