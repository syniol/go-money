# Architecture

`go-money` is a monetary value type for Go. This document describes how it is built, why it is built that way, and how the automation around it works.

The library is small on purpose. The whole surface fits in around 2,000 lines of Go excluding the generated ISO 4217 table. Every design decision documented below is a trade-off, and each has an explicit rejection alternative recorded next to it so a future maintainer can see what was considered.

## Design principles

Five principles, in priority order. When they conflict, the earlier one wins.

1. **Correctness over convenience.** No API returns a value that is subtly wrong. When we cannot compute a right answer, we return `error`. This is why `Add`, `Sub`, `Mul`, `Compare`, `Neg`, `Abs`, `FromDecimal`, `NewFromString`, and every codec entry point can fail. `float64` never touches the core.
2. **Immutability by value.** `Money` is a small (16-byte) value type. Every mutating operation returns a new `Money`. There is no method with a pointer receiver on `Money` that mutates state. This makes concurrent use safe by construction and makes the type cheap to pass by value.
3. **Overflow is a first-class error.** Every arithmetic operation checks for `int64` overflow before performing the operation, not after. A silent wrap-around in a payment path is a critical bug; the design refuses to produce one.
4. **Zero allocation on hot paths.** Construct, parse, arithmetic and the plain `String()` render all measure at zero allocations per operation. Formatting via CLDR allocates on the order of half a dozen small buffers per call and is documented as such.
5. **Small dependency surface.** The runtime library depends on the Go standard library plus `golang.org/x/text` for locale-aware display. Nothing else. The `benchmarks/` and `sqltest/` sub-modules pull in comparison libraries and drivers respectively, but they are isolated modules and never appear in a consumer's `go.sum`.

## Package layout

```
go-money/
├── LICENSE
├── README.md
├── CHANGELOG.md
├── Makefile
├── go.mod                      # root module, minimal deps
├── money.go                    # Money type, constructors, predicates, accessors
├── currency.go                 # Currency type, sentinels, pow10, init validation
├── arith.go                    # Add, Sub, Mul, Split, Compare, Cmp, Equal, Neg, Abs
├── parse.go                    # NewFromString (fuzz-tested, zero-alloc)
├── decimal.go                  # FromDecimal, RoundingMode (float boundary code, isolated)
├── format.go                   # AsDecimalString, String, Format, LocalisedString, SymbolStyle
├── codec.go                    # MarshalJSON, UnmarshalJSON, MarshalText, UnmarshalText
├── sql.go                      # Money.Value, Money.Scan, NullMoney
├── errors.go                   # sentinel errors, MoneyError wrapper
├── currencies.gen.go           # generated ISO 4217 table (do not edit)
│
├── *_test.go                   # per-concern test files: arith, parse, format, etc.
├── example_test.go             # runnable doc examples for pkg.go.dev
│
├── cmd/gen_currencies/
│   ├── main.go                 # generator; reads iso-4217.json via go:embed
│   └── iso-4217.json           # ISO 4217 source data
│
├── benchmarks/                 # separate module: comparison vs Rhymond, bojanz, leekchan
│   ├── go.mod                  # replace directive points at ../
│   ├── compare_test.go
│   └── README.md
│
├── sqltest/                    # separate module: SQL codec integration via modernc/sqlite
│   ├── go.mod                  # replace directive points at ../
│   ├── integration_test.go
│   └── README.md
│
├── docs/
│   ├── ARCHITECTURE.md         # this file
│   ├── BENCHMARK.md
│   ├── CONTRIBUTING.md
│   ├── CODE_OF_CONDUCT.md
│   └── SECURITY.md
│
└── .github/workflows/
    ├── ci.yml                  # per-push and per-PR pipeline
    ├── bench.yml               # nightly cross-library benchmarks
    └── release.yml             # tag-triggered release with SBOM and attestation
```

The one-concern-per-file split in the root package is deliberate. Reviewers touching `Money.Add` open `arith.go` and see every arithmetic operation without noise. Reviewers touching the codecs open `codec.go` and see JSON and text next to each other, so the two wire formats stay aligned.

## Type system

Two exported types carry all state. Both are treated as immutable after construction.

### `Money`

```go
type Money struct {
    amount   int64
    currency *Currency
}
```

