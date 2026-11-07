// Property-based tests for Masterminds/semver Constraints. See
// ../PROPERTIES.md (C1–C8) for the invariant list and sources.
package pbt

import (
	"fmt"
	"strings"
	"testing"

	semver "github.com/Masterminds/semver/v3"
	"pgregory.net/rapid"
)

var basicOps = []string{"=", "!=", ">", "<", ">=", "<="}

// drawVersionish draws a constraint-style version: full, partial, or
// wildcard, with an optional prerelease.
func drawVersionish(t *rapid.T, label string) string {
	s := drawVSpecNoMeta(t, label)
	switch rapid.IntRange(0, 5).Draw(t, label+"Form") {
	case 0:
		return fmt.Sprintf("%d", s.major)
	case 1:
		return fmt.Sprintf("%d.%d", s.major, s.minor)
	case 2:
		return fmt.Sprintf("%d.%d.x", s.major, s.minor)
	case 3:
		return fmt.Sprintf("%d.x", s.major)
	case 4:
		return "*"
	default:
		return s.String()
	}
}

func drawSingleConstraint(t *rapid.T, label string) string {
	op := rapid.SampledFrom([]string{"", "=", "!=", ">", "<", ">=", "<=", "~", "^"}).Draw(t, label+"Op")
	v := drawVersionish(t, label+"V")
	if op == "" {
		return v
	}
	return op + " " + v
}

func drawConstraintString(t *rapid.T, label string) string {
	n := rapid.IntRange(1, 2).Draw(t, label+"N")
	parts := make([]string, n)
	for i := range parts {
		parts[i] = drawSingleConstraint(t, fmt.Sprintf("%s%d", label, i))
	}
	sep := rapid.SampledFrom([]string{", ", " "}).Draw(t, label+"Sep")
	c := strings.Join(parts, sep)
	if rapid.Bool().Draw(t, label+"Or") {
		c += " || " + drawSingleConstraint(t, label+"OrPart")
	}
	return c
}

func mustConstraint(t *rapid.T, s string) *semver.Constraints {
	c, err := semver.NewConstraint(s)
	if err != nil {
		t.Fatalf("NewConstraint(%q) errored: %v", s, err)
	}
	return c
}

// C1: constraint parsing/checking never panics.
func TestPBTConstraintsNeverPanic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.StringMatching(`[0-9a-zA-Z.\-+vV*~,^<>=!| ]{0,60}`).Draw(t, "s")
		c, err := semver.NewConstraint(s)
		v := mustStrict(t, drawVSpec(t, "v").String())
		if err == nil {
			_ = c.Check(v)
			_, _ = c.Validate(v)
			_ = c.String()
		}
		plain := rapid.String().Draw(t, "plain")
		_, _ = semver.NewConstraint(plain)
	})
}

// C2: Check and Validate agree; message-shape invariants.
func TestPBTCheckValidateAgree(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cs := drawConstraintString(t, "c")
		c, err := semver.NewConstraint(cs)
		if err != nil {
			return // generator may produce unparsable combos; C1 covers the no-panic side
		}
		if rapid.Bool().Draw(t, "incPre") {
			c.IncludePrerelease = true
		}
		v := mustStrict(t, drawVSpec(t, "v").String())
		check1 := c.Check(v)
		valid, msgs := c.Validate(v)
		check2 := c.Check(v)
		if check1 != check2 {
			t.Fatalf("Check not deterministic/stable around Validate for %q vs %q", cs, v)
		}
		if valid != check1 {
			t.Fatalf("Check(%q, %q) = %v but Validate = %v (msgs %v)", cs, v, check1, valid, msgs)
		}
		if valid && len(msgs) != 0 {
			t.Fatalf("Validate(%q, %q) true but messages %v", cs, v, msgs)
		}
		if !valid && len(msgs) == 0 {
			t.Fatalf("Validate(%q, %q) false but no reasons given", cs, v)
		}
	})
}

