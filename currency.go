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
	//
	// int64 holds values up to math.MaxInt64 = 9223372036854775807, which is
	// 19 decimal digits. Reserving 12 digits for the fractional part leaves
	// 7 digits (up to 9999999) of major-unit headroom, which comfortably
	// covers every real currency's practical range. Currencies with more
	// than 12 decimal places are refused by validateScale.
	MaxSafeDecimals = 12

	// MaxStringLength caps NewFromString input to guarantee bounded parse
	// time and reject pathological input.
	//
	// The widest legitimate amount is a signed 4-decimal currency at
	// MaxInt64 (e.g. CLF), which is "-922337203685477.5807", 22 bytes. 64
	// bytes gives ample room for extra leading zeros in the integer part
	// without ever admitting adversarial multi-kilobyte input.
	MaxStringLength = 64

	// MaxSplitParts caps Split fan-out to prevent memory exhaustion.
	//
	// A single Split call allocates a []Money slice of length n. At 16 bytes
	// per Money on 64-bit builds, 10000 parts is 160 KB, an amount an
	// attacker cannot use to exhaust memory even under repeated calls.
	MaxSplitParts = 10000

	// ISONumUnknown is the sentinel written to Currency.ISONum when the
	// upstream ISO 4217 source has no numeric code for that currency. Valid
	// ISO 4217 numeric codes are always positive (1..999), so a negative
	// value is unambiguously "unknown".
	ISONumUnknown = -1

	// NumToBasicUnknown is the sentinel for Currency.NumToBasic when the
	// upstream ISO source has no conversion factor.
	NumToBasicUnknown = -1
)

// HasISONum reports whether Currency.ISONum holds a real ISO 4217 numeric
// code rather than the ISONumUnknown sentinel.
func (c *Currency) HasISONum() bool { return c.ISONum != ISONumUnknown }

// HasNumToBasic reports whether Currency.NumToBasic holds a real conversion
// factor rather than the NumToBasicUnknown sentinel.
func (c *Currency) HasNumToBasic() bool { return c.NumToBasic != NumToBasicUnknown }

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

// PluralMajorUnit returns the singular or plural name of the currency's major
// unit for the given quantity (e.g. "dollar" for 1, "dollars" for 2, "0" or
// negative). English rules only; callers needing CLDR plural categories
// should format with golang.org/x/text/message instead.
func (c *Currency) PluralMajorUnit(qty int64) string {
	if qty == 1 || qty == -1 {
		return c.MajorSingle
	}
	return c.MajorPlural
}

// PluralMinorUnit is the minor-unit counterpart of PluralMajorUnit.
func (c *Currency) PluralMinorUnit(qty int64) string {
	if qty == 1 || qty == -1 {
		return c.MinorSingle
	}
	return c.MinorPlural
}
