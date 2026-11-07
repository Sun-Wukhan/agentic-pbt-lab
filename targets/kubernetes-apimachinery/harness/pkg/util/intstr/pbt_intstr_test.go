// Property-based tests for intstr.GetScaledValueFromIntOrPercent.
//
// Lab: Agentic PBT Lab (targets/kubernetes-apimachinery) — pristine copy.
// run.sh copies this file into the apimachinery checkout at
// pkg/util/intstr/pbt_intstr_test.go and runs it in-package
// (package intstr).
package intstr

import (
	"testing"

	"pgregory.net/rapid"
)

// floorDiv is exact integer floor division for a possibly negative
// numerator and a positive denominator.
func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && a < 0 {
		q--
	}
	return q
}

func ceilDiv(a, b int64) int64 {
	return -floorDiv(-a, b)
}

// TestPBTIntstrScaledNeverPanics: GetScaledValueFromIntOrPercent and
// FromString consume values from API objects (rolling-update params,
// probes); they must error or return, never panic.
func TestPBTIntstrScaledNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.String().Draw(t, "s")
		v := FromString(s)
		total := rapid.Int().Draw(t, "total")
		_, _ = GetScaledValueFromIntOrPercent(&v, total, rapid.Bool().Draw(t, "roundUp"))
		iv := FromInt32(rapid.Int32().Draw(t, "i32"))
		_, _ = GetScaledValueFromIntOrPercent(&iv, total, false)
	})
}

// TestPBTIntstrIntPassthrough: for a non-percent IntOrString the
// function returns the integer unchanged, whatever the total (its
// documented contract: the value is used as-is). The type stores an
// int32 (FromInt: "It is your responsibility not to call this method
// with a value greater than int32"), so the property ranges over the
// int32 domain; see findings/001.
func TestPBTIntstrIntPassthrough(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := int(rapid.Int32().Draw(t, "n"))
		total := rapid.Int().Draw(t, "total")
		v := FromInt(n)
		for _, roundUp := range []bool{false, true} {
			got, err := GetScaledValueFromIntOrPercent(&v, total, roundUp)
			if err != nil {
				t.Fatalf("int value %d: unexpected error %v", n, err)
			}
			if got != n {
				t.Fatalf("int value %d with total %d roundUp=%v: got %d", n, total, roundUp, got)
			}
		}
	})
}

// TestPBTIntstrPercentExact: for percent p and total t, the result must
// be the exact mathematical floor (roundUp=false) or ceil (roundUp=true)
// of p*t/100 — the doc comment promises percentage of the total, rounded
// as selected. Bounds are kept where float64 arithmetic in the
// implementation is exact for the product, so any mismatch is a real
// rounding defect, not float noise: p in [-1000, 1000], t in [0, 10^7].
func TestPBTIntstrPercentExact(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		p := rapid.IntRange(-1000, 1000).Draw(t, "p")
		total := rapid.IntRange(0, 10_000_000).Draw(t, "total")
		v := FromString(intToPercent(p))
		gotFloor, err := GetScaledValueFromIntOrPercent(&v, total, false)
		if err != nil {
			t.Fatalf("percent %d total %d: unexpected error %v", p, total, err)
		}
		if want := int(floorDiv(int64(p)*int64(total), 100)); gotFloor != want {
			t.Fatalf("floor: %d%% of %d = %d, want exact %d", p, total, gotFloor, want)
		}
		gotCeil, err := GetScaledValueFromIntOrPercent(&v, total, true)
		if err != nil {
			t.Fatalf("percent %d total %d: unexpected error %v", p, total, err)
		}
		if want := int(ceilDiv(int64(p)*int64(total), 100)); gotCeil != want {
			t.Fatalf("ceil: %d%% of %d = %d, want exact %d", p, total, gotCeil, want)
		}
	})
}

// TestPBTIntstrPercentBoundsAndMonotonic: for a percentage in [0,100],
// the result must lie in [0, total] and be non-decreasing in total.
func TestPBTIntstrPercentBoundsAndMonotonic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		p := rapid.IntRange(0, 100).Draw(t, "p")
		t1 := rapid.IntRange(0, 1_000_000).Draw(t, "t1")
		t2 := t1 + rapid.IntRange(0, 1_000_000).Draw(t, "t2delta")
		v := FromString(intToPercent(p))
		for _, roundUp := range []bool{false, true} {
			r1, err := GetScaledValueFromIntOrPercent(&v, t1, roundUp)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			r2, err := GetScaledValueFromIntOrPercent(&v, t2, roundUp)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r1 < 0 || r1 > t1 {
				t.Fatalf("%d%% of %d (roundUp=%v) = %d, outside [0, %d]", p, t1, roundUp, r1, t1)
			}
			if r1 > r2 {
				t.Fatalf("not monotone: %d%% of %d = %d but of larger %d = %d (roundUp=%v)",
					p, t1, r1, t2, r2, roundUp)
			}
		}
	})
}

func intToPercent(p int) string {
	digits := "0123456789"
	if p == 0 {
		return "0%"
	}
	neg := p < 0
	if neg {
		p = -p
	}
	s := "%"
	for p > 0 {
		s = string(digits[p%10]) + s
		p /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
