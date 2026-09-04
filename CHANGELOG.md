# Changelog

All notable changes to this project are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Community and documentation files (`CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, `BENCHMARK.md`) moved to `docs/`. GitHub Community Standards side-panel entries stay detected via `docs/`.
- `iso-4217.json` moved into `cmd/gen_currencies/` and embedded at build time via `//go:embed`. The generator is now independent of the working directory it is invoked from.
- `CONTRIBUTING.md` rewritten to match the README register (drop marketing prose and H2 emojis), point at per-concern test files and align coverage guidance with the CI 85% floor.
- Quick start snippet in README declares `prices` so it compiles as-is.

### Added
- `Money.Value`, `Money.Scan` and `NullMoney` for `database/sql` round-trips into `TEXT`/`VARCHAR(24)` columns. Wire format is `"<decimal> <ISO>"`, matching `MarshalText`. Closes #16.
- `GetCurrency(code)` and `Currencies()` for reachable Currency metadata without constructing a Money.
- `Neg` and `Abs` methods on Money for sign helpers.
- `Cmp(other) int` panicking comparison variant for `sort.Slice` adapters.
- `Valid()` accessor on Money.
- `MarshalText` and `UnmarshalText` for YAML, TOML, URL and form encoding.
- `fmt.Formatter` implementation: `%+v` prints a structured view; `%#v` prints a reconstructible Go expression.
- `SymbolStyle` option on `LocalisedString` for narrow, standard or ISO symbol forms.
- `HasISONum` and `HasNumToBasic` accessors alongside `ISONumUnknown = -1` and `NumToBasicUnknown = -2` sentinels.
- `FuzzUnmarshalJSON` covers the JSON parsing boundary.

### Changed
- `LocalisedString` now probes the CLDR template with a zero-value amount for robust prefix/suffix discovery (previously used a live-sample substitution that could collide with template literals).
- Fractional format strings are precomputed at init to eliminate a nested `fmt.Sprintf` from every `LocalisedString` call.
- Currency ISOCode comparison replaces pointer identity in `assertSameCurrency` and `Equal`.
- CI adds `staticcheck`, `govulncheck`, an 85% coverage floor (excluding the code generator) and a `go generate` drift check.
- Go module matrix trimmed to `1.24.x` in line with `go.mod`.

### Fixed
- `UnmarshalJSON` preserves the underlying `json` error alongside `ErrMalformedInput` via a two-verb `%w` wrap.
- `NewFromString` classifies overflowing integer part as `ErrAmountTooLarge` rather than `ErrInvalidFormat`.
- `NewFromString` classifies whitespace-only input as `ErrEmptyInput`; length cap now applies after trimming.
- `FromDecimal` rejects floats at the exact `MaxInt64` boundary (`>=` instead of `>`) to avoid a spec-undefined int64 conversion.
- `FromDecimal` rejects unknown `RoundingMode` with `ErrInvalidRoundingMode` instead of silently falling back to banker's rounding.
- `LocalisedString` no longer strips NBSP; French, Russian and Swedish thousands separators are preserved.
- `LocalisedString` decimal separator sniff iterates runes so non-ASCII digit locales (e.g. Arabic) do not slice mid-codepoint.
- `Mul` overflow returns a currency-preserving zero (matches `Add`/`Sub` shape).
- `Add`, `Sub`, `Mul`, `FromDecimal`: every error path preserves the receiver's currency where known.
- `Split` message reports the numeric `MaxSplitParts` value instead of the identifier name.
- `MarshalJSON` uses append-based byte building (no `fmt.Sprintf`).
- `MoneyError.Error` and `Unwrap` guard against a nil receiver.
- `MoneyError.Error` defaults an empty `Op` to `"?"` rather than emitting `"money.: ..."`.
- `MustNew` panic uses the underlying `*MoneyError` (no doubled `money.` prefix).
- Generator preserves null distinction for `ISONum` and `NumToBasic` via typed sentinels.
- `IsZero`, `IsPositive`, `IsNegative` require a currency; a zero-value Money satisfies none of them.
- `Equal` returns true for two zero-value Moneys (matches `time.Time.Equal`).

### Removed
- `PluralMajorUnit` / `PluralMinorUnit` helpers (English-only, misleading). Use `golang.org/x/text/feature/plural` against the raw field accessors instead.

### BREAKING (accumulated since v1.3.0)
- `Config` type renamed to `Currency`.
- Comparison methods `IsLessThan`, `IsGreaterThan`, `IsEqual` removed. Use `Compare(other) (int, error)` and `Equal(other) bool`.
- `PluralMajorUnit` and `PluralMinorUnit` removed.
- `Currency` fields unexported; read via `ISOCode()`, `Symbol()`, `Decimals()` and the other accessors.

## [1.3.0]

Earlier releases predate this changelog. See the Git history for details.
