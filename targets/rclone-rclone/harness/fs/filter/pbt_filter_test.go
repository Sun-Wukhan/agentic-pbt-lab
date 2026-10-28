// Property-based tests for fs/filter glob parsing.
//
// Lab: Agentic PBT Lab (targets/rclone-rclone) — pristine copy.
// run.sh copies this file into the rclone checkout at
// fs/filter/pbt_filter_test.go (in-package, like glob_test.go).
package filter

import (
	"testing"

	"pgregory.net/rapid"
)

// globCharset is dense in the characters globToRegexp (fs/filter/glob.go)
// treats specially, including the {{ raw-regexp }} section markers.
const globCharset = "*?[]{}\\,/ab.:-"

// TestPBTGlobNeverPanics: glob patterns come from user flags and filter
// files; conversion to a regexp must return an error, never panic.
func TestPBTGlobNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		glob := rapid.OneOf(
			rapid.String(),
			rapid.StringOf(rapid.RuneFrom([]rune(globCharset))),
		).Draw(t, "glob")
		ignoreCase := rapid.Bool().Draw(t, "ignoreCase")
		_, _ = GlobPathToRegexp(glob, ignoreCase)
		_, _ = GlobStringToRegexp(glob, true, ignoreCase)
		_, _ = GlobStringToRegexp(glob, false, ignoreCase)
	})
}

// TestPBTGlobLiteralSelfMatch: a glob containing no metacharacters is a
// literal name, and per docs/content/filtering.md it must match itself.
func TestPBTGlobLiteralSelfMatch(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		literal := rapid.StringOfN(
			rapid.RuneFrom([]rune("abcXYZ019._- ")),
			1, 24, -1,
		).Draw(t, "literal")
		re, err := GlobPathToRegexp(literal, false)
		if err != nil {
			t.Fatalf("GlobPathToRegexp(%q) errored: %v", literal, err)
		}
		if !re.MatchString(literal) {
			t.Fatalf("literal glob %q does not match itself (regexp %q)", literal, re.String())
		}
		re2, err := GlobStringToRegexp(literal, true, false)
		if err != nil {
			t.Fatalf("GlobStringToRegexp(%q) errored: %v", literal, err)
		}
		if !re2.MatchString(literal) {
			t.Fatalf("literal glob %q does not match itself (regexp %q)", literal, re2.String())
		}
	})
}

// TestPBTFilterAddNeverPanics: Filter.Add and Filter.AddRule consume
// user-supplied patterns and rule lines; neither must panic.
func TestPBTFilterAddNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		glob := rapid.StringOf(rapid.RuneFrom([]rune(globCharset))).Draw(t, "glob")
		f, err := NewFilter(nil) // nil selects the default options, as in TestNewFilterDefault
		if err != nil {
			t.Fatalf("NewFilter: %v", err)
		}
		_ = f.Add(rapid.Bool().Draw(t, "include"), glob)
		prefix := rapid.SampledFrom([]string{"+ ", "- "}).Draw(t, "prefix")
		_ = f.AddRule(prefix + glob)
	})
}
