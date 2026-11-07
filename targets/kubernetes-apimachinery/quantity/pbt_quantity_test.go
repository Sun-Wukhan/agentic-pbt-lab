// Property-based tests for resource.Quantity serialisation formats,
// constructors, backend equivalence and the Mul/Neg/RoundUp algebra.
//
// Lab: Agentic PBT Lab (targets/kubernetes-apimachinery, wave 2 — see
// PROPERTIES.md in this directory). This module replaces
// k8s.io/apimachinery with the lab's local clone; the clone is never
// modified.
package quantity_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strings"
	"testing"

	inf "gopkg.in/inf.v0"
	"k8s.io/apimachinery/pkg/api/resource"

	"pgregory.net/rapid"
)

// ---------- generators ----------

var adversarialStrings = []string{
	"8Ei", "9Ei", "0.9n", "0.1n", "1.5n", "0.000000001",
	"18446744073709551616", "9223372036854775807", "-9223372036854775808",
	"1.2345", "1e30", "-1e30", "1E-9", "1e-10", "1e100", "1e-100",
	"1.1Ki", "0.5Mi", "1.5Gi", "1024", "1000", "0", "-0", "0.000",
	"3.141592653589793238462643383279502884197",
	"250m", "2", "0.5", "129e6", "123456789n", "7.5Ki",
}

func drawParsedString(t *rapid.T) string {
	return rapid.OneOf(
		rapid.SampledFrom(adversarialStrings),
		rapid.Custom(func(t *rapid.T) string {
			var b strings.Builder
			if rapid.Bool().Draw(t, "negative") {
				b.WriteByte('-')
			}
			b.WriteString(rapid.StringOfN(rapid.RuneFrom([]rune("0123456789")), 1, 19, -1).Draw(t, "intDigits"))
			if rapid.Bool().Draw(t, "hasFraction") {
				b.WriteByte('.')
				b.WriteString(rapid.StringOfN(rapid.RuneFrom([]rune("0123456789")), 0, 12, -1).Draw(t, "fracDigits"))
			}
			b.WriteString(rapid.SampledFrom([]string{
				"", "n", "u", "m", "k", "M", "G", "T", "P", "E",
				"Ki", "Mi", "Gi", "Ti", "Pi", "Ei",
				"e3", "e-3", "E6", "E-9", "e0", "e30", "e-30",
			}).Draw(t, "suffix"))
			return b.String()
		}),
	).Draw(t, "qs")
}

func drawFormat(t *rapid.T) resource.Format {
	return rapid.SampledFrom([]resource.Format{
		resource.DecimalSI, resource.BinarySI, resource.DecimalExponent,
	}).Draw(t, "format")
}

// drawQuantity builds quantities through every public construction path,
// so properties exercise both the int64 fast-path backend and the
// inf.Dec backend, with mixed formats.
func drawQuantity(t *rapid.T) resource.Quantity {
	return rapid.OneOf(
		rapid.Custom(func(t *rapid.T) resource.Quantity {
			s := drawParsedString(t)
			q, err := resource.ParseQuantity(s)
			if err != nil {
				t.Skipf("unparsable generated input %q: %v", s, err)
			}
			return q
		}),
		rapid.Custom(func(t *rapid.T) resource.Quantity {
			v := rapid.Int64().Draw(t, "v")
			scale := resource.Scale(rapid.IntRange(-12, 24).Draw(t, "scale"))
			return *resource.NewScaledQuantity(v, scale)
		}),
		rapid.Custom(func(t *rapid.T) resource.Quantity {
			return *resource.NewMilliQuantity(rapid.Int64().Draw(t, "v"), drawFormat(t))
		}),
		rapid.Custom(func(t *rapid.T) resource.Quantity {
			return *resource.NewQuantity(rapid.Int64().Draw(t, "v"), drawFormat(t))
		}),
		rapid.Custom(func(t *rapid.T) resource.Quantity {
			unscaled := rapid.Int64().Draw(t, "unscaled")
			scale := inf.Scale(rapid.IntRange(-20, 20).Draw(t, "decScale"))
			return *resource.NewDecimalQuantity(*inf.NewDec(unscaled, scale), drawFormat(t))
		}),
		rapid.Custom(func(t *rapid.T) resource.Quantity {
			q := resource.Quantity{}
			q.SetScaled(rapid.Int64().Draw(t, "v"), resource.Scale(rapid.IntRange(-12, 24).Draw(t, "scale")))
			q.Format = drawFormat(t)
			return q
		}),
	).Draw(t, "q")
}

