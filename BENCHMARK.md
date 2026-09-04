# Benchmarks

Comparison of `github.com/syniol/go-money` against three established Go money and money-adjacent libraries on the operations most commonly hit in a payment or ledger service.

## How to run

```sh
cd benchmarks
go test -bench=. -benchmem -run=^$ -benchtime=3s -count=1
```

The `benchmarks/` directory is a separate module, so the comparison libraries never leak into consumers of `github.com/syniol/go-money`.

## Environment

| Field | Value |
|---|---|
| CPU | Apple M2 |
| OS | darwin/arm64 |
| Go | 1.24.7 |
| go-money | this branch (see git log) |
| Rhymond/go-money | v1.0.15 |
| bojanz/currency | v1.4.4 |
| leekchan/accounting | v1.0.0 |

## Results

Numbers below are from `-benchtime=1s -count=1`. `ns/op` is per operation, `B/op` and `allocs/op` are per operation as reported by `-benchmem`. Lower is better on all three columns.

### Construct from int64 minor units

| Library | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| **go-money** | **12.3** | **0** | **0** |
| Rhymond | 22.8 | 16 | 1 |
| bojanz | N/A (parse-only constructor) | N/A | N/A |

### Parse from string

| Library | ns/op | B/op | allocs/op | Notes |
|---|---:|---:|---:|---|
| **go-money** | **53.3** | **0** | **0** | fuzz-tested, ASCII-strict |
| bojanz | 92.6 | 8 | 1 | decimal-backed |
| Rhymond (float) | 34.7 | 16 | 1 | via `NewFromFloat`; not exact for arbitrary decimals |

Rhymond has no string parser; `NewFromFloat` is the closest analogue and is documented as unsafe for exact input.

### Add

| Library | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| **go-money** | **3.0** | **0** | **0** |
| bojanz | 23.1 | 0 | 0 |
| Rhymond | 37.0 | 32 | 2 |

### Sub

| Library | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| **go-money** | **3.0** | **0** | **0** |
| bojanz | 24.8 | 0 | 0 |
| Rhymond | 37.1 | 32 | 2 |

### Mul (integer multiplier)

| Library | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| **go-money** | **1.9** | **0** | **0** |
| Rhymond | 36.1 | 32 | 2 |
| bojanz | 60.1 | 0 | 0 |

bojanz `Mul` takes a decimal string so pays the parse cost per call.

### MarshalJSON

| Library | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| **go-money** | **277** | 197 | 5 |
| Rhymond | 326 | 152 | 5 |
| bojanz | 380 | 181 | 5 |

### UnmarshalJSON

| Library | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| **go-money** | **448** | **256** | **7** |
| Rhymond | 602 | 632 | 16 |
| bojanz | 838 | 648 | 13 |

### Format for display

| Library | ns/op | B/op | allocs/op | What it does |
|---|---:|---:|---:|---|
| **go-money** `String()` | **27.3** | **8** | **1** | symbol + decimal, no locale |
| Rhymond `Display()` | 106 | 40 | 4 | symbol + decimal, no locale |
| leekchan `FormatMoney` | 531 | 216 | 12 | grouped digits, one locale |
| bojanz `Format` | 882 | 1448 | 22 | full CLDR, all locales |

The lower rows do more work; comparison is illustrative rather than apples-to-apples. For CLDR-aware output, `go-money`'s `LocalisedString` costs roughly 5-6 allocations per call and is documented as not zero-allocation.

## Summary

`go-money` wins on every measured integer arithmetic and codec path. The wins are large: arithmetic is 8x-19x faster than the incumbent (Rhymond) with no per-op allocation, and unmarshalling JSON is 1.3x-1.9x faster while using half the bytes and allocations.

Where `go-money` does not win outright:

- **Rhymond `NewFromFloat`** is faster than `go-money` `NewFromString`, but goes through `float64` and loses precision on arbitrary decimals. It is the wrong comparison for correct input.
- **Rhymond `MarshalJSON`** allocates fewer bytes than `go-money`'s (152 vs 197) because it stores amount as an integer. `go-money` serialises amount as a string to prevent JavaScript float precision loss on the client; that is a deliberate trade for correctness and matches every high-integrity JSON contract.
- **bojanz** is the only alternative with genuinely arbitrary precision. If you need amounts larger than int64 can hold (some crypto, some pathological hyperinflation cases), bojanz is the right choice today. Track [issue #17](https://github.com/syniol/go-money/issues/17) for `BigMoney` support here.

## Reproducibility

The comparison harness in `benchmarks/compare_test.go` pins every library version in `benchmarks/go.mod`. Re-running on different hardware will produce different absolute numbers; the relative ordering has held across every run so far.

If you get materially different numbers on your workload, please open an issue with your `go env`, hardware, and the raw benchmark output.
