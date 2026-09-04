# go-money: floats lie.

Integer-backed monetary amounts for Go. `int64` minor units, overflow-safe arithmetic, fuzz-tested string parser, ISO 4217 metadata generated from the official source, JSON and text codecs, CLDR-aware localised display.

[![CI](https://github.com/syniol/go-money/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/syniol/go-money/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/syniol/go-money.svg)](https://pkg.go.dev/github.com/syniol/go-money)
[![License: BSD](https://img.shields.io/badge/License-BSD-blue.svg)](https://opensource.org/license/bsd-3-clause)
[![ISO 4217](https://img.shields.io/badge/ISO%204217-compliant-brightgreen.svg)](https://www.iso.org/iso-4217-currency-codes.html)

## When to use this

Pick `go-money` if you want:

- Exact arithmetic on monetary amounts without `float64` rounding.
- Explicit overflow errors on every `Add`, `Sub`, `Mul` (no silent wrap).
- Currency mismatch errors on every arithmetic operation.
- A fuzz-tested, zero-allocation string parser for wire and user input.
- A small dependency footprint: only `golang.org/x/text` is required, and only for locale-aware display.

Pick something else if you need:

- Arbitrary-precision decimals across the API today. See [`github.com/bojanz/currency`](https://github.com/bojanz/currency).
- Amounts larger than `int64` can hold (some crypto). Track [issue #17](https://github.com/syniol/go-money/issues/17) for `BigMoney`.
- Built-in currency conversion. Track [issue #18](https://github.com/syniol/go-money/issues/18).
- SQL scanning for numeric-typed columns. `TEXT`/`VARCHAR` is supported natively via `Money.Value` and `Money.Scan`; a pgx `NUMERIC` codec is a follow-up.

## Feature comparison

| | `go-money` | `Rhymond/go-money` | `bojanz/currency` |
|---|---|---|---|
| Backing type | int64 | int64 | decimal (arbitrary) |
| Overflow-safe arithmetic | error | wraps silently | native (no overflow) |
| String parser | fuzz-tested, zero-alloc | none (float only) | decimal-backed |
| Currency mismatch guard | error | error | error |
| Locale-aware display | CLDR via x/text | none | CLDR (native) |
| `sql.Scanner`/`driver.Valuer` | yes (TEXT/VARCHAR) | no | yes |
| FX conversion | tracked in #18 | single rate | no |
| Fuzz tests | yes | no | some |
| Big-integer amount | tracked in #17 | no | native |

## Benchmarks

Selected results on Apple M2, Go 1.24.7. Full table and methodology in [BENCHMARK.md](BENCHMARK.md).

| Operation | `go-money` | Rhymond | bojanz |
|---|---:|---:|---:|
| Parse from string | **53 ns/op, 0 allocs** | 35 ns (float, lossy) | 93 ns, 1 alloc |
| Add | **3 ns/op, 0 allocs** | 37 ns, 2 allocs | 23 ns, 0 allocs |
| Mul | **1.9 ns/op, 0 allocs** | 36 ns, 2 allocs | 60 ns, 0 allocs |
| MarshalJSON | **277 ns/op** | 326 ns | 380 ns |
| UnmarshalJSON | **448 ns/op, 7 allocs** | 602 ns, 16 allocs | 838 ns, 13 allocs |

Reproduce with `make bench-compare` or `cd benchmarks && go test -bench=. -benchmem`.

## Install

```sh
go get github.com/syniol/go-money
```

## Quick start

```go
import (
    "encoding/json"
    "fmt"
    "sort"

    "github.com/syniol/go-money"
)

// From minor units (cents).
m, _ := money.New(1050, "USD")

// From an ASCII decimal string (fuzz-tested, refuses non-ASCII digits).
price, _ := money.NewFromString("1234.56", "EUR")

// Exact int64 arithmetic; errors on currency mismatch or overflow.
total, _ := m.Add(m)
neg, _   := total.Neg()
abs, _   := neg.Abs()

// Comparison. Cmp panics on mismatch, use it in sort adapters.
sort.Slice(prices, func(i, j int) bool {
    return prices[i].Cmp(prices[j]) < 0
})

// JSON: {"amount":"10.50","currency":"USD"}
data, _ := json.Marshal(m)

// Text (YAML, TOML, URL params, flag.TextVar): "10.50 USD"
txt, _ := m.MarshalText()

fmt.Println(m, price, total, neg, abs, string(data), string(txt))
```

## SQL columns

`Money` satisfies `database/sql/driver.Valuer` and `sql.Scanner` for `TEXT` or `VARCHAR(24)` columns. `NullMoney` mirrors `sql.NullString` for nullable columns.

```go
// Write
_, err := db.Exec(`INSERT INTO orders (id, total) VALUES ($1, $2)`, id, m)

// Read
var m money.Money
err = db.QueryRow(`SELECT total FROM orders WHERE id = $1`, id).Scan(&m)

// Nullable
var nm money.NullMoney
err = db.QueryRow(`SELECT refund FROM orders WHERE id = $1`, id).Scan(&nm)
if nm.Valid {
    // nm.Money is populated
}
```

Wire format is `"<decimal> <ISO>"` (e.g. `"10.50 USD"`), the same as `MarshalText`, so a value stored via `database/sql` round-trips identically through JSON, YAML and TOML.

## Currency metadata

```go
usd, ok := money.GetCurrency("USD")
if ok {
    fmt.Println(usd.ISOCode(), usd.Symbol(), usd.Decimals())
}

for _, code := range money.Currencies() {
    // 150+ codes generated from iso-4217.json
    _ = code
}
```

## Localised display

```go
m := money.MustNew(123456789, "EUR")

fmt.Println(m.LocalisedString(language.French))
// "1 234 567,89 €"  (NBSP thousands separator preserved)

fmt.Println(m.LocalisedString(language.AmericanEnglish, money.SymbolStyleISO))
// "USD 1,234,567.89"
```

`LocalisedString` is not zero-allocation (typically five to six small allocations per call for the CLDR template dance). Use `String()` for hot paths and `LocalisedString` for user-facing text.

## Precision limits

Amounts are stored as `int64` minor units. Practical maxima:

| Currency decimals | Max representable amount |
|---:|---|
| 0 (JPY, KRW) | ±9.22 × 10^18 |
| 2 (USD, EUR, GBP) | ±$92 quadrillion |
| 4 (CLF) | ±$922 trillion |
| 8 (crypto sats) | ±$92 billion |
| 12 | ±$9 million |

If you need larger amounts, track [issue #17](https://github.com/syniol/go-money/issues/17).

## Error handling

Every failure returns a `*money.MoneyError` wrapping a sentinel error, comparable with `errors.Is`.

```go
_, err := m.Add(eur)
switch {
case errors.Is(err, money.ErrCurrencyMismatch):
    // ...
case errors.Is(err, money.ErrOverflow):
    // ...
}
```

Sentinels:

| Sentinel | Fires when |
|---|---|
| `ErrInvalidCurrency` | Currency code is empty, not ISO 4217, or unknown to the library. |
| `ErrInvalidFormat` | `NewFromString` input has structural problems the strict validator rejects (stray characters, wrong shape). |
| `ErrTooMuchDetail` | String input carries more fractional digits than the currency's `Decimals` allows (e.g. `"1.234"` for USD). |
| `ErrCurrencyMismatch` | An arithmetic or comparison operation combines two `Money` values with different currencies. |
| `ErrOverflow` | `Add`, `Sub`, `Mul`, `Neg` or `Abs` would exceed `int64` range. |
| `ErrAmountTooLarge` | `NewFromString` or `FromDecimal` produces a value beyond `math.MaxInt64` or below `math.MinInt64`. |
| `ErrInputTooLong` | `NewFromString` input exceeds `MaxStringLength` (64 bytes after trimming). |
| `ErrEmptyInput` | Parser or codec receives an empty or whitespace-only value, or a SQL `NULL` is scanned into a plain `Money`. |
| `ErrMalformedInput` | JSON, text or SQL input is unparseable or of an unsupported source type. |
| `ErrInvalidSplitCount` | `Split(n)` called with `n` outside `[1, MaxSplitParts]`. |
| `ErrInvalidRoundingMode` | `FromDecimal` called with a mode not in the exported `RoundingMode` set. |
| `ErrUnsafeScale` | Defence in depth: a `Currency`'s `Decimals` exceeds `MaxSafeDecimals`. In practice only fires if the code generator is broken. |

## Development

```sh
git clone https://github.com/syniol/go-money && cd go-money
go test ./...
go test -race ./...
go test -fuzz=FuzzNewFromString -fuzztime=30s
make bench-compare
```

Every merge to `main` runs `go vet`, `staticcheck`, `govulncheck`, tests with race detector, an 85% coverage floor and a generator-drift check via GitHub Actions.

## Project files

- [CHANGELOG.md](CHANGELOG.md) release history in Keep-a-Changelog format
- [BENCHMARK.md](BENCHMARK.md) reproducible comparison against Rhymond, bojanz, leekchan
- [SECURITY.md](SECURITY.md) private vulnerability reporting workflow
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) Contributor Covenant 2.1
- [CONTRIBUTING.md](CONTRIBUTING.md) how to build, test and submit changes

## Data source

Currency metadata is generated from `iso-4217.json` by `cmd/gen_currencies`. Regenerate with `go generate ./...`; CI fails on drift. See the [ISO 4217 standard](https://www.iso.org/iso-4217-currency-codes.html) for the upstream.

## Roadmap

Adoption-driving items are tracked as issues; contributions welcome on any of these:

- [#17](https://github.com/syniol/go-money/issues/17) `BigMoney` backed by `big.Int`
- [#18](https://github.com/syniol/go-money/issues/18) FX `Rate` and `Money.Convert`
- [#15](https://github.com/syniol/go-money/issues/15) `shopspring/decimal` interop under a sub-package
- [#19](https://github.com/syniol/go-money/issues/19) multi-currency `Wallet` type

## Licence

BSD 3-Clause. See [LICENSE](LICENSE).
