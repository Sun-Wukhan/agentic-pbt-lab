// Property-based tests for fs/fspath.Parse.
//
// Lab: Agentic PBT Lab (targets/rclone-rclone) — pristine copy.
// run.sh copies this file into the rclone checkout at
// fs/fspath/pbt_fspath_test.go (in-package, like path_test.go).
//
// The round-trip invariants are the ones documented on Parsed in
// fs/fspath/path.go ("ConfigString + ':' + Path is equal to the input of
// Parse except that Path may have had \\ converted to /") and asserted
// by the repo's own go-fuzz harness in fs/fspath/fuzz.go.
package fspath

import (
	"path/filepath"
	"reflect"
	"testing"

	"pgregory.net/rapid"
)

// TestPBTParseNeverPanics: Parse consumes untrusted connection strings
// from the command line and config files; it must error, never panic.
func TestPBTParseNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.OneOf(
			rapid.String(),
			rapid.StringOf(rapid.RuneFrom([]rune("ab:,='\"/\\_-.zZ09 \t"))),
		).Draw(t, "s")
		_, _ = Parse(s)
	})
}

// TestPBTParseRoundTrip: on success, Parse splits the input losslessly
// and re-parsing the reconstruction yields an identical Parsed.
func TestPBTParseRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.OneOf(
			rapid.String(),
			rapid.StringOf(rapid.RuneFrom([]rune("ab:,='\"/\\_-.zZ09"))),
		).Draw(t, "s")
		parsed, err := Parse(s)
		if err != nil {
			return // errors are fine; the invariants constrain successes
		}
		if parsed.Name == "" {
			if parsed.ConfigString != "" {
				t.Fatalf("Parse(%q): Name empty but ConfigString=%q", s, parsed.ConfigString)
			}
			if parsed.Path != filepath.ToSlash(s) {
				t.Fatalf("Parse(%q): local path not preserved, got Path=%q", s, parsed.Path)
			}
		} else {
			if parsed.ConfigString+":"+parsed.Path != filepath.ToSlash(s) {
				t.Fatalf("Parse(%q): split %q + ':' + %q does not reconstruct input", s, parsed.ConfigString, parsed.Path)
			}
		}
		reconstructed := parsed.Path
		if parsed.Name != "" {
			reconstructed = parsed.ConfigString + ":" + parsed.Path
		}
		again, err := Parse(reconstructed)
		if err != nil {
			t.Fatalf("Parse(%q) succeeded but re-Parse(%q) errored: %v", s, reconstructed, err)
		}
		if again.Name != parsed.Name || again.ConfigString != parsed.ConfigString || again.Path != parsed.Path || !reflect.DeepEqual(again.Config, parsed.Config) {
			t.Fatalf("Parse not stable: Parse(%q)=%+v, re-Parse=%+v", s, parsed, again)
		}
	})
}
