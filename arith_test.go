package money

import (
	"errors"
	"math"
	"testing"
)

func TestArithmetic_And_Comparisons(t *testing.T) {
	m10 := MustNew(1000, "USD")
	m5 := MustNew(500, "USD")

	// Addition
	res, _ := m10.Add(m5)
	if res.Minor() != 1500 {
		t.Error("Add failed")
	}

	// Subtraction
	res, _ = m10.Sub(m5)
	if res.Minor() != 500 {
		t.Error("Sub failed")
	}

	// Multiplication
	res, _ = m10.Mul(3)
	if res.Minor() != 3000 {
		t.Error("Mul failed")
	}

	// Comparisons via Compare and Equal
	lessCmp, _ := m5.Compare(m10)
	greaterCmp, _ := m10.Compare(m5)
	if lessCmp != -1 || greaterCmp != 1 || !m10.Equal(m10) {
		t.Error("Comparison logic error")
	}
	if m10.Equal(m5) {
		t.Error("Equal returned true for unequal amounts")
	}

	// Compare method
	cmp, _ := m10.Compare(m5)
	if cmp != 1 {
		t.Errorf("Expected 1, got %d", cmp)
	}

	// Boolean checks
	if !MustNew(0, "USD").IsZero() {
		t.Error("IsZero failed")
	}
	if !m10.IsPositive() {
		t.Error("IsPositive failed")
	}
	if !MustNew(-10, "USD").IsNegative() {
		t.Error("IsNegative failed")
	}
}

func TestMul_NegativeMultiplier(t *testing.T) {
	m := MustNew(1000, "USD")

	// Multiplying positive amount by -1
	res, err := m.Mul(-1)
	if err != nil {
		t.Fatalf("m.Mul(-1) failed: %v", err)
	}
	if res.Minor() != -1000 {
		t.Errorf("got %d, want -1000", res.Minor())
	}

	// Multiplying negative amount by -1
	neg := MustNew(-1000, "USD")
	res, err = neg.Mul(-1)
	if err != nil {
		t.Fatalf("neg.Mul(-1) failed: %v", err)
	}
	if res.Minor() != 1000 {
		t.Errorf("got %d, want 1000", res.Minor())
	}

	// Multiplying by negative factor
	res, err = m.Mul(-5)
	if err != nil {
		t.Fatalf("m.Mul(-5) failed: %v", err)
	}
	if res.Minor() != -5000 {
		t.Errorf("got %d, want -5000", res.Minor())
	}

	// Zero multipliers
	res, err = m.Mul(0)
	if err != nil || res.Minor() != 0 {
		t.Errorf("m.Mul(0) failed")
	}

	zero := MustNew(0, "USD")
	res, err = zero.Mul(-1)
	if err != nil || res.Minor() != 0 {
		t.Errorf("zero.Mul(-1) failed")
	}
}

func TestMoney_Split_Bounds(t *testing.T) {
	m := MustNew(1000, "USD")

	// Valid split
	parts, err := m.Split(3)
	if err != nil || len(parts) != 3 {
		t.Fatalf("Split(3) failed: %v", err)
	}

	// Zero or negative splits
	_, err = m.Split(0)
	if !errors.Is(err, ErrInvalidSplitCount) {
		t.Errorf("Split(0) error = %v, want ErrInvalidSplitCount", err)
	}

	_, err = m.Split(-5)
	if !errors.Is(err, ErrInvalidSplitCount) {
		t.Errorf("Split(-5) error = %v, want ErrInvalidSplitCount", err)
	}

	// Exceeding MaxSplitParts (OOM prevention)
	_, err = m.Split(MaxSplitParts + 1)
	if !errors.Is(err, ErrInvalidSplitCount) {
		t.Errorf("Split(%d) error = %v, want ErrInvalidSplitCount", MaxSplitParts+1, err)
	}
}

func TestAdd_Overflow(t *testing.T) {
	maxUSD := MustNew(math.MaxInt64, "USD")
	one := MustNew(1, "USD")
	if _, err := maxUSD.Add(one); !errors.Is(err, ErrOverflow) {
		t.Errorf("MaxInt64 + 1 error = %v, want ErrOverflow", err)
	}
	minUSD := MustNew(math.MinInt64, "USD")
	negOne := MustNew(-1, "USD")
	if _, err := minUSD.Add(negOne); !errors.Is(err, ErrOverflow) {
		t.Errorf("MinInt64 + -1 error = %v, want ErrOverflow", err)
	}
}

func TestSub_Overflow(t *testing.T) {
	minUSD := MustNew(math.MinInt64, "USD")
	one := MustNew(1, "USD")
	if _, err := minUSD.Sub(one); !errors.Is(err, ErrOverflow) {
		t.Errorf("MinInt64 - 1 error = %v, want ErrOverflow", err)
	}
	maxUSD := MustNew(math.MaxInt64, "USD")
	negOne := MustNew(-1, "USD")
	if _, err := maxUSD.Sub(negOne); !errors.Is(err, ErrOverflow) {
		t.Errorf("MaxInt64 - -1 error = %v, want ErrOverflow", err)
	}
}