Sixteen bytes on 64-bit builds. Fields are unexported so no consumer can hand-forge a `Money` that bypasses the constructor's currency-code validation.

The `*Currency` pointer references an interned instance in the package-level `currencyConfig` map. Every `Money` for a given ISO code shares the same underlying `Currency` pointer. Currency comparison uses `ISOCode` string equality, not pointer identity, so a `*Currency` handed back through `GetCurrency` and re-used elsewhere still compares equal to a fresh `Money.New` result. Pointer identity would be one instruction faster but would break the moment a user obtained a `*Currency` and passed it back through the API.

The zero-value `Money` is deliberately invalid: `currency == nil` means the value was never through a constructor, and every method that touches currency refuses it. `Valid()` reports this explicitly.

### `Currency`

```go
type Currency struct {
    isoCode  string    // hot field
    symbol   string    // hot field
    decimals int       // hot field
    isoNum, isoDigits, numToBasic int
    name, demonym, symbolNative,
    majorSingle, majorPlural,
    minorSingle, minorPlural string
}
```

Fields are unexported to make the "read-only, shared instance" contract enforceable by the compiler rather than by developer discipline. Read access goes through method accessors (`ISOCode()`, `Decimals()`, `Symbol()`, and the rest).

Fields are ordered by access frequency. `isoCode`, `symbol` and `decimals` sit at the head of the struct so they land in the same cache line on 64-bit CPUs; cold ISO metadata and cold display fields follow.

Sentinel values encode "no upstream data":

- `ISONumUnknown = -1` for missing ISO numeric codes (real codes are 1..999).
- `NumToBasicUnknown = -2` (distinct sentinel value to keep accidental cross-comparison loud).
- `HasISONum()` / `HasNumToBasic()` are the intended query surface.

### `NullMoney`

```go
type NullMoney struct {
    Money Money
    Valid bool
}
```

Follows `sql.NullString` shape exactly. On `Scan(nil)`, `Valid` is cleared and `Money` is zeroed. On `Scan(err)`, `Valid` is cleared but `Money` is preserved (matches `sql.NullString.String` semantics). The staleness contract is documented on `NullMoney.Scan` and locked by `TestNullMoney_ScanFailurePreservesPreviousMoney`.

## Arithmetic

Every arithmetic entry point does two checks before touching the amount: currency mismatch, then overflow. Overflow checks are performed before the operation using standard patterns:

```go
if other.amount > 0 && m.amount > math.MaxInt64-other.amount {
    return Money{amount: 0, currency: m.currency}, &MoneyError{Op: "Add", Err: ErrOverflow}
}
```

On any failure the returned `Money` preserves the receiver's currency where one exists (`Money{amount: 0, currency: m.currency}`), not the zero `Money`. A caller who ignores the error and chains a second call fails on the original error class rather than on a spurious `ErrCurrencyMismatch` from a nil-currency operand.

`Split(n)` distributes the remainder one minor unit at a time to the first `remainder` parts so the sum of the pieces equals the input exactly, including for negative amounts and for `math.MinInt64`. `Cmp(other) int` panics on currency mismatch to fit the shape of `bytes.Compare` for `sort.Slice` adapters; `Compare(other) (int, error)` is the safe alternative when currencies are not guaranteed to match.

`Neg` and `Abs` refuse `math.MinInt64` because negating it does not fit in `int64`. `Mul` handles the `-1` case explicitly for the same reason.

## String parser

`NewFromString` is the single string entry point. It is ASCII-strict by design.

```
input string
    │
    ├── length gate (before TrimSpace): 0? -> ErrEmptyInput
    ├── TrimSpace
    ├── empty check after trim -> ErrEmptyInput
    ├── length gate (after TrimSpace) -> ErrInputTooLong
    ├── structural rejects: comma, multiple '.', multiple '-', misplaced '-'
    ├── structural stubs: ".", "-", "-." -> ErrInvalidFormat
    ├── strings.Cut on '.'                (integer and fractional halves)
    ├── ASCII digit sweep on each half   -> non-ASCII rejected
    ├── strings.ParseInt on integer half -> ErrRange mapped to ErrAmountTooLarge
    ├── strings.ParseInt on fractional half
    ├── explicit int64 overflow check on integer * pow10(decimals)
    ├── explicit int64 overflow check on fractional adjustment
    └── Money{amount: total, currency: cfg}
```

