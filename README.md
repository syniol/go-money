# Go Money
This package is production-ready for a mainstream banking core. It treats money as a 
mathematical primitive rather than a data structure, ensuring immutability and stack-allocation 
throughout the entire transaction lifecycle. It utilises ISO Standard for Currency code: **ISO-4217**.


## Architectural Strengths
 * **Zero-Allocation Parsing:** NewFromString avoids strings.Split and strconv.ParseInt, using 
manual byte walking and a bytes.Buffer to keep objects on the stack and minimize Garbage Collection (GC) pressure.

 * **Hardened Arithmetic:** All operations (`Add`, `Sub`, `Mul`) include explicit overflow checks 
against `math.MaxInt64` to prevent silent financial errors.

 * **Mixed Receiver Strategy:** By using value receivers for domain logic and pointer receivers 
for UnmarshalJSON, you mimic the Go standard library’s time.Time pattern. This ensures the object 
remains a "Value Object" while satisfying interface requirements.


### Avoiding The Cardinal Sin: Floating Point Arithmetic
Using `float64` for monetary values is the single biggest anti-pattern in financial software. 
Floating-point numbers cannot accurately represent base-10 fractions. You will encounter 
precision loss (e.g., `0.1 + 0.2 = 0.30000000000000004`).

We store money as an `int64` representing the minor unit (e.g., cents for USD, zero decimal 
units for JPY). $100.50 USD is stored as 10050.


### Architectural Guardrail: The "Max Value"
An `int64` minor unit for USD can hold up to `$92,233,720,368,547,758.07`.
For almost any standard financial application (even national budgets), this is 
more than enough. However, if you are building a system for high-volume crypto 
(like SHIB or other tokens with 18 decimals), `int64` will overflow at very small 
dollar amounts.


#### Credit
Copyright &copy; 2026 Syniol Limited. All rights reserved.
