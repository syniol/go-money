# sqltest

End-to-end tests for `Money.Value`, `Money.Scan` and `NullMoney` against a real SQL driver (`modernc.org/sqlite`, pure Go, no CGO).

Kept in a separate module so the SQLite driver never appears in the `go.sum` of consumers of the root `github.com/syniol/go-money` module. A `replace` directive points at `../` for local development and CI.

## Run

```sh
cd sqltest
go test ./...
```

## Adding a driver

To exercise the codec against Postgres or MySQL in addition to SQLite, add a driver here (not in the root module) and mirror the tests. A future pgx `NUMERIC` codec sub-module (`pgxmoney/`) will live alongside this one.