Non-ASCII numeral scripts (Arabic-Indic, Devanagari, full-width) are rejected to prevent homoglyph attacks and localisation ambiguity. The parser is documented as "the caller must canonicalise input before calling".

Zero allocation. Verified by `BenchmarkNewFromString` at 53 ns/op, 0 B/op, 0 allocs/op on Apple M2. Locked in by `FuzzNewFromString` with 3 million+ executions of no-panic verification.

## Codecs

Four codecs, all deriving from the same wire concept:

| Codec | Wire format | Symmetry |
|---|---|---|
| `MarshalText` / `UnmarshalText` | `"<decimal> <ISO>"` | round-trips through `NewFromString` |
| `MarshalJSON` / `UnmarshalJSON` | `{"amount":"<decimal>","currency":"<ISO>"}` | round-trips through `NewFromString` |
| `Value` / `Scan` (`database/sql`) | `<decimal> <ISO>` (identical to text) | delegates to `MarshalText` / `UnmarshalText` |
| `NullMoney.Value` / `Scan` | same or SQL NULL | wraps `Money` with `Valid` bit |

`Value()` returns `[]byte` rather than `string` to skip one allocation vs the previous `string(txt)` conversion. `Scan(src any)` handles `string`, `[]byte` and `sql.RawBytes` (the third is a distinct named type; Go type switches match by identity, not underlying type).

`MarshalJSON` writes amount as a JSON string (`"10.50"`) rather than a number to prevent client-side `float64` precision loss when a JavaScript client parses the payload. `Rhymond/go-money` writes it as a number (`1050`) and saves a few bytes at the cost of that class of bugs.

All JSON errors are wrapped in `MoneyError` so consumers categorising by `MoneyError.Op` see the same taxonomy across every codec.

## Localisation

`LocalisedString(tag, opts...)` renders CLDR-aware output without going through `float64`. It is the one function where we deliberately trade allocation cost for correctness of display.

The tricky part is that `golang.org/x/text/currency` exposes no way to fetch the CLDR pattern for a locale directly. You can only ask it to format a value. So we probe:

```
probe := p.Sprint(currency.NarrowSymbol(cur.Amount(0.0)))  // e.g. "$0.00"
zeroNumber := p.Sprintf("%.*f", c.decimals, 0.0)           // e.g. "0.00"
idx := strings.LastIndex(probe, zeroNumber)
// prefix := probe[:idx]           // "$"
// suffix := probe[idx+len(zeroNumber):]
// return prefix + numberStr + suffix
```

We probe with `0.0` (not `1.0`) so the placeholder cannot collide with template literals like `"1.00"` that appear in some exotic CLDR patterns. The negative sign lives in `numberStr` already so the probe never needs to match a signed form. If the probe fails for any reason (unknown locale rendering, no numeric slot found), we fall back to `symbol + numberStr` so the caller always gets a non-empty rendering.

Non-ASCII whitespace produced by CLDR (notably the NBSP used as thousands separator in French, Russian and Swedish) is preserved. An earlier version stripped all Unicode whitespace and destroyed those separators; the fix is locked by `TestLocalisedString_NBSPThousandsSeparator`.

The decimal-separator sniff iterates runes rather than bytes so locales with non-ASCII digit shaping (Arabic locale) do not slice mid-codepoint. Locked by `TestLocalisedString_ArabicLocaleNoCorruption`.

## Currency data pipeline

Currency metadata is generated once, embedded, and read at package init.

```
cmd/gen_currencies/iso-4217.json
        │
        │  //go:embed  (compile-time, no runtime file access)
        ▼
cmd/gen_currencies/main.go
        │
        │  go generate ./...  (developer or CI)
        │
        │  text/template with sorted-keys iteration for deterministic output
        ▼
currencies.gen.go   (map[string]*Currency)
        │
        │  package init:
        │    for _, c := range currencyConfig {
        │        if c.decimals > MaxSafeDecimals { panic(...) }
        │    }
        ▼
runtime lookup via GetCurrency, Currencies, currencyConfig
```

The generator embeds `iso-4217.json` at build time via `//go:embed`, so it works from any directory. The template writes to `currencies.gen.go` at the repo root, and CI fails on drift via the `Generate` job.

