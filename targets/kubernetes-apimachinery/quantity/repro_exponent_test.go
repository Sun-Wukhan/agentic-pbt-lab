package quantity_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
)

// Minimal standalone reproducer for finding 001 (exponent suffix int32
// wraparound). This test encodes the CORRECT contract — an exponent
// outside the representable scale range must be rejected — and
// therefore FAILS at the pinned commit. Each case also pins the value
// the buggy parser actually produces, so the failure output shows the
// silent corruption directly.
func TestReproExponentWraparound(t *testing.T) {
	cases := []struct {
		input string
		buggy string // value the pinned parser silently produces instead of an error
	}{
		{"1e4294967296", "1"},     // 2^32 wraps to exponent 0
		{"1e4294967297", "10"},    // 2^32+1 wraps to exponent 1
		{"1e4294967295", "100m"},  // 2^32-1 wraps to exponent -1 (0.1)
		{"1e-4294967296", "1"},    // -(2^32) wraps to exponent 0
		{"2e8589934592", "2"},     // 2^33 wraps to exponent 0
		{"3e12884901890", "300"},  // 3*2^32+2 wraps to exponent 2
		{"1.5e4294967297", "15"},  // Dec path, not the int64 fast path
	}
	for _, c := range cases {
		q, err := resource.ParseQuantity(c.input)
		if err == nil {
			want, _ := resource.ParseQuantity(c.buggy)
			if q.Cmp(want) == 0 {
				t.Errorf("ParseQuantity(%q) = %q (== %q): silently wrapped exponent, want an error",
					c.input, q.String(), c.buggy)
			} else {
				t.Errorf("ParseQuantity(%q) = %q: silently accepted out-of-range exponent, want an error",
					c.input, q.String())
			}
		}
	}
	// Controls: in-range and boundary exponents behave sanely today and
	// must keep doing so after any fix.
	if _, err := resource.ParseQuantity("1e2147483647"); err != nil {
		t.Errorf("ParseQuantity(1e2147483647) (MaxInt32): unexpected error %v", err)
	}
	if _, err := resource.ParseQuantity("1e2147483648"); err == nil {
		t.Errorf("ParseQuantity(1e2147483648) (MinInt32 after narrowing): want an error, got a value")
	}
	if _, err := resource.ParseQuantity("1e-2147483648"); err == nil {
		t.Errorf("ParseQuantity(1e-2147483648): want an error, got a value")
	}
}
