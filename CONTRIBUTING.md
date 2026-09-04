# Contributing

Thanks for looking. `go-money` aims to be the safe money type for Go: exact int64 arithmetic, overflow guards on every hot path, fuzz-tested parsers, no float64 in the core. Contributions that preserve those properties are welcome.

## Ground rules

Before you open a PR, please:

1. **Preserve immutability.** `Money` is a value type. Methods return a new `Money`; they never mutate the receiver.
2. **Guard every arithmetic path against int64 overflow.** Silent wrap-around is treated as a critical bug.
3. **Keep float64 out of the core.** `FromDecimal` is the one documented entry point that touches float, and its precision caveats are called out in the doc. Do not add new code paths that depend on float arithmetic for correctness.
4. **Match the existing test layout.** Tests live in per-concern files: `arith_test.go`, `parse_test.go`, `format_test.go`, `json_test.go`, `text_test.go`, `sql_test.go`, `currency_test.go`, `decimal_test.go`, `errors_test.go`, `predicates_test.go`, `money_test.go`. Add new tests to the file whose subject matches the change; do not resurrect a single `money_test.go`.

## Regenerating currency data

`currencies.gen.go` is generated from `cmd/gen_currencies/iso-4217.json` by `cmd/gen_currencies/main.go`. Do not edit the generated file by hand.

```sh
go generate ./...
```

CI fails on drift, so any change to the JSON must be paired with a regenerated `currencies.gen.go` in the same PR.

## Tests

CI enforces:

- `go vet ./...` clean
- `staticcheck ./...` clean
- `govulncheck ./...` clean
- `go test -race ./...` pass
- 85% coverage floor on the root module (the `cmd/` generator is excluded)
- `go generate ./...` drift check
- SQL integration tests in `sqltest/` pass against an in-memory SQLite driver

Locally:

```sh
go test ./...
go test -race ./...
go test -fuzz=FuzzNewFromString -fuzztime=30s
cd sqltest && go test ./...
```

### Fuzz new parsers

If your change touches `NewFromString`, `UnmarshalText`, `UnmarshalJSON` or `Money.Scan`, run the relevant fuzz target for at least 30 seconds and paste the summary line into the PR description. Fuzz targets live in `parse_test.go` (`FuzzNewFromString`) and `json_test.go` (`FuzzUnmarshalJSON`).

### Benchmarks

Cross-library benchmarks against `Rhymond/go-money`, `bojanz/currency` and `leekchan/accounting` live in `benchmarks/`. If your change touches an arithmetic, parsing, format or codec path, include before/after numbers in the PR:

```sh
make bench-compare
```

Update `docs/BENCHMARK.md` if the headline table shifts materially.

## PR process

1. **Open an issue first for anything non-trivial.** Design decisions land better in a discussion than in a PR review.
2. **Follow Conventional Commits** for commit subjects (`feat:`, `fix:`, `refactor:`, `docs:`, `test:`, `perf:`, `chore:`, `ci:`, `build:`, `style:`, `revert:`). Total subject length under 110 characters. Breaking changes get `!` in the type.
3. **Update `README.md` and add an entry to `docs/BENCHMARK.md` or `CHANGELOG.md`** when the change is user-visible.
4. **Prefer small, focused PRs.** One concern per PR; if a PR grows past ~500 lines of net change, consider splitting.

## Licence

By contributing you agree that your work is licensed under this project's [BSD 3-Clause Licence](../LICENSE).