func TestMul_Overflow(t *testing.T) {
	half := MustNew(math.MaxInt64/2+1, "USD")
	if _, err := half.Mul(2); !errors.Is(err, ErrOverflow) {
		t.Errorf("(MaxInt64/2+1) * 2 error = %v, want ErrOverflow", err)
	}
	minUSD := MustNew(math.MinInt64, "USD")
	if _, err := minUSD.Mul(-1); !errors.Is(err, ErrOverflow) {
		t.Errorf("MinInt64 * -1 error = %v, want ErrOverflow", err)
	}
}

func TestArithmetic_CurrencyMismatch(t *testing.T) {
	usd := MustNew(100, "USD")
	eur := MustNew(100, "EUR")
	if _, err := usd.Add(eur); !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("Add across currencies error = %v, want ErrCurrencyMismatch", err)
	}
	if _, err := usd.Sub(eur); !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("Sub across currencies error = %v, want ErrCurrencyMismatch", err)
	}
	if _, err := usd.Compare(eur); !errors.Is(err, ErrCurrencyMismatch) {
		t.Errorf("Compare across currencies error = %v, want ErrCurrencyMismatch", err)
	}
	if usd.Equal(eur) {
		t.Errorf("Equal across currencies returned true")
	}
}

func TestSplit_One(t *testing.T) {
	m := MustNew(1234, "USD")
	parts, err := m.Split(1)
	if err != nil {
		t.Fatalf("Split(1) failed: %v", err)
	}
	if len(parts) != 1 || parts[0].Minor() != 1234 {
		t.Errorf("Split(1) = %v, want a single unchanged part", parts)
	}
}

func BenchmarkAdd(b *testing.B) {
	m1 := MustNew(1000, "USD")
	m2 := MustNew(500, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m1.Add(m2)
	}
}

func BenchmarkSub(b *testing.B) {
	m1 := MustNew(1000, "USD")
	m2 := MustNew(500, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m1.Sub(m2)
	}
}

func BenchmarkMul(b *testing.B) {
	m := MustNew(1000, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.Mul(3)
	}
}

func TestEqual_ZeroValues(t *testing.T) {
	var a, b Money
	if !a.Equal(b) {
		t.Error("two zero-value Moneys should be equal, matching time.Time.Equal")
	}
	usd := MustNew(0, "USD")
	if a.Equal(usd) || usd.Equal(a) {
		t.Error("zero-value Money should not equal a real currency-carrying zero")
	}
}

func TestCmp_PanicsOnMismatch(t *testing.T) {
	usd := MustNew(100, "USD")
	eur := MustNew(100, "EUR")
	defer func() {
		if r := recover(); r == nil {
			t.Error("Cmp with mismatched currencies should panic")
		}
	}()
	_ = usd.Cmp(eur)
}

func TestCmp_ReturnsCompareOrder(t *testing.T) {
	a := MustNew(100, "USD")
	b := MustNew(200, "USD")
	if a.Cmp(b) != -1 || b.Cmp(a) != 1 || a.Cmp(a) != 0 {
		t.Error("Cmp ordering wrong")
	}
}

func TestNeg(t *testing.T) {
	pos := MustNew(1000, "USD")
	neg, err := pos.Neg()
	if err != nil || neg.Minor() != -1000 {
		t.Errorf("Neg(1000 USD) = (%v, %v)", neg.Minor(), err)
	}
	roundTrip, err := neg.Neg()
	if err != nil || roundTrip.Minor() != 1000 {
		t.Errorf("Neg(-1000 USD) = (%v, %v)", roundTrip.Minor(), err)
	}
	min := MustNew(math.MinInt64, "USD")
	if _, err := min.Neg(); !errors.Is(err, ErrOverflow) {
		t.Errorf("Neg(MinInt64 USD) err = %v, want ErrOverflow", err)
	}
	var zero Money
	got, err := zero.Neg()
	if err != nil || got.Valid() {
		t.Errorf("Neg(zero-value) = (%v, %v)", got, err)
	}
}

func TestAbs(t *testing.T) {
	neg := MustNew(-1000, "USD")
	abs, err := neg.Abs()
	if err != nil || abs.Minor() != 1000 {
		t.Errorf("Abs(-1000 USD) = (%v, %v)", abs.Minor(), err)
	}
	pos := MustNew(1000, "USD")
	if v, err := pos.Abs(); err != nil || v.Minor() != 1000 {
		t.Errorf("Abs(1000 USD) unchanged = (%v, %v)", v.Minor(), err)
	}
	min := MustNew(math.MinInt64, "USD")
	if _, err := min.Abs(); !errors.Is(err, ErrOverflow) {
		t.Errorf("Abs(MinInt64 USD) err = %v, want ErrOverflow", err)
	}
}
