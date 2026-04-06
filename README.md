# Go Money
There is an ISO Standard for Currency code: ISO 4217. This library is following 
the same standard.


## Avoiding The Cardinal Sin: Floating Point Arithmetic
Using `float64` for monetary values is the single biggest anti-pattern in financial software. 
Floating-point numbers cannot accurately represent base-10 fractions. You will encounter 
precision loss (e.g., `0.1 + 0.2 = 0.30000000000000004`).

We store money as an `int64` representing the minor unit (e.g., cents for USD, zero decimal 
units for JPY). $100.50 USD is stored as 10050.


#### Credit
Copyright &copy; 2026 Syniol Limited. All rights reserved.
