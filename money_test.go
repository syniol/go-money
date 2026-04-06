package money

import "testing"

func TestParseInt(t *testing.T) {
	sut, _ := parseInt(12, &Config{Decimals: 12})
	t.Log(sut.value)
}

func TestNew(t *testing.T) {
	sut, _ := New(122323, "GBP")

	negativeAmount, _ := New(-12.969, "GBP")
	t.Log(negativeAmount.value)

	ise, _ := sut.IsEqual(12.97)
	t.Log(ise)

	isge, _ := sut.IsEqualGreater(12.96)
	t.Log(isge)

	formatted := sut.Formatted()
	t.Log(formatted)
}
