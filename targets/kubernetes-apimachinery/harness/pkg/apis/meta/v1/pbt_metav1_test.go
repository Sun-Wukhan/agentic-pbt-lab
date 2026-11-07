// Property-based tests for metav1.ParseToLabelSelector robustness and
// its agreement with the labels package parser.
//
// Lab: Agentic PBT Lab (targets/kubernetes-apimachinery) — pristine copy.
// run.sh copies this file into the apimachinery checkout at
// pkg/apis/meta/v1/pbt_metav1_test.go and runs it as an external test
// (package v1_test; the directory's package is named v1 and imported
// everywhere as metav1).
package v1_test

import (
	"fmt"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"pgregory.net/rapid"
)

var lsKeys = []string{"app", "tier", "example.com/name", "x"}
var lsValues = []string{"web", "db", "stable", "v1"}

func drawLSRequirement(t *rapid.T, eqValues map[string]string) string {
	key := rapid.SampledFrom(lsKeys).Draw(t, "key")
	val := func(name string) string { return rapid.SampledFrom(lsValues).Draw(t, name) }
	// eqValue returns the value for an equality requirement on key,
	// reusing the first value drawn for that key in this selector.
	// A second, different equality value would trip the confirmed
	// ParseToLabelSelector defect in findings/004 (conflicting
	// equalities silently collapse in the MatchLabels map), which is
	// recorded there with its own reproducer rather than re-failed here.
	eqValue := func(name string) string {
		if v, ok := eqValues[key]; ok {
			return v
		}
		v := val(name)
		eqValues[key] = v
		return v
	}
	switch rapid.IntRange(0, 6).Draw(t, "form") {
	case 0:
		return key + "=" + eqValue("v")
	case 1:
		return key + "!=" + val("v")
	case 2:
		return fmt.Sprintf("%s in (%s, %s)", key, val("v1"), val("v2"))
	case 3:
		return fmt.Sprintf("%s notin (%s)", key, val("v1"))
	case 4:
		return key
	case 5:
		return "!" + key
	default:
		return key + "==" + eqValue("v")
	}
}

func drawLSSelector(t *rapid.T) string {
	n := rapid.IntRange(0, 3).Draw(t, "nreqs")
	parts := make([]string, 0, n)
	eqValues := map[string]string{}
	for i := 0; i < n; i++ {
		parts = append(parts, drawLSRequirement(t, eqValues))
	}
	return strings.Join(parts, ",")
}

func drawLSSet(t *rapid.T) labels.Set {
	ls := labels.Set{}
	for _, k := range lsKeys {
		if rapid.Bool().Draw(t, "include "+k) {
			ls[k] = rapid.SampledFrom(lsValues).Draw(t, "val "+k)
		}
	}
	return ls
}

// TestPBTParseToLabelSelectorNeverPanics: ParseToLabelSelector consumes
// selector strings from API machinery helpers; it must error, never
// panic. Its doc comment requires it to stay in sync with the
// pkg/labels parser.
func TestPBTParseToLabelSelectorNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.OneOf(
			rapid.String(),
			rapid.StringOf(rapid.RuneFrom([]rune("abx=!<>(),.inots-_/ "))),
		).Draw(t, "s")
		_, _ = metav1.ParseToLabelSelector(s) // a panic here is the failure
	})
}

// TestPBTParseToLabelSelectorAgreesWithLabels: when both parsers accept
// a selector, converting the LabelSelector back with
// LabelSelectorAsSelector must produce a selector that matches exactly
// the same label sets as labels.Parse — the two parsers are documented
// to be kept in sync.
func TestPBTParseToLabelSelectorAgreesWithLabels(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := drawLSSelector(t)
		ls, err := metav1.ParseToLabelSelector(s)
		if err != nil {
			t.Skipf("ParseToLabelSelector rejected generated %q: %v", s, err)
		}
		direct, err := labels.Parse(s)
		if err != nil {
			t.Fatalf("labels.Parse rejected %q that ParseToLabelSelector accepted: %v", s, err)
		}
		converted, err := metav1.LabelSelectorAsSelector(ls)
		if err != nil {
			t.Fatalf("LabelSelectorAsSelector(%q): %v", s, err)
		}
		set := drawLSSet(t)
		if converted.Matches(set) != direct.Matches(set) {
			t.Fatalf("parsers disagree on %v for %q: LabelSelector form %q vs direct %q",
				set, s, converted.String(), direct.String())
		}
	})
}

// TestPBTLabelsParseConsistency: labels.Parse and ParseToRequirements
// are two entry points over the same grammar; a selector accepted by
// Parse must expose requirements whose conjunction reproduces its
// Matches behaviour. (Requirement-level model agreement is covered
// in-package by the labels harness.)
func TestPBTLabelsParseConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := drawLSSelector(t)
		sel, err := labels.Parse(s)
		if err != nil {
			t.Fatalf("generated selector %q failed labels.Parse: %v", s, err)
		}
		reqs, err := labels.ParseToRequirements(s)
		if err != nil {
			t.Fatalf("ParseToRequirements rejected %q that Parse accepted: %v", s, err)
		}
		set := drawLSSet(t)
		want := true
		for i := range reqs {
			if !reqs[i].Matches(set) {
				want = false
				break
			}
		}
		if sel.Matches(set) != want {
			t.Fatalf("Parse Matches=%v but requirements conjunction=%v for %q on %v",
				sel.Matches(set), want, s, set)
		}
	})
}
