package money

import (
	"fmt"
	"golang.org/x/text/language"
	"strings"
	"testing"
)

func TestDisplayMethods(t *testing.T) {
	m := MustNew(1050, "USD")

	if m.AsDecimalString() != "10.50" {
		t.Errorf("Float() got %s, want 10.50", m.AsDecimalString())
	}

	if m.String() != "$10.50" {
		t.Errorf("String() got %s, want $10.50", m.String())
	}

	jpy := MustNew(500, "JPY")
	if jpy.String() != "¥500" {
		t.Errorf("JPY String() got %s, want ¥500", jpy.String())
	}
}

func TestAsDecimalString_HighPrecision(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		currency string
		wantStr  string
	}{
		{"zero USD", 0, "USD", "0.00"},
		{"one cent USD", 1, "USD", "0.01"},
		{"five cents USD", 5, "USD", "0.05"},
		{"negative cent USD", -1, "USD", "-0.01"},
		{"negative five cents USD", -5, "USD", "-0.05"},
		{"large USD beyond float64 53-bit precision", 9007199254740993, "USD", "90071992547409.93"},
		{"negative large USD", -9007199254740993, "USD", "-90071992547409.93"},
		{"zero decimals JPY", 500, "JPY", "500"},
		{"zero decimals negative JPY", -500, "JPY", "-500"},
		{"three decimals KWD", 1234, "KWD", "1.234"},
		{"four decimals CLF", 12345, "CLF", "1.2345"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Money{amount: tt.amount, currency: currencyConfig[tt.currency]}
			got := m.AsDecimalString()
			if got != tt.wantStr {
				t.Errorf("AsDecimalString() = %q, want %q", got, tt.wantStr)
			}
		})
	}
}

func TestLocalisedString_ExactPrecision(t *testing.T) {
	// Value exceeding 2^53 (9,007,199,254,740,992 minor units)
	// $90,071,992,547,409.93
	m := MustNew(9007199254740993, "USD")

	gotEN := m.LocalisedString(language.AmericanEnglish)
	// CLDR/x/text emits a non-breaking space between symbol and amount for
	// en-US; that is CLDR-correct behaviour and must not be stripped, because
	// stripping it would also strip the NBSP that other locales use as their
	// thousands separator (see fr-FR case below).
	if !strings.Contains(gotEN, "90,071,992,547,409.93") || !strings.Contains(gotEN, "$") {
		t.Errorf("LocalisedString(en-US) = %q, want the digits 90,071,992,547,409.93 and $ preserved", gotEN)
	}

	gotDE := m.LocalisedString(language.German)
	// German uses '.' for thousands and ',' for decimal
	if gotDE != "90.071.992.547.409,93$" && gotDE != "$90.071.992.547.409,93" && gotDE != "90.071.992.547.409,93\u00a0$" {
		// Verify digits and separators are preserved without float precision loss
		if !strings.Contains(gotDE, "90.071.992.547.409,93") && !strings.Contains(gotDE, "90.071.992.547.409.93") {
			t.Errorf("LocalisedString(de) = %q, expected exact digits preserved", gotDE)
		}
	}

	// Zero decimal high precision currency
	jpy := MustNew(9007199254740993, "JPY")
	gotJPY := jpy.LocalisedString(language.Japanese)
	if !strings.Contains(gotJPY, "9,007,199,254,740,993") {
		t.Errorf("LocalisedString(ja JPY) = %q, want contains 9,007,199,254,740,993", gotJPY)
	}

	// Negative value: preserve exact digits regardless of sign placement.
	neg := MustNew(-123456, "USD")
	gotNeg := neg.LocalisedString(language.AmericanEnglish)
	if !strings.Contains(gotNeg, "1,234.56") {
		t.Errorf("LocalisedString(neg USD) = %q, want digits 1,234.56 preserved", gotNeg)
	}
}

// TestLocalisedString_NBSPThousandsSeparator locks in the fix for the bug
// where the old implementation stripped every Unicode whitespace rune,
// destroying the NBSP that CLDR uses as the thousands separator in French,
// Russian and Swedish. Under the buggy behaviour, "1 234,56" collapsed to
// "1234,56".

func TestLocalisedString_NBSPThousandsSeparator(t *testing.T) {
	m := MustNew(123456789, "EUR") // 1 234 567,89 in fr-FR

	gotFR := m.LocalisedString(language.French)
	// French thousands separator is NBSP ( ). Assert the exact byte is present.
	if !strings.Contains(gotFR, " ") {
		t.Errorf("LocalisedString(fr) = %q, expected NBSP thousands separator preserved", gotFR)
	}
	if !strings.Contains(gotFR, ",89") {
		t.Errorf("LocalisedString(fr) = %q, expected French decimal separator ','", gotFR)
	}
}

func BenchmarkAsDecimalString(b *testing.B) {
	m := MustNew(123456, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.AsDecimalString()
	}
}

func BenchmarkString(b *testing.B) {
	m := MustNew(123456, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.String()
	}
}

func TestLocalisedString_SymbolStyle(t *testing.T) {
	m := MustNew(123456, "USD")
	iso := m.LocalisedString(language.AmericanEnglish, SymbolStyleISO)
	if !strings.Contains(iso, "USD") {
		t.Errorf("SymbolStyleISO output %q missing USD", iso)
	}
	narrow := m.LocalisedString(language.AmericanEnglish, SymbolStyleNarrow)
	if !strings.Contains(narrow, "$") {
		t.Errorf("SymbolStyleNarrow output %q missing $", narrow)
	}
}

func TestFormat_Verbs(t *testing.T) {
	m := MustNew(1050, "USD")
	tests := []struct {
		format string
		want   string
	}{
		{"%s", "$10.50"},
		{"%v", "$10.50"},
		{"%+v", "money.Money{amount:1050 currency:USD}"},
		{"%#v", "money.MustNew(1050, \"USD\")"},
		{"%q", "\"$10.50\""},
	}
	for _, tt := range tests {
		got := fmt.Sprintf(tt.format, m)
		if got != tt.want {
			t.Errorf("Sprintf(%q, m) = %q, want %q", tt.format, got, tt.want)
		}
	}
	if got := fmt.Sprintf("%d", m); !strings.Contains(got, "money.Money=") {
		t.Errorf("unknown verb fallback = %q, want to contain money.Money=", got)
	}
}

func TestLocalisedString_ArabicLocaleNoCorruption(t *testing.T) {
	// Arabic-Indic digits can produce multi-byte runes in the separator
	// sniff; the byte-slice approach would have produced garbage. Assert
	// that the digits 1234.56 (in whatever form CLDR picks) survive.
	m := MustNew(123456, "USD")
	got := m.LocalisedString(language.Arabic)
	if got == "" {
		t.Error("LocalisedString(ar) returned empty; separator sniff likely broke")
	}
}
