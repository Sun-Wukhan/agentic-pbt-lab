// Property-based tests for fs.SizeSuffix and fs.CountSuffix.
//
// Lab: Agentic PBT Lab (targets/rclone-rclone) — pristine copy.
// run.sh copies this file into the rclone checkout at fs/pbt_fs_test.go
// and runs it in-package (package fs) so it can use the unexported
// stringRoundTrip, whose doc comment states the round-trip contract
// tested by TestPBTSizeSuffixRoundTripString.
package fs

import (
	"math"
	"testing"

	"pgregory.net/rapid"
)

// suffixCharset produces strings dense in the characters SizeSuffix.Set
// and CountSuffix.Set treat specially: digits, decimal points, exponent
// markers, suffix letters and the letters of NaN/Inf.
const suffixCharset = "0123456789.eE+-KMGTPEBikNaInf"

func drawSuffixToken(t *rapid.T) string {
	mantissa := rapid.SampledFrom([]string{
		"NaN", "Inf", "+Inf", "-Inf", "nan", "inf",
		"0", "1", "1.5", "9.9", "0.1", "1e3", "1e17", "1e18", "1e19", "1e21",
		"9223372036854775807", "18446744073709551615", "999999999999999999999",
	}).Draw(t, "mantissa")
	suffix := rapid.SampledFrom([]string{
		"", "K", "M", "G", "T", "P", "E", "B", "k", "m", "g",
		"Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "KB", "MB", "i",
	}).Draw(t, "suffix")
	return mantissa + suffix
}

// TestPBTSizeSuffixSetNeverPanics: Set must return an error, never panic,
// on arbitrary input. (Robustness; inferred from Set being a flag parser
// for untrusted user input, fs/sizesuffix.go.)
func TestPBTSizeSuffixSetNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.OneOf(
			rapid.String(),
			rapid.StringOf(rapid.RuneFrom([]rune(suffixCharset))),
		).Draw(t, "s")
		var x SizeSuffix
		_ = x.Set(s) // a panic here is the failure rapid reports
	})
}

// TestPBTCountSuffixSetNeverPanics: as above for CountSuffix
// (fs/countsuffix.go).
func TestPBTCountSuffixSetNeverPanics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.OneOf(
			rapid.String(),
			rapid.StringOf(rapid.RuneFrom([]rune(suffixCharset))),
		).Draw(t, "s")
		var x CountSuffix
		_ = x.Set(s)
	})
}

// TestPBTSizeSuffixSetAcceptedValuesSane: Set explicitly rejects negative
// sizes ("size can't be negative") and maps only the literal "off" to -1,
// so any input it accepts must yield a value >= 0 or exactly -1.
// A silently wrapped overflow or a NaN conversion violates this.
func TestPBTSizeSuffixSetAcceptedValuesSane(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := drawSuffixToken(t)
		var x SizeSuffix
		if err := x.Set(s); err == nil {
			if !(x >= 0 || x == -1) {
				t.Fatalf("Set(%q) accepted but produced %d", s, int64(x))
			}
		}
	})
}

// TestPBTCountSuffixSetAcceptedValuesSane: same contract for CountSuffix
// (its Set has the identical negative check, fs/countsuffix.go).
func TestPBTCountSuffixSetAcceptedValuesSane(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := drawSuffixToken(t)
		var x CountSuffix
		if err := x.Set(s); err == nil {
			if !(x >= 0 || x == -1) {
				t.Fatalf("Set(%q) accepted but produced %d", s, int64(x))
			}
		}
	})
}

// TestPBTSizeSuffixRoundTripString: the documented contract of
// stringRoundTrip (fs/sizesuffix.go): "turns SizeSuffix into a string
// which Set parses back to the same value". Holds for every value >= 0
// (negative values print as "off" and are covered separately).
func TestPBTSizeSuffixRoundTripString(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := SizeSuffix(rapid.OneOf(
			rapid.Int64Range(0, 1_000_000),
			rapid.Int64Range(0, math.MaxInt64-1),
		).Draw(t, "v"))
		var y SizeSuffix
		if err := y.Set(v.stringRoundTrip()); err != nil {
			t.Fatalf("Set(%q) errored: %v", v.stringRoundTrip(), err)
		}
		if y != v {
			t.Fatalf("round trip of %d via %q gave %d", int64(v), v.stringRoundTrip(), int64(y))
		}
	})
}

// TestPBTSizeSuffixStringSetIdentity: the naive String()->Set() identity.
// String is the human display form, so this is expected to be the weak
// form of the round-trip property; it is recorded to show exactly where
// display and parsing diverge.
func TestPBTSizeSuffixStringSetIdentity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := SizeSuffix(rapid.Int64Range(0, math.MaxInt64-1).Draw(t, "v"))
		var y SizeSuffix
		if err := y.Set(v.String()); err != nil {
			t.Fatalf("Set(String(%d)=%q) errored: %v", int64(v), v.String(), err)
		}
		if y != v {
			t.Fatalf("String()->Set() of %d via %q gave %d", int64(v), v.String(), int64(y))
		}
	})
}

// TestPBTCountSuffixStringSetIdentity: naive String()->Set() identity
// for CountSuffix (fs/countsuffix.go has no stringRoundTrip analogue).
func TestPBTCountSuffixStringSetIdentity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := CountSuffix(rapid.Int64Range(0, math.MaxInt64-1).Draw(t, "v"))
		var y CountSuffix
		if err := y.Set(v.String()); err != nil {
			t.Fatalf("Set(String(%d)=%q) errored: %v", int64(v), v.String(), err)
		}
		if y != v {
			t.Fatalf("String()->Set() of %d via %q gave %d", int64(v), v.String(), int64(y))
		}
	})
}