// nanoExact reports whether q is an exact multiple of 1n, the documented
// precondition for round-trips through the parser.
func nanoExact(q resource.Quantity) bool {
	_, exact := q.AsScale(resource.Nano)
	return exact
}

var (
	maxInt64Quantity = resource.NewQuantity(math.MaxInt64, resource.DecimalSI)
	minInt64Quantity = resource.NewQuantity(math.MinInt64, resource.DecimalSI)
)

// exceedsBinaryCap reports whether q would be emitted with a binary SI
// suffix while exceeding the documented BinarySI parse cap of 2^63-1 in
// magnitude, in which case re-parsing its string form caps it by design
// (quantity.go type doc; finding 003).
func exceedsBinaryCap(q resource.Quantity) bool {
	return q.Format == resource.BinarySI &&
		(q.Cmp(*maxInt64Quantity) > 0 || q.Cmp(*minInt64Quantity) < 0)
}

// absExceedsInt64 reports whether |q| > 2^63-1, for cases where a test
// forces BinarySI rendering of a quantity of any original format.
func absExceedsInt64(q resource.Quantity) bool {
	return q.Cmp(*maxInt64Quantity) > 0 || q.Cmp(*minInt64Quantity) < 0
}

// ratFromDec converts an inf.Dec to the exact rational it denotes.
func ratFromDec(d *inf.Dec) *big.Rat {
	r := new(big.Rat).SetInt(d.UnscaledBig())
	scale := int64(d.Scale())
	if scale == 0 {
		return r
	}
	pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(abs64(scale)), nil)
	if scale > 0 {
		return r.Quo(r, new(big.Rat).SetInt(pow))
	}
	return r.Mul(r, new(big.Rat).SetInt(pow))
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func ratOf(q resource.Quantity) *big.Rat { return ratFromDec(q.AsDec()) }

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

// ---------- serialisation formats ----------

// R1: JSON round-trip preserves value; printed form is a fixed point.
func TestPBTQuantityJSONRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		q := drawQuantity(t)
		data, err := q.MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON(%q): %v", q.String(), err)
		}
		var back resource.Quantity
		if err := back.UnmarshalJSON(data); err != nil {
			t.Fatalf("UnmarshalJSON(%s) from %q: %v", data, q.String(), err)
		}
		if exceedsBinaryCap(q) {
			return // documented BinarySI parse cap; finding 003
		}
		if !nanoExact(q) {
			if back.Cmp(q) == 0 {
				t.Fatalf("sub-nano %q unexpectedly survived JSON round trip", q.String())
			}
			return // documented parse nano floor; see PROPERTIES.md notes
		}
		if back.Cmp(q) != 0 {
			t.Fatalf("JSON round trip changed value: %q -> %s -> %q", q.String(), data, back.String())
		}
		// via encoding/json as well
		data2, err := json.Marshal(q)
		if err != nil {
			t.Fatalf("json.Marshal(%q): %v", q.String(), err)
		}
		var back2 resource.Quantity
		if err := json.Unmarshal(data2, &back2); err != nil {
			t.Fatalf("json.Unmarshal(%s): %v", data2, err)
		}
		if back2.Cmp(q) != 0 {
			t.Fatalf("encoding/json round trip changed value: %q -> %s -> %q", q.String(), data2, back2.String())
		}
	})
}

