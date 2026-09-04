package money

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/text/currency"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// AsDecimalString renders the amount as an ASCII decimal string. Use it for
// display or transport; never for arithmetic, which must stay on Minor().
// A zero-value Money (no currency) renders as the empty string so accidental
// use in logs is visually obvious rather than looking like a real "0".
func (m Money) AsDecimalString() string {
	if m.currency == nil {
		return ""
	}
	if m.currency.decimals == 0 {
		return strconv.FormatInt(m.amount, 10)
	}
	val := m.amount
	isNeg := val < 0
	var uval uint64
	if isNeg {
		uval = uint64(-val)
	} else {
		uval = uint64(val)
	}
	decimals := m.currency.decimals
	// |MaxInt64| is 19 digits, plus one '.' and one '-' sign gives 21 bytes.
	// Buffer to 24 for a small safety margin.
	var buf [24]byte
	n := len(buf)
	for i := 0; i < decimals; i++ {
		n--
		buf[n] = byte('0' + (uval % 10))
		uval /= 10
	}
	n--
	buf[n] = '.'
	if uval == 0 {
		n--
		buf[n] = '0'
	} else {
		for uval > 0 {
			n--
			buf[n] = byte('0' + (uval % 10))
			uval /= 10
		}
	}
	if isNeg {
		n--
		buf[n] = '-'
	}
	return string(buf[n:])
}

// String returns the currency symbol followed by the decimal string, or the
// empty string for a zero-value Money so logs never quietly render "0"
// without a currency.
func (m Money) String() string {
	if m.currency == nil {
		return ""
	}
	return m.currency.symbol + m.AsDecimalString()
}

// Format implements fmt.Formatter so %v, %s, %+v, %#v and %q print
// consistently and never leak the raw unexported struct layout.
//
//   - %s and %v render the same as String (e.g. "$10.50")
//   - %+v renders as "money.Money{amount:1050 currency:USD}" for logs
//     that want a structured view without leaking pointers
//   - %#v renders the same as %+v for Go-syntax debugging
//   - %q wraps String output in double quotes
//
// Unknown verbs fall back to a "%!verb(money.Money=...)" marker so an
// accidental %d does not silently print nothing.
func (m Money) Format(f fmt.State, verb rune) {
	switch verb {
	case 's', 'v':
		if f.Flag('+') || f.Flag('#') {
			fmt.Fprintf(f, "money.Money{amount:%d currency:%s}", m.amount, m.Currency())
			return
		}
		fmt.Fprint(f, m.String())
	case 'q':
		fmt.Fprintf(f, "%q", m.String())
	default:
		fmt.Fprintf(f, "%%!%c(money.Money=%s)", verb, m.String())
	}
}

// SymbolStyle selects which currency form LocalisedString substitutes into
// the CLDR template. Passing no option defaults to SymbolStyleNarrow to
// preserve the historical output.
type SymbolStyle int

const (
	// SymbolStyleNarrow uses the shortest locale-appropriate symbol
	// (e.g. "$" for USD everywhere).
	SymbolStyleNarrow SymbolStyle = iota
	// SymbolStyleStandard uses the standard locale symbol
	// (e.g. "US$" for USD in some locales).
	SymbolStyleStandard
	// SymbolStyleISO uses the ISO 4217 code (e.g. "USD 1,234.56").
	SymbolStyleISO
)

// LocalisedString renders the amount using CLDR conventions for the given
// language tag, preserving exact digits without going through float64.
// NBSP and other non-breaking spaces produced by CLDR are preserved: they are
// legitimate thousands separators in French, Russian, Swedish and others.
// Pass a SymbolStyle to change which currency form is used.
//
// Performance: LocalisedString is not zero-allocation. It builds several
// small strings via message.Printer (locale-aware separator sniffing plus
// integer, fractional and CLDR template rendering) and typically allocates
// on the order of half a dozen small buffers per call. Use it for display,
// not for hot logging paths; for allocation-free rendering, use String.
func (m Money) LocalisedString(tag language.Tag, opts ...SymbolStyle) string {
	if m.currency == nil {
		return ""
	}
	p := message.NewPrinter(tag)
	cur, err := currency.ParseISO(m.currency.isoCode)
	if err != nil {
		return m.String()
	}
	style := SymbolStyleNarrow
	if len(opts) > 0 {
		style = opts[0]
	}
	numberStr := formatLocalisedNumber(p, m.amount, m.currency.decimals)
	return applyCLDRTemplate(p, cur, m.currency, m.amount < 0, style, numberStr)
}

// formatLocalisedNumber returns the amount rendered with the locale's
// decimal separator, without a currency symbol. Negative amounts are
// prefixed with '-'.
func formatLocalisedNumber(p *message.Printer, amount int64, decimals int) string {
	isNeg := amount < 0
	uval := uint64(amount)
	if isNeg {
		uval = uint64(-amount)
	}
	var numberStr string
	if decimals == 0 {
		numberStr = p.Sprintf("%d", uval)
	} else {
		divisor := uint64(getPow10(decimals))
		intPart := uval / divisor
		fracPart := uval % divisor
		// Sniff the locale's decimal separator by asking Printer to format a
		// known value; the middle rune of "0<sep>0" is the separator.
		sample := p.Sprintf("%.1f", 0.0)
		decSep := "."
		if len(sample) >= 3 {
			decSep = sample[1 : len(sample)-1]
		}
		fracStr := fmt.Sprintf(fmt.Sprintf("%%0%dd", decimals), fracPart)
		numberStr = p.Sprintf("%d", intPart) + decSep + fracStr
	}
	if isNeg {
		numberStr = "-" + numberStr
	}
	return numberStr
}

// applyCLDRTemplate substitutes our exact numberStr into the CLDR currency
// template for the chosen SymbolStyle. If the CLDR template has no unique
// placeholder to substitute into (unlikely, but possible for exotic
// locales) it falls back to symbol + number so the caller always gets a
// non-empty rendering.
func applyCLDRTemplate(p *message.Printer, cur currency.Unit, c *Currency, negative bool, style SymbolStyle, numberStr string) string {
	sampleAmount := 1.0
	if negative {
		sampleAmount = -1.0
	}
	amt := cur.Amount(sampleAmount)
	var sampleTemplate string
	switch style {
	case SymbolStyleStandard:
		sampleTemplate = p.Sprint(amt)
	case SymbolStyleISO:
		sampleTemplate = p.Sprint(currency.ISO(amt))
	default:
		sampleTemplate = p.Sprint(currency.NarrowSymbol(amt))
	}
	samplePlaceholder := p.Sprintf("%.*f", c.decimals, sampleAmount)
	if strings.Count(sampleTemplate, samplePlaceholder) == 1 {
		return strings.Replace(sampleTemplate, samplePlaceholder, numberStr, 1)
	}
	return c.symbol + numberStr
}
