package money

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Money struct {
	value  float64
	config *Config
}

type Config struct {
	Name         string `json:"name"`
	Demonym      string `json:"demonym"`
	MajorSingle  string `json:"majorSingle"`
	MajorPlural  string `json:"majorPlural"`
	ISONum       int    `json:"ISOnum"`
	ISOCode      string
	Symbol       string `json:"symbol"`
	SymbolNative string `json:"symbolNative"`
	MinorSingle  string `json:"minorSingle"`
	MinorPlural  string `json:"minorPlural"`
	ISODigits    int    `json:"ISOdigits"`
	Decimals     int    `json:"decimals"`
	NumToBasic   int    `json:"numToBasic"`
}

//go:embed iso-4217.json
var currencyCodes []byte

var currencyConfig map[string]*Config

func init() {
	err := json.NewDecoder(bytes.NewReader(currencyCodes)).Decode(&currencyConfig)
	if err != nil {
		panic(err)
	}

	for isoCurrencyCode, cfg := range currencyConfig {
		cfg.ISOCode = isoCurrencyCode
	}
}

func New(amount interface{}, currencyCode string) (*Money, error) {
	if len(currencyCode) != 3 {
		return nil, fmt.Errorf("invalid currency code: %s", currencyCode)
	}

	cfg, exists := currencyConfig[strings.ToUpper(currencyCode)]
	if !exists {
		return nil, fmt.Errorf("currencyCode %s not supported", currencyCode)
	}

	switch reflect.TypeOf(amount).Kind() {
	case reflect.Float64:
		money, err := parseFloat(amount.(float64), cfg)
		if err != nil {
			return nil, err
		}
		return money, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		money, err := parseInt(amount.(int), cfg)
		if err != nil {
			return nil, err
		}
		return money, nil

	default:
		return nil, fmt.Errorf("unsupported amount type: %v", amount)
	}
}

func (m *Money) Value() float64 {
	return m.value
}

func (m *Money) String() string {
	switch m.config.Decimals {
	case 2:
		return fmt.Sprintf("%.2f", m.value)
	case 3:
		return fmt.Sprintf("%.3f", m.value)
	default:
		return fmt.Sprintf("%f", m.value)
	}
}

func (m *Money) IsEqual(amount interface{}) (bool, error) {
	comparableMoney, err := New(amount, m.config.ISOCode)
	if err != nil {
		return false, err
	}

	return m.value == comparableMoney.value, nil
}

func (m *Money) IsEqualGreater(amount interface{}) (bool, error) {
	comparableMoney, err := New(amount, m.config.ISOCode)
	if err != nil {
		return false, err
	}

	return m.value <= comparableMoney.value, nil

}

func (m *Money) IsLess(amount interface{}) (bool, error) {
	comparableMoney, err := New(amount, m.config.ISOCode)
	if err != nil {
		return false, err
	}

	return m.value > comparableMoney.value, nil
}

func (m *Money) IsEqualLess(amount interface{}) (bool, error) {
	comparableMoney, err := New(amount, m.config.ISOCode)
	if err != nil {
		return false, err
	}

	return m.value >= comparableMoney.value, nil
}

func (m *Money) Formatted() string {
	switch m.config.Decimals {
	case 1:
		return fmt.Sprintf("%s%.1f", m.config.Symbol, m.value)
	case 2:
		return fmt.Sprintf("%s%.2f", m.config.Symbol, m.value)
	case 3:
		return fmt.Sprintf("%s%.3f", m.config.Symbol, m.value)
	default:
		return fmt.Sprintf("%s%f", m.config.Symbol, m.value)
	}
}

func parseInt(amount int, cfg *Config) (*Money, error) {
	amountWithDecimal := fmt.Sprintf("%d.00", amount)
	value, err := strconv.ParseFloat(amountWithDecimal, 64)
	if err != nil {
		return nil, err
	}

	return &Money{value: value, config: cfg}, nil
}

func parseFloat(amount float64, cfg *Config) (*Money, error) {
	amountWithDecimal := fmt.Sprintf("%.2f", amount)
	value, err := strconv.ParseFloat(amountWithDecimal, 64)
	if err != nil {
		return nil, err
	}

	return &Money{value: value, config: cfg}, nil
}