// R2: bare JSON numbers and null.
func TestPBTQuantityJSONBareNumbersAndNull(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var b strings.Builder
		if rapid.Bool().Draw(t, "negative") {
			b.WriteByte('-')
		}
		b.WriteString(rapid.StringOfN(rapid.RuneFrom([]rune("123456789")), 1, 1, -1).Draw(t, "lead"))
		b.WriteString(rapid.StringOfN(rapid.RuneFrom([]rune("0123456789")), 0, 15, -1).Draw(t, "digits"))
		if rapid.Bool().Draw(t, "hasFraction") {
			b.WriteByte('.')
			b.WriteString(rapid.StringOfN(rapid.RuneFrom([]rune("0123456789")), 1, 9, -1).Draw(t, "frac"))
		}
		if rapid.Bool().Draw(t, "hasExponent") {
			b.WriteByte('e')
			if rapid.Bool().Draw(t, "expNegative") {
				b.WriteByte('-')
			}
			b.WriteString(rapid.StringOfN(rapid.RuneFrom([]rune("0123456789")), 1, 2, -1).Draw(t, "exp"))
		}
		lit := b.String()
		want, err := resource.ParseQuantity(lit)
		if err != nil {
			t.Skipf("generator produced unparsable literal %q: %v", lit, err)
		}
		var got resource.Quantity
		if err := got.UnmarshalJSON([]byte(lit)); err != nil {
			t.Fatalf("UnmarshalJSON(bare %s): %v", lit, err)
		}
		if got.Cmp(want) != 0 {
			t.Fatalf("bare JSON number %s decoded as %q, want %q", lit, got.String(), want.String())
		}
		// null resets to zero
		nonzero := resource.MustParse("5")
		if err := nonzero.UnmarshalJSON([]byte("null")); err != nil {
			t.Fatalf("UnmarshalJSON(null): %v", err)
		}
		if !nonzero.IsZero() {
			t.Fatalf("UnmarshalJSON(null) left %q, want 0", nonzero.String())
		}
	})
}

// R2b: the unmarshalers never panic on arbitrary bytes.
func TestPBTQuantityUnmarshalNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		u8 := rapid.SliceOf(rapid.Uint8()).Draw(t, "data")
		data := make([]byte, len(u8))
		for i, v := range u8 {
			data[i] = v
		}
		var q resource.Quantity
		_ = q.UnmarshalJSON(data)
		_ = q.Unmarshal(data)
		_ = q.UnmarshalCBOR(data)
	})
}

// R3: protobuf round-trip, Size() accuracy, determinism.
func TestPBTQuantityProtoRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		q := drawQuantity(t)
		data, err := q.Marshal()
		if err != nil {
			t.Fatalf("proto Marshal(%q): %v", q.String(), err)
		}
		if got := q.Size(); got != len(data) {
			t.Fatalf("Size() = %d but Marshal produced %d bytes for %q", got, len(data), q.String())
		}
		data2, err := q.Marshal()
		if err != nil || !bytes.Equal(data, data2) {
			t.Fatalf("proto Marshal not deterministic for %q", q.String())
		}
		var back resource.Quantity
		if err := back.Unmarshal(data); err != nil {
			t.Fatalf("proto Unmarshal of %q encoding: %v", q.String(), err)
		}
		if !nanoExact(q) || exceedsBinaryCap(q) {
			return // documented parse nano floor / BinarySI cap on decode
		}
		if back.Cmp(q) != 0 {
			t.Fatalf("proto round trip changed value: %q -> %q", q.String(), back.String())
		}
	})
}

// R4: CBOR round-trip.
func TestPBTQuantityCBORRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		q := drawQuantity(t)
		data, err := q.MarshalCBOR()
		if err != nil {
			t.Fatalf("MarshalCBOR(%q): %v", q.String(), err)
		}
		var back resource.Quantity
		if err := back.UnmarshalCBOR(data); err != nil {
			t.Fatalf("UnmarshalCBOR of %q encoding: %v", q.String(), err)
		}
		if !nanoExact(q) || exceedsBinaryCap(q) {
			return
		}
		if back.Cmp(q) != 0 {
			t.Fatalf("CBOR round trip changed value: %q -> %q", q.String(), back.String())
		}
		// CBOR null decodes to zero
		nonzero := resource.MustParse("5")
		if err := nonzero.UnmarshalCBOR([]byte{0xf6}); err != nil {
			t.Fatalf("UnmarshalCBOR(null): %v", err)
		}
		if !nonzero.IsZero() {
			t.Fatalf("UnmarshalCBOR(null) left %q, want 0", nonzero.String())
		}
	})
}

// ---------- backend model equivalence ----------

// R5: Cmp agrees with the inf.Dec model; AsDec is a defensive copy.
func TestPBTQuantityDecModelEquivalence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := drawQuantity(t)
		b := drawQuantity(t)
		if got, want := sign(a.Cmp(b)), sign(a.AsDec().Cmp(b.AsDec())); got != want {
			t.Fatalf("Cmp sign %d but Dec model sign %d for %q vs %q", got, want, a.String(), b.String())
		}
		d := a.AsDec()
		before := a.String()
		d.SetUnscaled(123456789)
		d.SetScale(3)
		if a.String() != before {
			t.Fatalf("mutating AsDec() result changed quantity: %q -> %q", before, a.String())
		}
		rebuilt := resource.NewDecimalQuantity(*a.AsDec(), a.Format)
		if rebuilt.Cmp(a) != 0 {
			t.Fatalf("NewDecimalQuantity(AsDec) != original: %q vs %q", rebuilt.String(), a.String())
		}
	})
}

