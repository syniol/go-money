package money

import "fmt"

// Currency holds immutable ISO 4217 metadata for a supported currency.
// Instances are populated once by the code generator and shared by pointer;
// callers must treat them as read-only.
type Currency struct {
	ISOCode      string
	Name         string
	Demonym      string
	MajorSingle  string
	MajorPlural  string
	ISONum       int
	Symbol       string
	SymbolNative string
	MinorSingle  string
	MinorPlural  string
	ISODigits    int
	Decimals     int
	NumToBasic   int
}

const (
	// MaxSafeDecimals is the largest decimal precision that fits in int64
	// arithmetic without risking overflow on typical monetary amounts.
	MaxSafeDecimals = 12

	// MaxStringLength caps NewFromString input to prevent pathological input.
	MaxStringLength = 64

	// MaxSplitParts caps Split fan-out to prevent memory exhaustion.
	MaxSplitParts = 10000
)

var pow10 = [...]int64{
	1, 10, 100, 1000, 10000, 100000, 1000000, 10000000,
	100000000, 1000000000, 10000000000, 100000000000, 1000000000000,
}

func getPow10(n int) int64 {
	if n >= 0 && n < len(pow10) {
		return pow10[n]
	}
	panic(fmt.Sprintf("money: scale %d exceeds maximum supported scale %d", n, MaxSafeDecimals))
}

// validateScale is defence in depth against a broken code generator; the
// generated currencyConfig should never contain an unsafe scale in practice.
func validateScale(c *Currency) error {
	if c.Decimals > MaxSafeDecimals {
		return ErrUnsafeScale
	}
	return nil
}
