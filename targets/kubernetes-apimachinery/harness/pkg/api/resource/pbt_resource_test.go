// Property-based tests for resource.Quantity parsing, serialisation and
// arithmetic.
//
// Lab: Agentic PBT Lab (targets/kubernetes-apimachinery) — pristine copy.
// run.sh copies this file into the apimachinery checkout at
// pkg/api/resource/pbt_resource_test.go and runs it in-package
// (package resource).
package resource

import (
	"math"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// quantityCharset is dense in the characters ParseQuantity treats
// specially: digits, signs, decimal points, exponent markers and every
// suffix letter.
const quantityCharset = "0123456789.eE+-nukmMGTPEi"

// adversarialQuantities are fixed tokens aimed at the paths the package
// docs call out explicitly: the silent BinarySI cap at 2^63-1, nano
// rounding of sub-nano values, >int64 decimal values, 29-significant-digit
// rounding, and exponent forms.
var adversarialQuantities = []string{
	"8Ei", "9Ei", "16Ei", "7Ei", "-8Ei",
	"0.9n", "0.1n", "1.5n", "0.0000000001", "0.000000001",
	"18446744073709551616", "9223372036854775807", "9223372036854775808",
	"-9223372036854775808", "123456789012345678901234567890123456789",
	"1.2345", "1e30", "-1e30", "1E-9", "1e-10", "1e100", "1e-100",
	"1.1Ki", "0.5Mi", "1.5Gi", "1024", "1000", "0", "-0", "0.000",
	"999999999999999999999999999999999999999Ki",
	"3.141592653589793238462643383279502884197",
}

// drawQuantityPart builds one structured quantity string: optional sign,
// 1-22 integer digits, optional fraction, and a suffix drawn from the
// documented grammar plus near-miss forms.
func drawQuantityPart(t *rapid.T) string {
	var b strings.Builder
	if rapid.Bool().Draw(t, "negative") {
		b.WriteByte('-')
	} else if rapid.Bool().Draw(t, "plus") {
		b.WriteByte('+')
	}
	intDigits := rapid.StringOfN(rapid.RuneFrom([]rune("0123456789")), 1, 22, -1).Draw(t, "intDigits")
	b.WriteString(intDigits)
	if rapid.Bool().Draw(t, "hasFraction") {
		b.WriteByte('.')
		b.WriteString(rapid.StringOfN(rapid.RuneFrom([]rune("0123456789")), 0, 12, -1).Draw(t, "fracDigits"))
	}
	suffix := rapid.SampledFrom([]string{
		"", "n", "u", "m", "k", "M", "G", "T", "P", "E",
		"Ki", "Mi", "Gi", "Ti", "Pi", "Ei",
		"e3", "e-3", "E6", "E-9", "e0", "e100", "e-100",
	}).Draw(t, "suffix")
	b.WriteString(suffix)
	return b.String()
}

func drawQuantityString(t *rapid.T) string {
	return rapid.OneOf(
		rapid.SampledFrom(adversarialQuantities),
		rapid.Custom(drawQuantityPart),
	).Draw(t, "qs")
}

// drawQuantity parses a drawn string; failing parses are skipped (the
// parse itself is exercised by the never-panics property).
func drawQuantity(t *rapid.T) Quantity {
	s := drawQuantityString(t)
	q, err := ParseQuantity(s)
	if err != nil {
		t.Skipf("unparsable generated input %q: %v", s, err)
	}
	return q
}

func sign(x int) int {
	switch {
	case x < 0:
		return -1
	case x > 0:
		return 1
	default:
		return 0
	}
}

// TestPBTParseQuantityNeverPanics: ParseQuantity consumes strings from
// API objects and user input; it must error, never panic.
func TestPBTParseQuantityNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.OneOf(
			rapid.String(),
			rapid.StringOf(rapid.RuneFrom([]rune(quantityCharset))),
			rapid.SampledFrom(adversarialQuantities),
		).Draw(t, "s")
		_, _ = ParseQuantity(s) // a panic here is the failure rapid reports
	})
}

// TestPBTQuantityStringRoundTrip: the package documents that String()
// emits a canonical form with "no precision lost", so re-parsing the
// printed form must yield an equal value, and printing that again must
// reproduce the same string (parse-print-parse stability).
func TestPBTQuantityStringRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		q := drawQuantity(t)
		s1 := q.String()
		q2, err := ParseQuantity(s1)
		if err != nil {
			t.Fatalf("String() = %q does not re-parse: %v", s1, err)
		}
		if c := q2.Cmp(q); c != 0 {
			t.Fatalf("round trip changed value: %q -> %q -> reparse compares %d", q.String(), s1, c)
		}
		if s2 := q2.String(); s2 != s1 {
			t.Fatalf("String() not stable: %q -> %q", s1, s2)
		}
	})
}

