# benchmarks

Comparison benchmarks for `github.com/syniol/go-money` against pinned versions of `github.com/Rhymond/go-money`, `github.com/bojanz/currency`, and `github.com/leekchan/accounting`.

This is a separate Go module (`go.mod` here) so the comparison dependencies never appear in consumers of the root `go-money` module. A `replace` directive points the module at `../` for local development.

## Run

```sh
go test -bench=. -benchmem -run=^$ -benchtime=3s -count=1
```

## Interpret

Results are collated in [`../docs/BENCHMARK.md`](../docs/BENCHMARK.md). When updating that file, regenerate on the same hardware and paste the raw output section headers exactly.

## Update pinned versions

```sh
go get -u ./...
go mod tidy
```

Then re-run and update `../docs/BENCHMARK.md`.
