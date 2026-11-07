// Property-based tests for labels selector parsing, serialisation and
// matching.
//
// Lab: Agentic PBT Lab (targets/kubernetes-apimachinery) — pristine copy.
// run.sh copies this file into the apimachinery checkout at
// pkg/labels/pbt_labels_test.go and runs it in-package (package labels).
package labels

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/selection"

	"pgregory.net/rapid"
)

var labelKeys = []string{"app", "tier", "track", "example.com/name", "x", "a-b_c.d"}
var labelValues = []string{"web", "db", "stable", "canary", "v1", "3", "0", "10"}

// selectorCharset is dense in the selector grammar's special symbols.
const selectorCharset = "abx=!<>(),.inots-_/ "

// drawRequirementString builds one syntactically valid requirement in
// any of the grammar's forms.
func drawRequirementString(t *rapid.T) string {
	key := rapid.SampledFrom(labelKeys).Draw(t, "key")
	form := rapid.IntRange(0, 8).Draw(t, "form")
	val := func(name string) string { return rapid.SampledFrom(labelValues).Draw(t, name) }
	switch form {
	case 0:
		return key + "=" + val("v")
	case 1:
		return key + "==" + val("v")
	case 2:
		return key + "!=" + val("v")
	case 3:
		return fmt.Sprintf("%s in (%s, %s)", key, val("v1"), val("v2"))
	case 4:
		return fmt.Sprintf("%s notin (%s)", key, val("v1"))
	case 5:
		return key
	case 6:
		return "!" + key
	case 7:
		// Gt/Lt values pass through label-value validation, so only
		// non-negative integers are parseable (see findings/002).
		return fmt.Sprintf("%s > %d", key, rapid.IntRange(0, 10).Draw(t, "n"))
	default:
		return fmt.Sprintf("%s < %d", key, rapid.IntRange(0, 10).Draw(t, "n"))
	}
}

func drawSelectorString(t *rapid.T) string {
	n := rapid.IntRange(0, 3).Draw(t, "nreqs")
	parts := make([]string, 0, n)
	for i := 0; i < n; i++ {
		parts = append(parts, drawRequirementString(t))
	}
	return strings.Join(parts, ",")
}

// drawLabelSet builds a label set covering some of the generator keys,
// with values from the generator pool plus occasional numerics.
func drawLabelSet(t *rapid.T) Set {
	ls := Set{}
	for _, k := range labelKeys {
		if rapid.Bool().Draw(t, "include "+k) {
			ls[k] = rapid.SampledFrom(labelValues).Draw(t, "val "+k)
		}
	}
	return ls
}

// modelRequirement evaluates one Requirement against a label set
// directly from the documented Matches contract (selector.go doc
// comment on Requirement.Matches, cases 1-5).
func modelRequirement(r *Requirement, ls Set) bool {
	val, exists := ls[r.Key()]
	switch r.Operator() {
	case selection.In, selection.Equals, selection.DoubleEquals:
		return exists && r.Values().Has(val)
	case selection.NotIn, selection.NotEquals:
		return !exists || !r.Values().Has(val)
	case selection.Exists:
		return exists
	case selection.DoesNotExist:
		return !exists
	case selection.GreaterThan, selection.LessThan:
		if !exists {
			return false
		}
		lv, err1 := strconv.ParseInt(val, 10, 64)
		vals := r.Values().List()
		if err1 != nil || len(vals) != 1 {
			return false
		}
		rv, err2 := strconv.ParseInt(vals[0], 10, 64)
		if err2 != nil {
			return false
		}
		if r.Operator() == selection.GreaterThan {
			return lv > rv
		}
		return lv < rv
	}
	return false
}

// TestPBTLabelsParseNeverPanics: labels.Parse consumes selector strings
// from API objects and CLI input; it must error, never panic.
func TestPBTLabelsParseNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.OneOf(
			rapid.String(),
			rapid.StringOf(rapid.RuneFrom([]rune(selectorCharset))),
		).Draw(t, "s")
		_, _ = Parse(s) // a panic here is the failure rapid reports
	})
}

// TestPBTSelectorStringRoundTrip: a parsed selector's String() must
// re-parse to a selector with the same String() (canonical fixed point)
// and identical matching behaviour.
func TestPBTSelectorStringRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := drawSelectorString(t)
		sel, err := Parse(s)
		if err != nil {
			t.Fatalf("generated selector %q failed to parse: %v", s, err)
		}
		str := sel.String()
		sel2, err := Parse(str)
		if err != nil {
			t.Fatalf("String() = %q does not re-parse: %v", str, err)
		}
		if sel2.String() != str {
			t.Fatalf("String() not a fixed point: %q -> %q -> %q", s, str, sel2.String())
		}
		ls := drawLabelSet(t)
		if sel.Matches(ls) != sel2.Matches(ls) {
			t.Fatalf("round trip changed Matches for %v: %q vs %q", ls, str, sel2.String())
		}
	})
}

// TestPBTSelectorMatchesModel: Selector.Matches must equal the
// conjunction of its Requirements, each evaluated by the naive model
// of the documented Requirement.Matches contract.
func TestPBTSelectorMatchesModel(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := drawSelectorString(t)
		sel, err := Parse(s)
		if err != nil {
			t.Fatalf("generated selector %q failed to parse: %v", s, err)
		}
		ls := drawLabelSet(t)
		reqs, ok := sel.Requirements()
		if !ok {
			t.Fatalf("parsed selector %q does not expose Requirements", s)
		}
		want := true
		for i := range reqs {
			if !modelRequirement(&reqs[i], ls) {
				want = false
				break
			}
		}
		if got := sel.Matches(ls); got != want {
			t.Fatalf("Matches(%v) = %v, model says %v for selector %q", ls, got, want, sel.String())
		}
		// Each exposed Requirement's own Matches must agree with the model too.
		for i := range reqs {
			if got := reqs[i].Matches(ls); got != modelRequirement(&reqs[i], ls) {
				t.Fatalf("Requirement %q Matches(%v) = %v, model says %v",
					reqs[i].String(), ls, got, modelRequirement(&reqs[i], ls))
			}
		}
	})
}

// TestPBTSelectorFromSetMatches: SelectorFromSet(ls) must match
// exactly the label sets that contain every entry of ls with the same
// value (its documented purpose: a selector requiring the set).
func TestPBTSelectorFromSetMatches(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		base := drawLabelSet(t)
		candidate := drawLabelSet(t)
		want := true
		for k, v := range base {
			if candidate[k] != v {
				want = false
				break
			}
		}
		if got := SelectorFromSet(base).Matches(candidate); got != want {
			t.Fatalf("SelectorFromSet(%v).Matches(%v) = %v, want %v", base, candidate, got, want)
		}
	})
}