// TestPBTQuantityCanonicalizeRoundTrip: CanonicalizeBytes must emit a
// form that re-parses to the same value, and canonicalising the re-parse
// must produce byte-identical output (canonical form is a fixed point).
func TestPBTQuantityCanonicalizeRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		q := drawQuantity(t)
		res, suffix := q.CanonicalizeBytes(nil)
		s := string(res) + string(suffix)
		q2, err := ParseQuantity(s)
		if err != nil {
			t.Fatalf("CanonicalizeBytes() = %q does not re-parse: %v", s, err)
		}
		if c := q2.Cmp(q); c != 0 {
			t.Fatalf("canonical round trip changed value via %q: cmp %d", s, c)
		}
		res2, suffix2 := q2.CanonicalizeBytes(nil)
		if s2 := string(res2) + string(suffix2); s2 != s {
			t.Fatalf("canonical form not a fixed point: %q -> %q", s, s2)
		}
	})
}

// TestPBTQuantityCmpAntisymmetricAndSign: Cmp must be antisymmetric,
// reflexive, and agree with Sign() against zero.
func TestPBTQuantityCmpAntisymmetricAndSign(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := drawQuantity(t)
		b := drawQuantity(t)
		if got, want := a.Cmp(b), -b.Cmp(a); got != want {
			t.Fatalf("Cmp not antisymmetric: Cmp(a,b)=%d but Cmp(b,a) implies %d", got, want)
		}
		if a.Cmp(a) != 0 {
			t.Fatalf("Cmp(a,a) != 0 for %q", a.String())
		}
		zero := Quantity{}
		if got, want := a.Sign(), sign(a.Cmp(zero)); got != want {
			t.Fatalf("Sign()=%d but Cmp(a,0) has sign %d for %q", got, want, a.String())
		}
		if a.IsZero() != (a.Sign() == 0) {
			t.Fatalf("IsZero() disagrees with Sign() for %q", a.String())
		}
	})
}

// TestPBTQuantityCmpTransitive: Cmp must induce a total order.
func TestPBTQuantityCmpTransitive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := drawQuantity(t)
		b := drawQuantity(t)
		c := drawQuantity(t)
		if a.Cmp(b) <= 0 && b.Cmp(c) <= 0 && a.Cmp(c) > 0 {
			t.Fatalf("Cmp not transitive: %q <= %q <= %q but first > last",
				a.String(), b.String(), c.String())
		}
	})
}

// TestPBTQuantityAddCommutative: Add is documented as plain addition of
// values; operand order must not change the result.
func TestPBTQuantityAddCommutative(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := drawQuantity(t)
		b := drawQuantity(t)
		ab := a
		ab.Add(b)
		ba := b
		ba.Add(a)
		if ab.Cmp(ba) != 0 {
			t.Fatalf("Add not commutative: %q + %q = %q but reversed = %q",
				a.String(), b.String(), ab.String(), ba.String())
		}
	})
}

// TestPBTQuantityAddSubInverse: (a + b) - b must equal a, and (a - b) + b
// must equal a — whenever Add/Sub's documented bounded-scale rounding
// does not engage. That rounding (amount.go, maxScaleGap) rounds the
// finer-scale operand away from zero when the operands' Dec scales are
// more than 29 apart, which deliberately trades exactness for bounded
// cost; the unconditioned property fails there by design
// (see findings/003). The precondition below mirrors the documented
// rule exactly: |scale(a) - scale(b)| <= 29 at the Dec level.
func TestPBTQuantityAddSubInverse(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := drawQuantity(t)
		b := drawQuantity(t)
		gap := int64(a.AsDec().Scale()) - int64(b.AsDec().Scale())
		if gap < 0 {
			gap = -gap
		}
		if gap > 29 {
			t.Skipf("documented bounded-scale rounding engages: Dec scale gap %d > 29", gap)
		}
		sum := a
		sum.Add(b)
		back := sum
		back.Sub(b)
		if back.Cmp(a) != 0 {
			t.Fatalf("(a+b)-b != a: a=%q b=%q got %q", a.String(), b.String(), back.String())
		}
		diff := a
		diff.Sub(b)
		back2 := diff
		back2.Add(b)
		if back2.Cmp(a) != 0 {
			t.Fatalf("(a-b)+b != a: a=%q b=%q got %q", a.String(), b.String(), back2.String())
		}
	})
}

