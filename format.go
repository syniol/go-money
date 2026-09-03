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
func (m Money) AsDecimalString() string {
	if m.currency == nil {
		return "0"
	}
	if m.currency.Decimals == 0 {
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
	decimals := m.currency.Decimals
	// 20 digits for MaxUint64 + 1 dot + 1 sign = 22; buffer to 24 for margin.
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

// String returns the currency symbol followed by the decimal string.
func (m Money) String() string {
	if m.currency == nil {
		return "0"
	}
	return m.currency.Symbol + m.AsDecimalString()
}

// LocalisedString renders the amount using CLDR conventions for the given
// language tag, preserving exact digits without going through float64.
// NBSP and other non-breaking spaces produced by CLDR are preserved: they are
// legitimate thousands separators in French, Russian, Swedish and others.
func (m Money) LocalisedString(tag language.Tag) string {
	if m.currency == nil {
		return "0"
	}
	p := message.NewPrinter(tag)
	cur, err := currency.ParseISO(m.currency.ISOCode)
	if err != nil {
		return m.String()
	}
	val := m.amount
	isNeg := val < 0
	var uval uint64
	if isNeg {
		uval = uint64(-val)
	} else {
		uval = uint64(val)
	}
	var numberStr string
	if m.currency.Decimals == 0 {
		numberStr = p.Sprintf("%d", uval)
	} else {
		divisor := uint64(getPow10(m.currency.Decimals))
		intPart := uval / divisor
		fracPart := uval % divisor
		sample := p.Sprintf("%.1f", 0.0)
		decSep := "."
		if len(sample) >= 3 {
			decSep = sample[1 : len(sample)-1]
		}
		fracFormat := fmt.Sprintf("%%0%dd", m.currency.Decimals)
		fracStr := fmt.Sprintf(fracFormat, fracPart)
		numberStr = p.Sprintf("%d", intPart) + decSep + fracStr
	}
	if isNeg {
		numberStr = "-" + numberStr
	}
	var sampleAmount float64 = 1.0
	if isNeg {
		sampleAmount = -1.0
	}
	sampleTemplate := p.Sprint(currency.NarrowSymbol(cur.Amount(sampleAmount)))
	samplePlaceholder := p.Sprintf("%.*f", m.currency.Decimals, sampleAmount)
	if strings.Contains(sampleTemplate, samplePlaceholder) {
		return strings.Replace(sampleTemplate, samplePlaceholder, numberStr, 1)
	}
	return m.currency.Symbol + numberStr
}
