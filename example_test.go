package money_test

import (
	"fmt"
	"github.com/syniol/go-money"
)

func ExampleNew() {
	moneyExample, err := money.New(8881, "GBP")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(moneyExample.Float())
	fmt.Println(moneyExample.Minor())
	fmt.Println(moneyExample.String())

	// Output:
	// 88.81
	// 8881
	// £88.81
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

func ExampleMoney_IsLess() {
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

	fmt.Println(moneyExample.IsLess(moneyExample))
	fmt.Println(moneyExample.IsLess(moneyGreater))
	fmt.Println(moneyExample.IsLess(moneyLess))

	// Output:
	// false <nil>
	// true <nil>
	// false <nil>
}

func ExampleMoney_IsGreat() {
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

	fmt.Println(moneyExample.IsLess(moneyExample))
	fmt.Println(moneyExample.IsGreat(moneyGreater))
	fmt.Println(moneyExample.IsGreat(moneyLess))

	// Output:
	// false <nil>
	// false <nil>
	// true <nil>
}
