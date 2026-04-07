# Contributing to Go Money
By moving the currency data from a runtime-parsed JSON file to a generated Go file, 
we achieve three things:

 * Zero Startup Latency: No more `init()` functions blocking your service startup while it decodes 
thousands of lines of JSON.

 * Reduced Binary Bloat: We remove the `encoding/json` dependency from your production binary 
(if not used elsewhere) and eliminate the need for the `//go:embed` filesystem overhead.

 * Compile-Time Safety: Your currency data is now hardcoded as static Go structures. No more runtime 
panics because someone messed up a comma in a JSON file.

> The `//go:generate` line makes this a first-class Go citizen. Anyone working on this repo can 
simply run `go generate ./...` to update the currency definitions if the ISO spec changes.
