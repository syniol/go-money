package money

import (
	"testing"
)

func TestCurrencyConfigs(t *testing.T) {
	if len(currencyConfig) == 0 {
		t.Fatal("currencyConfig map should not be empty")
	}

	for code, cfg := range currencyConfig {
		if cfg.isoCode != code {
			t.Errorf("currency %s has mismatched ISOCode: %s", code, cfg.isoCode)
		}
		if cfg.decimals < 0 || cfg.decimals > MaxSafeDecimals {
			t.Errorf("currency %s has invalid decimals: %d", code, cfg.decimals)
		}
		if cfg.isoDigits < 0 {
			t.Errorf("currency %s has negative ISODigits: %d", code, cfg.isoDigits)
		}
		if cfg.name == "" {
			t.Errorf("currency %s has empty Name", code)
		}
	}
}

func TestSpecificISOCurrencies(t *testing.T) {
	testCases := []struct {
		code         string
		expectedNum  int
		expectedDec  int
		stringInput  string
		expectedUnit int64
	}{
		{"USD", 840, 2, "10.50", 1050},
		{"EUR", 978, 2, "25.00", 2500},
		{"JPY", 392, 0, "500", 500},
		{"KRW", 410, 0, "1000", 1000},
		{"VED", 926, 2, "15.75", 1575},
		{"SLE", 925, 2, "100.50", 10050},
		{"ZWG", 924, 2, "50.25", 5025},
		{"XCG", 532, 2, "12.34", 1234},
		{"CLF", 990, 4, "1.2345", 12345},
		{"XAU", 959, 0, "10", 10},
	}

	for _, tc := range testCases {
		t.Run(tc.code, func(t *testing.T) {
			cfg, exists := currencyConfig[tc.code]
			if !exists {
				t.Fatalf("currency %s does not exist in configuration", tc.code)
			}
			if cfg.isoNum != tc.expectedNum {
				t.Errorf("%s ISONum = %d, want %d", tc.code, cfg.isoNum, tc.expectedNum)
			}
			if cfg.decimals != tc.expectedDec {
				t.Errorf("%s Decimals = %d, want %d", tc.code, cfg.decimals, tc.expectedDec)
			}

			m, err := NewFromString(tc.stringInput, tc.code)
			if err != nil {
				t.Fatalf("NewFromString(%q, %q) returned error: %v", tc.stringInput, tc.code, err)
			}
			if m.Minor() != tc.expectedUnit {
				t.Errorf("%s minor unit = %d, want %d", tc.code, m.Minor(), tc.expectedUnit)
			}
		})
	}
}

func TestGetPow10_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("getPow10 should panic on out-of-range scale")
		}
	}()
	_ = getPow10(99)
}

func TestCurrency_HasISONum(t *testing.T) {
	// USD has a real ISO num, so HasISONum must be true.
	if !currencyConfig["USD"].HasISONum() {
		t.Error("USD.HasISONum() = false, want true")
	}
	// Any dataset entry that HasISONum reports as unknown must equal the
	// sentinel exactly; otherwise the generator lost a distinction.
	for _, c := range currencyConfig {
		if !c.HasISONum() && c.isoNum != ISONumUnknown {
			t.Errorf("%s: HasISONum false but ISONum = %d, want ISONumUnknown", c.isoCode, c.isoNum)
		}
	}
	// Synthetic construction: guarantee the sentinel path is exercised even
	// when the current generated dataset happens to have no nulls.
	syntheticUnknown := &Currency{isoCode: "XXX", isoNum: ISONumUnknown, numToBasic: NumToBasicUnknown}
	if syntheticUnknown.HasISONum() {
		t.Error("synthetic Currency with ISONumUnknown reported HasISONum = true")
	}
	if syntheticUnknown.HasNumToBasic() {
		t.Error("synthetic Currency with NumToBasicUnknown reported HasNumToBasic = true")
	}
	syntheticKnown := &Currency{isoCode: "YYY", isoNum: 999, numToBasic: 100}
	if !syntheticKnown.HasISONum() {
		t.Error("synthetic Currency with real ISONum reported HasISONum = false")
	}
	if !syntheticKnown.HasNumToBasic() {
		t.Error("synthetic Currency with real NumToBasic reported HasNumToBasic = false")
	}
}

func TestGetCurrency(t *testing.T) {
	usd, ok := GetCurrency("USD")
	if !ok {
		t.Fatal("GetCurrency(USD) not found")
	}
	if usd.ISOCode() != "USD" || usd.Symbol() != "$" || usd.Decimals() != 2 {
		t.Errorf("USD accessors returned wrong values: %s %s %d", usd.ISOCode(), usd.Symbol(), usd.Decimals())
	}
	if _, ok := GetCurrency("ZZZ"); ok {
		t.Error("GetCurrency(ZZZ) should not exist")
	}
}

func TestCurrencies(t *testing.T) {
	codes := Currencies()
	if len(codes) == 0 {
		t.Fatal("Currencies() returned empty")
	}
	for i := 1; i < len(codes); i++ {
		if codes[i-1] >= codes[i] {
			t.Fatalf("codes not sorted at index %d: %q >= %q", i, codes[i-1], codes[i])
		}
	}
	// mutation should not affect the internal map
	codes[0] = "MUTATED"
	fresh := Currencies()
	if fresh[0] == "MUTATED" {
		t.Error("Currencies() returned shared slice; caller mutation leaked back")
	}
}