The `init()` in `currency.go` walks the generated map and panics if any currency declares `decimals > MaxSafeDecimals`. This is defence in depth: a broken generator fails at package load rather than returning an error on every subsequent `New()` call.

## Error taxonomy

Twelve sentinel errors, all wrapped by `MoneyError`.

```go
type MoneyError struct {
    Op       string   // e.g. "Add", "NewFromString", "MarshalJSON"
    Amount   string   // input causing the failure (optional)
    Currency string   // currency involved (optional)
    Err      error    // wrapped sentinel or upstream error
}
```

`errors.Is(err, money.ErrOverflow)` works because `MoneyError` implements `Unwrap`. The `Op` field is deliberately exposed so operational tooling can categorise failures by operation without pattern-matching the message string. Nil-receiver `Error()` and `Unwrap()` are guarded.

The full sentinel table lives in the README.

## Concurrency

`Money` is immutable, so it is safe to share and pass by value across goroutines without locks. `Currency` is populated once at package init and never mutated; the map itself is not modified after init.

There are no goroutines, channels, mutexes or atomics in the library. If you spawn work in parallel that touches `Money` values, no coordination is required.

## Performance profile

Zero-allocation paths (verified by `-benchmem`):

- `New(minor, code)` and `MustNew`
- `NewFromString(s, code)`
- `Add`, `Sub`, `Mul`, `Cmp`, `Compare`, `Equal`, `Neg`, `Abs`
- `AsDecimalString` for zero-decimal currencies (single allocation for `strconv.FormatInt`)
- `Format` for `%s` and `%v` common paths (writes bytes to `fmt.State` directly)

Allocating paths (documented):

- `AsDecimalString` and `String` for non-zero-decimal currencies: one small allocation for the returned string.
- `MarshalJSON`: five small allocations for the JSON envelope.
- `UnmarshalJSON`: seven allocations, six of which are inside `encoding/json` decoding into the auxiliary struct.
- `LocalisedString`: roughly five to six small allocations per call for the CLDR template dance.
- `Split(n)`: allocates `[]Money` of length `n`.

The trade-offs are explicit in each function's doc comment. Cross-library benchmarks in `benchmarks/` measure the same operations against `Rhymond/go-money`, `bojanz/currency` and `leekchan/accounting`; results in [`BENCHMARK.md`](BENCHMARK.md).

## Testing strategy

Three layers, in order of increasing cost.

**Unit tests**, one file per concern:

| File | Subject |
|---|---|
| `money_test.go` | constructors |
| `predicates_test.go` | `Valid`, `IsZero`, `IsPositive`, `IsNegative` |
| `arith_test.go` | arithmetic, sign helpers, sort ordering |
| `parse_test.go` | `NewFromString` including boundary and fuzz |
| `decimal_test.go` | `FromDecimal` including float precision trap |
| `format_test.go` | `AsDecimalString`, `String`, `Format`, `LocalisedString` |
| `codec_test.go` (split into `json_test.go`, `text_test.go`) | JSON/text codecs |
| `sql_test.go` | SQL codec unit tests |
| `currency_test.go` | Currency metadata and generator invariants |
| `errors_test.go` | `MoneyError` format contract |

**Fuzz tests**: `FuzzNewFromString` and `FuzzUnmarshalJSON`. Both use property-based invariants (currency identity preservation, sign preservation, no panic) rather than golden values so new inputs discovered during fuzzing extend coverage automatically. The CI runs them briefly on every push; developers are expected to run 30-second local fuzz on any parser change (documented in `CONTRIBUTING.md`).

**Integration tests**: `sqltest/` is a separate module that pulls in `modernc.org/sqlite` (pure Go, no CGO) to exercise `Value`/`Scan` and `NullMoney` end to end against a real driver. Kept in its own module so consumers of `github.com/syniol/go-money` never carry a SQLite dependency.

**Benchmarks**: `benchmarks/` is a separate module that measures `go-money` alongside `Rhymond/go-money`, `bojanz/currency` and `leekchan/accounting`. Pinned versions in `benchmarks/go.mod` make results reproducible. Numbers land in `BENCHMARK.md` on release.

## CI pipelines

Three GitHub Actions workflows, each with a distinct trigger and job set.

### `ci.yml`: per-push and per-PR

