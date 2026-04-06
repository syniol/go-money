package money_test

import (
	"fmt"
	"github.com/syniol/go-money"
)

func ExampleNew() {
	moneyExample, err := money.New(88.1, "GBP")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(moneyExample.Value())
	fmt.Println(moneyExample.String())
	fmt.Println(moneyExample.Formatted())

	moneyArabicExample, err := money.New(88.1, "AED")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(moneyArabicExample.Formatted())

	// Output:
	// 88.1
	// 88.10
	// £88.10
	// sss
}

func ExampleMoney_IsEqual() {
	moneyExample, err := money.New(88.1, "GBP")
	if err != nil {
		fmt.Println(err)
	}

	assertEqual, _ := moneyExample.IsEqual(88.10)
	assertEqualGreater, _ := moneyExample.IsEqualGreater(88.11)

	fmt.Println(assertEqual)
	fmt.Println(assertEqualGreater)
	// Output:
	// true
	// false
}