// opModel is the Compare-sign relation a basic operator denotes.
func opModel(op string, cmp int) bool {
	switch op {
	case "=", "":
		return cmp == 0
	case "!=":
		return cmp != 0
	case ">":
		return cmp > 0
	case "<":
		return cmp < 0
	case ">=":
		return cmp >= 0
	case "<=":
		return cmp <= 0
	}
	return false
}

// C3: basic operators follow Compare (with the documented prerelease gate).
func TestPBTBasicOpsFollowCompare(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		op := rapid.SampledFrom(basicOps).Draw(t, "op")
		xs := drawVSpecNoMeta(t, "x")
		vs := drawVSpecNoMeta(t, "v")
		v := mustStrict(t, vs.String())
		if preHasOverflowNumericID(vs.pre) || preHasOverflowNumericID(xs.pre) {
			t.Skip("findings/001 pattern: overflow numeric prerelease identifier")
		}

		c := mustConstraint(t, op+" "+xs.String())
		want := opModel(op, sign(modelCompare(vs, xs)))
		// Documented gate: a prerelease version is excluded when the
		// constraint group carries no prerelease.
		if vs.pre != "" && xs.pre == "" {
			want = false
		}
		if got := c.Check(v); got != want {
			t.Fatalf("Check(%q, %q) = %v, model = %v (Compare sign %d)", op+" "+xs.String(), vs.String(), got, want, sign(modelCompare(vs, xs)))
		}
	})
}

// C4: documented range equivalences.
func TestPBTRangeEquivalences(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.Uint64Range(0, 20).Draw(t, "a")
		b := rapid.Uint64Range(0, 20).Draw(t, "b")
		cc := rapid.Uint64Range(0, 20).Draw(t, "c")
		var left, right string
		switch rapid.IntRange(0, 6).Draw(t, "family") {
		case 0: // tilde, full (excluding the documented ~0.0.0 special case)
			if a == 0 && b == 0 && cc == 0 {
				cc = 1
			}
			left = fmt.Sprintf("~%d.%d.%d", a, b, cc)
			right = fmt.Sprintf(">= %d.%d.%d, < %d.%d.0", a, b, cc, a, b+1)
		case 1: // caret, major > 0
			if a == 0 {
				a = 1
			}
			left = fmt.Sprintf("^%d.%d.%d", a, b, cc)
			right = fmt.Sprintf(">= %d.%d.%d, < %d.0.0", a, b, cc, a+1)
		case 2: // caret, 0.minor
			if b == 0 {
				b = 1
			}
			left = fmt.Sprintf("^0.%d.%d", b, cc)
			right = fmt.Sprintf(">= 0.%d.%d, < 0.%d.0", b, cc, b+1)
		case 3: // caret, 0.0.patch
			left = fmt.Sprintf("^0.0.%d", cc)
			right = fmt.Sprintf(">= 0.0.%d, < 0.0.%d", cc, cc+1)
		case 4: // wildcard patch
			left = fmt.Sprintf("%d.%d.x", a, b)
			right = fmt.Sprintf(">= %d.%d.0, < %d.%d.0", a, b, a, b+1)
		case 5: // wildcard all
			left = "*"
			right = ">= 0.0.0"
		default: // hyphen range
			left = fmt.Sprintf("%d.%d - %d.%d.%d", a, b, a, b+1, cc)
			right = fmt.Sprintf(">= %d.%d, <= %d.%d.%d", a, b, a, b+1, cc)
		}
		cl := mustConstraint(t, left)
		cr := mustConstraint(t, right)
		vs := drawVSpecNoMeta(t, "v")
		vs.pre = "" // equivalences are stated for release versions
		v := mustStrict(t, vs.String())
		if cl.Check(v) != cr.Check(v) {
			t.Fatalf("equivalence broken: %q vs %q disagree on %q (%v vs %v)",
				left, right, vs.String(), cl.Check(v), cr.Check(v))
		}
	})
}