// R13: CmpInt64 agrees with Cmp against a constructed quantity.
func TestPBTQuantityCmpInt64Consistent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		q := drawQuantity(t)
		v := rapid.Int64().Draw(t, "v")
		if got, want := q.CmpInt64(v), q.Cmp(*resource.NewQuantity(v, resource.DecimalSI)); got != want {
			t.Fatalf("CmpInt64(%d) = %d but Cmp(NewQuantity) = %d for %q", v, got, want, q.String())
		}
	})
}

// ---------- arithmetic algebra beyond wave 1 ----------

// R6: Mul is always exact (big.Rat model), whatever the bool reports.
func TestPBTQuantityMulExact(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		q := drawQuantity(t)
		y := rapid.Int64Range(-1_000_000, 1_000_000).Draw(t, "y")
		got := q
		got.Mul(y)
		want := new(big.Rat).Mul(ratOf(q), big.NewRat(y, 1))
		if ratOf(got).Cmp(want) != 0 {
			t.Fatalf("%q * %d = %q, exact model says %s", q.String(), y, got.String(), want.RatString())
		}
		one := q
		one.Mul(1)
		if one.Cmp(q) != 0 {
			t.Fatalf("Mul(1) changed %q to %q", q.String(), one.String())
		}
		zero := q
		zero.Mul(0)
		if !zero.IsZero() {
			t.Fatalf("Mul(0) of %q gave %q", q.String(), zero.String())
		}
		neg := q
		neg.Mul(-1)
		negated := q
		negated.Neg()
		if neg.Cmp(negated) != 0 {
			t.Fatalf("Mul(-1) of %q gave %q, Neg gives %q", q.String(), neg.String(), negated.String())
		}
	})
}

// R7: Neg algebra.
func TestPBTQuantityNegAlgebra(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := drawQuantity(t)
		b := drawQuantity(t)
		nn := a
		nn.Neg()
		nn.Neg()
		if nn.Cmp(a) != 0 {
			t.Fatalf("Neg(Neg(%q)) = %q", a.String(), nn.String())
		}
		na, nb := a, b
		na.Neg()
		nb.Neg()
		if got, want := na.Cmp(nb), -a.Cmp(b); got != want {
			t.Fatalf("Neg order: Cmp(-a,-b)=%d, want %d for %q, %q", got, want, a.String(), b.String())
		}
		sum := a
		negA := a
		negA.Neg()
		sum.Add(negA)
		if !sum.IsZero() {
			t.Fatalf("a + (-a) = %q for %q", sum.String(), a.String())
		}
		sub := a
		sub.Sub(b)
		negB := b
		negB.Neg()
		add := a
		add.Add(negB)
		if sub.Cmp(add) != 0 {
			t.Fatalf("Sub != Add(Neg): %q - %q = %q but Add(Neg) = %q", a.String(), b.String(), sub.String(), add.String())
		}
	})
}

// R8: RoundUp contract.
func TestPBTQuantityRoundUpContract(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		q := drawQuantity(t)
		s := resource.Scale(rapid.IntRange(-12, 24).Draw(t, "scale"))
		before := q
		r := q
		r.RoundUp(s)
		if _, exact := r.AsScale(s); !exact {
			t.Fatalf("after RoundUp(%d), AsScale not exact for %q (was %q)", s, r.String(), before.String())
		}
		r2 := r
		r2.RoundUp(s)
		if r2.Cmp(r) != 0 {
			t.Fatalf("RoundUp not idempotent: %q -> %q at scale %d", r.String(), r2.String(), s)
		}
		if r.Sign() != before.Sign() {
			t.Fatalf("RoundUp changed sign: %q -> %q", before.String(), r.String())
		}
		switch {
		case before.Sign() > 0 && r.Cmp(before) < 0:
			t.Fatalf("RoundUp moved positive value toward zero: %q -> %q", before.String(), r.String())
		case before.Sign() < 0 && r.Cmp(before) > 0:
			t.Fatalf("RoundUp moved negative value toward zero: %q -> %q", before.String(), r.String())
		}
	})
}