// TestPBTQuantityAddZeroSubSelf: adding zero is the identity, and
// subtracting a value from itself yields zero.
func TestPBTQuantityAddZeroSubSelf(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := drawQuantity(t)
		zero := Quantity{}
		plus := a
		plus.Add(zero)
		if plus.Cmp(a) != 0 {
			t.Fatalf("a + 0 != a for %q (got %q)", a.String(), plus.String())
		}
		self := a
		self.Sub(a)
		if !self.IsZero() {
			t.Fatalf("a - a != 0 for %q (got %q)", a.String(), self.String())
		}
	})
}

// TestPBTQuantityCmpVsSubSign: Cmp's sign must agree with the sign of
// the difference a - b.
func TestPBTQuantityCmpVsSubSign(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := drawQuantity(t)
		b := drawQuantity(t)
		diff := a
		diff.Sub(b)
		if got, want := diff.Sign(), sign(a.Cmp(b)); got != want {
			t.Fatalf("sign(a-b)=%d but sign(Cmp(a,b))=%d for a=%q b=%q (diff %q)",
				got, want, a.String(), b.String(), diff.String())
		}
	})
}

// drawScaled builds exact int64-backed quantities: value/10^-scale.
func drawScaled(t *rapid.T) (Quantity, int64, Scale) {
	v := rapid.Int64().Draw(t, "v")
	scale := Scale(rapid.IntRange(int(Nano), int(Kilo)).Draw(t, "scale"))
	return *NewScaledQuantity(v, scale), v, scale
}

// TestPBTQuantityScaledValueExact: ScaledValue at the quantity's own
// scale must return the exact constructor value ("ScaledValue returns
// the value of q/10^scale"); Value()/MilliValue() are its scale 0/-3
// cases for quantities built at those scales.
func TestPBTQuantityScaledValueExact(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		q, v, scale := drawScaled(t)
		if got := q.ScaledValue(scale); got != v {
			t.Fatalf("ScaledValue(%d) = %d, want constructor value %d", scale, got, v)
		}
		if got, ok := q.AsScaledInt64(scale); !ok || got != v {
			t.Fatalf("AsScaledInt64(%d) = (%d, %v), want (%d, true)", scale, got, ok, v)
		}
	})
}

// TestPBTQuantityScaledValueRounding: ScaledValue is documented to
// round away from zero. A milli quantity v/1000 read at scale 0 must be
// ceil(|v|/1000) with v's sign.
func TestPBTQuantityScaledValueRounding(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rapid.Int64Range(-1_000_000_000, 1_000_000_000).Draw(t, "v")
		q := *NewScaledQuantity(v, Milli)
		want := v / 1000
		if v%1000 != 0 {
			if v > 0 {
				want++
			} else {
				want--
			}
		}
		if got := q.ScaledValue(0); got != want {
			t.Fatalf("ScaledValue(0) of %dm = %d, want %d (away from zero)", v, got, want)
		}
		if got := q.MilliValue(); got != v {
			t.Fatalf("MilliValue() of %dm = %d, want %d", v, got, v)
		}
	})
}

// TestPBTQuantityScaledValueSaturates: the docs promise ScaledValue
// saturates to MinInt64/MaxInt64 when the value does not fit, while
// AsScaledInt64 reports ok=false.
func TestPBTQuantityScaledValueSaturates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		big := rapid.SampledFrom([]string{"1e30", "1e100", "123456789012345678901234567890"}).Draw(t, "big")
		if rapid.Bool().Draw(t, "negative") {
			big = "-" + big
		}
		q, err := ParseQuantity(big)
		if err != nil {
			t.Fatalf("ParseQuantity(%q): %v", big, err)
		}
		want := int64(math.MaxInt64)
		if strings.HasPrefix(big, "-") {
			want = math.MinInt64
		}
		if got := q.ScaledValue(0); got != want {
			t.Fatalf("ScaledValue(0) of %q = %d, want saturation to %d", big, got, want)
		}
		if _, ok := q.AsScaledInt64(0); ok {
			t.Fatalf("AsScaledInt64(0) of %q reported ok=true, want overflow", big)
		}
	})
}

// TestPBTQuantityScaledValueMonotonic: ScaledValue at a fixed scale
// must be monotone in the quantity's order (rounding away from zero and
// saturation both preserve order).
func TestPBTQuantityScaledValueMonotonic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := drawQuantity(t)
		b := drawQuantity(t)
		scale := Scale(rapid.IntRange(int(Nano), int(Kilo)).Draw(t, "scale"))
		if a.Cmp(b) <= 0 && a.ScaledValue(scale) > b.ScaledValue(scale) {
			t.Fatalf("ScaledValue(%d) not monotone: %q <= %q but %d > %d",
				scale, a.String(), b.String(), a.ScaledValue(scale), b.ScaledValue(scale))
		}
	})
}
