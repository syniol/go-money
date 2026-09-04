package money

import "fmt"

// Currency holds immutable ISO 4217 metadata for a supported currency.
// Instances are populated once by the code generator and shared by pointer;
// callers must treat them as read-only.
//
// The MajorSingle, MajorPlural, MinorSingle and MinorPlural accessors expose
// raw display forms. This package intentionally does not ship a plural
// helper: correct pluralisation requires CLDR plural categories, which are
// handled properly by golang.org/x/text/feature/plural. A single-condition
// helper here would only be right for English and would mislead callers
// into shipping it elsewhere.
//
// The fields are unexported so a downstream caller cannot mutate the shared
// map values by accident. Read state via the accessor methods below.
type Currency struct {
	// Hot fields first, in access-frequency order. isoCode, decimals and
	// symbol are read on nearly every arithmetic, comparison or format
	// operation. Placing them at the head of the struct keeps them within
	// the same cache line on 64-bit CPUs.
	isoCode  string
	symbol   string
	decimals int

	// Cold ISO metadata.
	isoNum     int
	isoDigits  int
	numToBasic int

	// Cold display metadata used only by callers formatting locale-aware
	// strings outside the library's own hot paths.
	name         string
	demonym      string
	symbolNative string
	majorSingle  string
	majorPlural  string
	minorSingle  string
	minorPlural  string
}

// ISOCode returns the ISO 4217 three-letter code (e.g. "USD").
func (c *Currency) ISOCode() string { return c.isoCode }

// Name returns the human-readable currency name (e.g. "US Dollar").
func (c *Currency) Name() string { return c.name }

// Demonym returns the currency demonym (e.g. "American").
func (c *Currency) Demonym() string { return c.demonym }

// MajorSingle returns the singular name of the major unit (e.g. "dollar").
func (c *Currency) MajorSingle() string { return c.majorSingle }

// MajorPlural returns the plural name of the major unit (e.g. "dollars").
func (c *Currency) MajorPlural() string { return c.majorPlural }

// ISONum returns the ISO 4217 numeric code, or ISONumUnknown when the
// upstream source has no numeric code. Use HasISONum to distinguish.
func (c *Currency) ISONum() int { return c.isoNum }

// Symbol returns the ASCII-safe currency symbol (e.g. "$").
func (c *Currency) Symbol() string { return c.symbol }

// SymbolNative returns the native-script symbol.
func (c *Currency) SymbolNative() string { return c.symbolNative }

// MinorSingle returns the singular name of the minor unit (e.g. "cent").
func (c *Currency) MinorSingle() string { return c.minorSingle }

// MinorPlural returns the plural name of the minor unit (e.g. "cents").
func (c *Currency) MinorPlural() string { return c.minorPlural }

// ISODigits returns the number of decimal digits per ISO 4217.
func (c *Currency) ISODigits() int { return c.isoDigits }

// Decimals returns the number of decimal places this library uses when
// parsing and formatting amounts.
func (c *Currency) Decimals() int { return c.decimals }

// NumToBasic returns the conversion factor from minor to major unit, or
// NumToBasicUnknown when the upstream source has no factor. Use
// HasNumToBasic to distinguish.
func (c *Currency) NumToBasic() int { return c.numToBasic }

const (
	// MaxSafeDecimals is the largest decimal precision that fits in int64
	// arithmetic without risking overflow on typical monetary amounts.
	//
	// int64 holds values up to math.MaxInt64 = 9223372036854775807, which is
	// 19 decimal digits. Reserving 12 digits for the fractional part leaves
	// 7 digits (up to 9999999) of major-unit headroom, which comfortably
	// covers every real currency's practical range. Currencies with more
	// than 12 decimal places are rejected at package init.
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
	// upstream ISO source has no conversion factor. Distinct value from
	// ISONumUnknown so accidental cross-comparison fails loudly rather
	// than passing silently.
	NumToBasicUnknown = -2
)

// HasISONum reports whether ISONum holds a real ISO 4217 numeric code
// rather than the ISONumUnknown sentinel.
func (c *Currency) HasISONum() bool { return c.isoNum != ISONumUnknown }

// HasNumToBasic reports whether NumToBasic holds a real conversion factor
// rather than the NumToBasicUnknown sentinel.
func (c *Currency) HasNumToBasic() bool { return c.numToBasic != NumToBasicUnknown }

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

// init runs the defence-in-depth scale check once at package load rather
// than on every New call. A broken code generator will panic here instead
// of returning ErrUnsafeScale to every caller.
func init() {
	for _, c := range currencyConfig {
		if c.decimals > MaxSafeDecimals {
			panic(fmt.Sprintf("money: currency %s has Decimals=%d exceeding MaxSafeDecimals=%d",
				c.isoCode, c.decimals, MaxSafeDecimals))
		}
	}
}