// ---------- rendering / canonical form ----------

// R9 + R16: re-render under any format / String() re-parses to the same
// value (nano-exact precondition).
func TestPBTQuantityFormatRerender(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		q := drawQuantity(t)
		if !nanoExact(q) {
			t.Skip("sub-nano value: documented parse nano floor")
		}
		for _, f := range []resource.Format{resource.DecimalSI, resource.BinarySI, resource.DecimalExponent} {
			if f == resource.BinarySI && absExceedsInt64(q) {
				continue // documented BinarySI parse cap; finding 003
			}
			r := q
			r.Format = f
			s := r.String()
			back, err := resource.ParseQuantity(s)
			if err != nil {
				t.Fatalf("String() under %s = %q does not re-parse: %v", f, s, err)
			}
			if back.Cmp(q) != 0 {
				t.Fatalf("format %s re-render changed value: %q -> %q -> %q", f, q.String(), s, back.String())
			}
		}
	})
}

// R10: CanonicalizeBytes fixed point + out-buffer prefix contract.
func TestPBTQuantityCanonicalizeConstructed(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		q := drawQuantity(t)
		if !nanoExact(q) {
			t.Skip("sub-nano value: documented parse nano floor")
		}
		if exceedsBinaryCap(q) {
			t.Skip("documented BinarySI parse cap; finding 003")
		}
		num, suf := q.CanonicalizeBytes(nil)
		s := string(num) + string(suf)
		back, err := resource.ParseQuantity(s)
		if err != nil {
			t.Fatalf("CanonicalizeBytes %q does not re-parse: %v", s, err)
		}
		if back.Cmp(q) != 0 {
			t.Fatalf("canonical round trip changed value: %q -> %q -> %q", q.String(), s, back.String())
		}
		// Byte fixed point holds within a format. A BinarySI quantity
		// whose canonical rendering carries no binary suffix (base-1024
		// exponent 0, e.g. 10000) re-parses as DecimalSI and re-renders
		// differently ("10k") — the value survives, the format does
		// not, by design (see PROPERTIES.md notes).
		if back.Format == q.Format {
			num2, suf2 := back.CanonicalizeBytes(nil)
			if s2 := string(num2) + string(suf2); s2 != s {
				t.Fatalf("canonical form not a fixed point: %q -> %q", s, s2)
			}
		}
		if !q.IsZero() {
			pnum, _ := q.CanonicalizeBytes([]byte("PRE"))
			if !strings.HasPrefix(string(pnum), "PRE") {
				t.Fatalf("CanonicalizeBytes dropped the caller's out prefix: %q", pnum)
			}
		}
	})
}

// ---------- value semantics ----------

// R11: copy independence under mutation.
func TestPBTQuantityCopyIndependence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orig := drawQuantity(t)
		before := orig.String()
		mutate := func(c *resource.Quantity) {
			other := drawQuantity(t)
			c.Add(other)
			c.Sub(other)
			c.Neg()
			c.Mul(3)
			c.RoundUp(resource.Milli)
			c.SetScaled(42, resource.Milli)
		}
		byValue := orig
		mutate(&byValue)
		if orig.String() != before {
			t.Fatalf("mutating a value copy changed the original: %q -> %q", before, orig.String())
		}
		byDeepCopy := orig.DeepCopy()
		mutate(&byDeepCopy)
		if orig.String() != before {
			t.Fatalf("mutating a DeepCopy changed the original: %q -> %q", before, orig.String())
		}
		var byInto resource.Quantity
		orig.DeepCopyInto(&byInto)
		mutate(&byInto)
		if orig.String() != before {
			t.Fatalf("mutating a DeepCopyInto copy changed the original: %q -> %q", before, orig.String())
		}
		// and the other direction: mutating orig must not move a snapshot copy
		snap := orig
		mutate(&orig)
		if snap.String() != before {
			t.Fatalf("mutating the original changed an earlier copy: %q -> %q", before, snap.String())
		}
	})
}