```
push / pull_request to main
        │
        ├── Test (Go 1.24.x)        go test -race -coverprofile
        │       └── 85% coverage floor (cmd/ excluded via awk)
        │
        ├── Vet                     go vet ./...
        ├── Lint                    golangci-lint
        ├── Staticcheck             staticcheck ./...
        ├── Govulncheck             govulncheck ./...
        ├── Build                   go build -v ./...
        ├── Generate                go generate ./... + git diff --exit-code
        └── SQL Integration         cd sqltest && go test -race ./...
```

Eight parallel jobs; every green tick is a live invariant. The 85% coverage floor uses `awk` to filter out `cmd/gen_currencies` (build-time tool, not runtime code) before computing the total.

### `bench.yml`: nightly and manual

```
cron 03:00 UTC / workflow_dispatch
        │
        └── Comparison benchmarks   cd benchmarks && go test -bench=. -benchmem -benchtime=3s
                                    │
                                    └── upload bench.out artefact (30-day retention)
```

Nightly runs produce a benchmark artefact against pinned comparison libraries. Regression alerting is not yet gated; a future step adds a threshold that opens an issue on material regression.

### `release.yml`: tag-triggered

```
push tag v[0-9]+.[0-9]+.[0-9]+
        │
        ├── verify-tag              refuse if tag not reachable from origin/main
        │
        ├── test                    go vet, go test -race, go generate drift
        │
        ├── sql-integration         cd sqltest && go test -race ./...
        │
        └── publish-release
                ├── git archive     go-money-<version>.tar.gz
                ├── sha256sum       go-money-<version>.tar.gz.sha256
                ├── SPDX SBOM       via anchore/sbom-action
                ├── attestation     via actions/attest-build-provenance
                └── GitHub Release  auto-generated notes + three artefacts
```

The pipeline refuses to run if the tag is not reachable from `origin/main`, so an accidental tag on a feature branch cannot ship. Every artefact is signed with a GitHub OIDC-based provenance attestation, and the SPDX SBOM enumerates every dependency for supply-chain auditors.

`pkg.go.dev` picks up the new tag via the Go module proxy automatically; no explicit publish step is required.

## Extension points

Deliberately narrow, and every extension has a corresponding open issue.

- **Big-integer amounts** for crypto and hyperinflation (`BigMoney`): [issue #17](https://github.com/syniol/go-money/issues/17). Would ship as a parallel type sharing the API shape of `Money`.
- **FX conversion** (`Rate`, `Money.Convert`): [issue #18](https://github.com/syniol/go-money/issues/18). Rate is a numerator/denominator pair with explicit rounding at conversion time.
- **`shopspring/decimal` interop**: [issue #15](https://github.com/syniol/go-money/issues/15). Ships as a sub-package (`github.com/syniol/go-money/shopspring`) so the runtime library stays dependency-free for callers that do not need it.
- **Multi-currency `Wallet`**: [issue #19](https://github.com/syniol/go-money/issues/19). A holder for zero-or-more `Money` values, at most one per currency, with `Deposit`/`Withdraw`/`Convert`/`Total` semantics.

Sub-package extensions are the pattern for anything that needs its own dependency (like `shopspring/decimal`). Runtime API extensions live in the root package.

## Non-goals

Recorded so a well-intentioned contributor does not open a PR against them.

- **FX rate sourcing.** The library will not fetch rates from any provider. A future `Rate` type is a value object; where the rates come from is the caller's responsibility.
- **Currency creation at runtime.** New currencies land through the code generator and the `iso-4217.json` source. Runtime registration is rejected because it makes concurrent invariants impossible to guarantee for the `currencyConfig` map.
- **Locale-aware parsing.** `NewFromString` is ASCII-strict. Localised input goes through the caller's UI layer and reaches the library canonicalised. This prevents homoglyph attacks and keeps the parser fuzz-tractable.
- **Time-series or historical rates.** Out of scope; the library is a value type, not a time series.
- **Support for `float64` in the core arithmetic.** `FromDecimal` is the one documented boundary and carries a warning; no other API path touches float.

## Further reading

- [`README.md`](../README.md) for API and quick-start.
- [`BENCHMARK.md`](BENCHMARK.md) for reproducible cross-library benchmarks.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) for the contributor workflow and release procedure.
- [`SECURITY.md`](SECURITY.md) for vulnerability reporting.
- [`CHANGELOG.md`](../CHANGELOG.md) for what shipped in each release.
