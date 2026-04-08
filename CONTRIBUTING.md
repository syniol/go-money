# Contributing to Go-Money

Thank you for your interest in contributing to `go-money`. Because this library is designed for financial institutions and banking cores, we maintain a higher bar for contributions than typical open-source projects.

Accuracy, precision, and performance are our primary metrics.

---

## 🏗 Architectural Philosophy

Before submitting a Pull Request, please ensure your changes adhere to our core architectural guardrails:

1.  **Immutability:** The `Money` struct must remain a value object. Methods should return a new `Money` instance rather than modifying the receiver.
2.  **Stack Allocation:** Avoid any logic that forces the `Money` struct to escape to the heap. We aim for zero-allocation in the "hot paths" (arithmetic and parsing).
3.  **Overflow Safety:** All arithmetic operations must be explicitly checked against `math.MaxInt64`. Silent wrap-around is a critical failure.
4.  **No Floating Point:** Under no circumstances should `float64` be used for internal storage or intermediate arithmetic logic.

---

## 📜 Currency Data & Generation

We do not manually edit `currencies.gen.go`. This file is a static representation of the ISO-4217 standard derived from our JSON source.

If you need to update currency definitions:
1.  Update the source data in `iso-4217.json`.
2.  Run the generator:
    ```bash
    go generate ./...
    ```
3.  Ensure the generator logic in `cmd/gen_currencies/main.go` remains deterministic (e.g., keeping map keys sorted) to prevent noisy diffs.

---

## 🧪 Testing Standards

We require 100% code coverage for all public methods. Your PR will not be merged without:

### 1. Unit Tests
Every new feature must include table-driven tests in `money_test.go` covering:
* Positive and negative values.
* Zero-decimal currencies (e.g., JPY).
* Multi-decimal currencies (e.g., KWD).
* Edge cases (max/min `int64`).

### 2. Fuzz Testing
If you modify the parser (`NewFromString`), you **must** run the fuzzer to ensure no input can cause a panic:
```bash
go test -fuzz=FuzzNewFromString
```

### 3. Benchmarking
If you modify arithmetic or parsing logic, provide benchmark results to ensure we haven't introduced regressions 
in latency or allocations:

```bash
go test -bench=. -benchmem
```

---

## 🚀 Pull Request Process
 * **1. Issue First:** For significant changes, please open an issue to discuss the architectural impact before writing code.

 * **2. Linting:** Ensure your code passes golangci-lint.

 * **3. Commit Messages:** Use descriptive, imperative commit messages (e.g., feat: add support for custom rounding modes).

 * **4. Documentation:** Update README.md or example_test.go if you are adding new public-facing functionality.

---

## ⚖️ License
By contributing to this repository, you agree that your contributions will be licensed under the project's **BSD 3-Clause License**.