// R17 + R18: zero value behaviour and format adoption.
func TestPBTQuantityZeroValueAndFormatAdoption(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		var z resource.Quantity
		if z.String() != "0" || !z.IsZero() || z.Sign() != 0 {
			t.Fatalf("zero value: String=%q IsZero=%v Sign=%d", z.String(), z.IsZero(), z.Sign())
		}
		if data, _ := z.MarshalJSON(); string(data) != `"0"` {
			t.Fatalf("zero MarshalJSON = %s, want \"0\"", data)
		}
		zeroParsed := resource.MustParse("0")
		if z.Cmp(zeroParsed) != 0 {
			t.Fatalf("zero value != ParseQuantity(0)")
		}
		y := drawQuantity(t)
		added := resource.Quantity{}
		added.Add(y)
		if added.Format != y.Format {
			t.Fatalf("0.Add(y) format = %q, want y's %q", added.Format, y.Format)
		}
		subbed := resource.Quantity{}
		subbed.Sub(y)
		if subbed.Format != y.Format {
			t.Fatalf("0.Sub(y) format = %q, want y's %q", subbed.Format, y.Format)
		}
	})
}

// ---------- float conversion ----------

// R12: AsFloat64Slow is the correctly rounded float64 of the exact value.
func TestPBTQuantityFloat64SlowExact(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		q := drawQuantity(t)
		d := q.AsDec()
		if d.Scale() > 300 || d.Scale() < -300 {
			t.Skip("outside the bounded model range")
		}
		want, _ := ratOf(q).Float64()
		if got := q.AsFloat64Slow(); got != want {
			t.Fatalf("AsFloat64Slow(%q) = %v, exact model %v", q.String(), got, want)
		}
		approx := q.AsApproximateFloat64()
		if (want == 0) != (approx == 0) && math.Abs(want) > 1e-300 && math.Abs(want) < 1e300 {
			t.Fatalf("AsApproximateFloat64(%q) = %v but exact is %v", q.String(), approx, want)
		}
		if want > 0 && approx < 0 || want < 0 && approx > 0 {
			t.Fatalf("AsApproximateFloat64(%q) sign disagrees: %v vs exact %v", q.String(), approx, want)
		}
	})
}

// ---------- extreme scales & exponent range ----------

// R14: extreme constructor scales.
func TestPBTQuantityExtremeScales(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := rapid.Int64().Draw(t, "v")
		s := rapid.SampledFrom([]resource.Scale{
			-12, -10, -9, -3, 0, 3, 18, 24, 100, -100, 1000, -1000,
			resource.Scale(math.MaxInt32 - 2), resource.Scale(math.MinInt32 + 2),
		}).Draw(t, "scale")
		q := resource.NewScaledQuantity(v, s)
		if got := q.ScaledValue(s); got != v {
			t.Fatalf("NewScaledQuantity(%d, %d).ScaledValue(%d) = %d", v, s, s, got)
		}
		if got, ok := q.AsScaledInt64(s); !ok || got != v {
			t.Fatalf("AsScaledInt64(%d) = (%d, %v), want (%d, true)", s, got, ok, v)
		}
		q2 := resource.Quantity{}
		q2.SetScaled(v, s)
		if q2.Cmp(*q) != 0 {
			t.Fatalf("SetScaled(%d, %d) != NewScaledQuantity", v, s)
		}
		if s >= resource.Nano {
			str := q.String()
			back, err := resource.ParseQuantity(str)
			if err != nil {
				t.Fatalf("String() of scale-%d quantity = %q does not re-parse: %v", s, str, err)
			}
			if back.Cmp(*q) != 0 {
				t.Fatalf("String round trip at scale %d: %q -> %q", s, str, back.String())
			}
		}
	})
}

// R15 (directed): exponent suffixes outside the representable scale
// range must be rejected, never wrapped modulo 2^32.
// FAILS at the pinned commit — see findings/001.
func TestPBTQuantityExponentRange(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// k in [2^32, 2^40): int32(k) wraps, so a wrapping parser
		// silently accepts these with a wrong exponent.
		k := rapid.Int64Range(1<<32, 1<<40).Draw(t, "k")
		if rapid.Bool().Draw(t, "negative") {
			k = -k
		}
		s := fmt.Sprintf("1e%d", k)
		q, err := resource.ParseQuantity(s)
		if err == nil {
			t.Fatalf("ParseQuantity(%q) silently accepted with wrapped value %q; exponents outside int32 range must error", s, q.String())
		}
	})
}
