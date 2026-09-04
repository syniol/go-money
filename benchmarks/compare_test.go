// Package benchmarks compares github.com/syniol/go-money against three
// established Go money and money-adjacent libraries on the operations most
// commonly hit in a payment or ledger service.
//
// Run:
//
//	cd benchmarks && go test -bench=. -benchmem -run=^$ -benchtime=3s
//
// The libraries under test are pinned in this module's go.mod so benchmark
// numbers are reproducible against exact versions:
//
//   - github.com/syniol/go-money (this repo, replaced to ../)
//   - github.com/Rhymond/go-money
//   - github.com/bojanz/currency
//   - github.com/leekchan/accounting (formatting only)
//
// Notation: Bench<Op>_<Lib>. Every group of related benchmarks does the
// same semantic work (subject to each library's API shape).
package benchmarks

import (
	"encoding/json"
	"testing"

	rhymond "github.com/Rhymond/go-money"
	bojanz "github.com/bojanz/currency"
	"github.com/leekchan/accounting"
	gomoney "github.com/syniol/go-money"
)

// Section: Construct.

func BenchmarkNew_GoMoney(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = gomoney.New(1050, "USD")
	}
}

func BenchmarkNew_Rhymond(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = rhymond.New(1050, "USD")
	}
}

// bojanz has no int64 constructor; parse is the only way in. See
// BenchmarkParse_Bojanz below.

// Section: Parse from string.

func BenchmarkParse_GoMoney(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = gomoney.NewFromString("1234.56", "USD")
	}
}

func BenchmarkParse_Bojanz(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = bojanz.NewAmount("1234.56", "USD")
	}
}

// Rhymond has no string parser; NewFromFloat is the closest analogue and
// silently converts through float64 (documented as unsafe for exact input).
func BenchmarkParse_Rhymond_NewFromFloatUnsafe(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = rhymond.NewFromFloat(1234.56, "USD")
	}
}

// Section: Add.

func BenchmarkAdd_GoMoney(b *testing.B) {
	a, _ := gomoney.New(1000, "USD")
	c, _ := gomoney.New(500, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Add(c)
	}
}

func BenchmarkAdd_Rhymond(b *testing.B) {
	a := rhymond.New(1000, "USD")
	c := rhymond.New(500, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Add(c)
	}
}

func BenchmarkAdd_Bojanz(b *testing.B) {
	a, _ := bojanz.NewAmount("10.00", "USD")
	c, _ := bojanz.NewAmount("5.00", "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Add(c)
	}
}

// Section: Sub.

func BenchmarkSub_GoMoney(b *testing.B) {
	a, _ := gomoney.New(1000, "USD")
	c, _ := gomoney.New(500, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Sub(c)
	}
}

func BenchmarkSub_Rhymond(b *testing.B) {
	a := rhymond.New(1000, "USD")
	c := rhymond.New(500, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Subtract(c)
	}
}

func BenchmarkSub_Bojanz(b *testing.B) {
	a, _ := bojanz.NewAmount("10.00", "USD")
	c, _ := bojanz.NewAmount("5.00", "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Sub(c)
	}
}

// Section: Mul.

func BenchmarkMul_GoMoney(b *testing.B) {
	a, _ := gomoney.New(1000, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Mul(3)
	}
}

func BenchmarkMul_Rhymond(b *testing.B) {
	a := rhymond.New(1000, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = a.Multiply(3)
	}
}

func BenchmarkMul_Bojanz(b *testing.B) {
	a, _ := bojanz.NewAmount("10.00", "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = a.Mul("3")
	}
}

// Section: MarshalJSON.

func BenchmarkMarshalJSON_GoMoney(b *testing.B) {
	m, _ := gomoney.New(1050, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(m)
	}
}

func BenchmarkMarshalJSON_Rhymond(b *testing.B) {
	m := rhymond.New(1050, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(m)
	}
}

func BenchmarkMarshalJSON_Bojanz(b *testing.B) {
	a, _ := bojanz.NewAmount("10.50", "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(a)
	}
}

// Section: UnmarshalJSON.

func BenchmarkUnmarshalJSON_GoMoney(b *testing.B) {
	data := []byte(`{"amount":"10.50","currency":"USD"}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m gomoney.Money
		_ = m.UnmarshalJSON(data)
	}
}

func BenchmarkUnmarshalJSON_Rhymond(b *testing.B) {
	data := []byte(`{"amount":1050,"currency":"USD"}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var m rhymond.Money
		_ = m.UnmarshalJSON(data)
	}
}

func BenchmarkUnmarshalJSON_Bojanz(b *testing.B) {
	data := []byte(`{"number":"10.50","currency_code":"USD"}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var a bojanz.Amount
		_ = json.Unmarshal(data, &a)
	}
}

// Section: Format.

func BenchmarkFormat_GoMoney(b *testing.B) {
	m, _ := gomoney.New(123456, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.String()
	}
}

func BenchmarkFormat_Rhymond(b *testing.B) {
	m := rhymond.New(123456, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.Display()
	}
}

func BenchmarkFormat_Bojanz(b *testing.B) {
	a, _ := bojanz.NewAmount("1234.56", "USD")
	locale := bojanz.NewLocale("en")
	f := bojanz.NewFormatter(locale)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Format(a)
	}
}

func BenchmarkFormat_Leekchan(b *testing.B) {
	ac := accounting.Accounting{Symbol: "$", Precision: 2}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ac.FormatMoney(1234.56)
	}
}
