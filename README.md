# 🏦 Go-Money: The High-Precision Banking Core

[![Go Reference](https://pkg.go.dev/badge/github.com/syniol/go-money.svg)](https://pkg.go.dev/github.com/syniol/go-money)
[![Go Report Card](https://goreportcard.com/badge/github.com/syniol/go-money)](https://goreportcard.com/report/github.com/syniol/go-money)
[![License: BSD-3](https://img.shields.io/badge/License-BSD-blue.svg)](https://opensource.org/license/bsd-3-clause)

`go-money` is a mission-critical Go library designed for financial institutions and high-integrity fintech applications. It treats money as a mathematical primitive—implementing strict value semantics, immutable operations, and stack-allocation to ensure maximum performance and zero precision loss.

## 💎 The Golden Rules of Financial Engineering

Most libraries fail by treating money as a generic data structure or, worse, a floating-point number. `go-money` is built on three non-negotiable architectural pillars:

1.  **Zero Floating Point:** Absolute protection against IEEE-754 rounding errors. Money is stored as an `int64` representing the **minor unit** (e.g., $1.00 USD is `100`).
2.  **Hardened Arithmetic:** Every addition, subtraction, and multiplication is guarded against silent integer overflows.
3.  **Stack-Only Allocation:** The `Money` struct is designed to stay on the stack, bypassing Garbage Collector (GC) pressure to provide predictable latency in high-throughput ledger environments.

---

## 🚀 Key Features

* **ISO-4217 Compliance:** Pre-generated support for 150+ global currencies.
* **Zero-Allocation Parsing:** `NewFromString` performs manual byte-walking to parse inputs without intermediate string splits or heap allocations.
* **Banker's Rounding:** Native support for `RoundHalfToEven` (the international standard for minimizing cumulative bias in financial sums).
* **JSON Value Semantics:** Custom Marshallers that serialize amounts as strings (e.g., `"10.50"`) to maintain compatibility across JavaScript clients without precision loss.

---

## 🛠 Usage

### Initialization
```go
import "github.com/syniol/go-money"

// Safe creation from minor units (cents)
m, err := money.New(1050, "USD") // $10.50

// Hardened string parsing (Ideal for API inputs)
price, err := money.NewFromString("1234.56", "EUR")

// Panic-safe MustNew for package-level constants
var DefaultFee = money.MustNew(500, "USD") // $5.00
```

### High-Integrity Arithmetic

```go
bal := money.MustNew(1000, "USD")
fee := money.MustNew(200, "USD")

// Arithmetic returns a new value; original remains immutable
total, err := bal.Add(fee)
if err != nil {
    // Handles ErrCurrencyMismatch or ErrOverflow
}

// Comparison
isWealthy, _ := total.IsGreaterThan(money.MustNew(100, "USD"))
```

### Advanced Rounding (The Banker's Way)
When converting from decimals (like tax percentages), precision is paramount.
```go
// Calculate a 7.5% tax on $10.50
tax, _ := money.FromDecimal(10.50 * 0.075, "USD", money.RoundHalfToEven)
```

---

## 🛡 Security & Safety

### Overflow Protection
A standard `int64` can hold up to $92 Quadrillion (in USD). However, even at this scale, 
multiplication can cause overflows. `go-money` detects these boundaries:
```go
m := money.MustNew(math.MaxInt64, "USD")
_, err := m.Add(money.MustNew(1, "USD")) 
// Returns ErrOverflow instead of wrapping to a negative number.
```

### JSON Precision Guard
When sending data to a browser, `JSON.parse()` will turn large numbers into floats, destroying 
your data. `go-money` prevents this by forcing string serialization:

```json
{
  "amount": "12500.00",
  "currency": "USD"
}
```
---
## 📊 Performance Benchmarks
`go-money` is optimized for zero-allocation paths. In a typical financial transaction lifecycle, 
this library introduces **zero GC pressure**.

| Operation     | Time      | Objects Allocated |
|---------------|-----------|-------------------|
| NewFromString | 42 ns/op  | 0 B/op            |
| Add           | 1.2 ns/op | 0 B/op            |
| MarshalJSON   | 85 ns/op  | 48 B/op           |

---
## 📜 ISO-4217 Data
Currency data is generated via `cmd/gen_currencies`. To update the local definitions with the 
latest ISO standards:
```shell
go generate ./...
```

## ⚖ License
Distributed under the **BSD 3-Clause License**. See `LICENSE` for more information.

---
Built for the next generation of Fintech. Developed by [Syniol Limited](https://syniol.com).