// C5: boundary self-consistency.
func TestPBTBoundarySelfConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		xs := drawVSpecNoMeta(t, "x")
		x := mustStrict(t, xs.String())
		expect := map[string]bool{
			">=": true, "<=": true, "=": true, "!=": false,
			">": false, "<": false, "~": true, "^": true,
		}
		for op, want := range expect {
			c := mustConstraint(t, op+" "+xs.String())
			if got := c.Check(x); got != want {
				t.Fatalf("boundary: Check(%q, %q) = %v, want %v", op+" "+xs.String(), xs.String(), got, want)
			}
		}
	})
}

// C6: Constraints.String round trip.
func TestPBTConstraintStringRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cs := drawConstraintString(t, "c")
		c1, err := semver.NewConstraint(cs)
		if err != nil {
			return
		}
		s := c1.String()
		c2, err := semver.NewConstraint(s)
		if err != nil {
			t.Fatalf("String() of %q produced unparsable %q: %v", cs, s, err)
		}
		if c2.String() != s {
			t.Fatalf("constraint String() not a fixed point: %q -> %q -> %q", cs, s, c2.String())
		}
		v := mustStrict(t, drawVSpec(t, "v").String())
		if c1.Check(v) != c2.Check(v) {
			t.Fatalf("round-tripped constraint %q (from %q) disagrees on %q", s, cs, v)
		}
	})
}

// C7: AND/OR composition (non-prerelease versions).
func TestPBTConstraintComposition(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s1 := drawSingleConstraint(t, "a")
		s2 := drawSingleConstraint(t, "b")
		c1, err1 := semver.NewConstraint(s1)
		c2, err2 := semver.NewConstraint(s2)
		if err1 != nil || err2 != nil {
			return
		}
		vs := drawVSpecNoMeta(t, "v")
		vs.pre = ""
		v := mustStrict(t, vs.String())

		and, err := semver.NewConstraint(s1 + ", " + s2)
		if err != nil {
			t.Fatalf("AND composition %q + %q unparsable: %v", s1, s2, err)
		}
		if got, want := and.Check(v), c1.Check(v) && c2.Check(v); got != want {
			t.Fatalf("AND: Check(%q && %q, %q) = %v, want %v", s1, s2, vs.String(), got, want)
		}
		or, err := semver.NewConstraint(s1 + " || " + s2)
		if err != nil {
			t.Fatalf("OR composition %q + %q unparsable: %v", s1, s2, err)
		}
		if got, want := or.Check(v), c1.Check(v) || c2.Check(v); got != want {
			t.Fatalf("OR: Check(%q || %q, %q) = %v, want %v", s1, s2, vs.String(), got, want)
		}
	})
}

// C8: IncludePrerelease.
func TestPBTIncludePrerelease(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		op := rapid.SampledFrom(basicOps).Draw(t, "op")
		xs := drawVSpecNoMeta(t, "x")
		xs.pre = "" // constraint without a prerelease of its own
		vs := drawVSpecNoMeta(t, "v")
		v := mustStrict(t, vs.String())
		if preHasOverflowNumericID(vs.pre) {
			t.Skip("findings/001 pattern: overflow numeric prerelease identifier")
		}

		cs := op + " " + xs.String()
		c := mustConstraint(t, cs)
		before := c.Check(v)
		c.IncludePrerelease = true
		after := c.Check(v)
		if vs.pre == "" && before != after {
			t.Fatalf("IncludePrerelease changed a release-version result: %q vs %q", cs, vs.String())
		}
		if vs.pre != "" {
			// With inclusion on, the raw Compare model applies.
			want := opModel(op, sign(modelCompare(vs, xs)))
			if after != want {
				t.Fatalf("IncludePrerelease: Check(%q, %q) = %v, model = %v", cs, vs.String(), after, want)
			}
		}
	})
}
