// Property-based tests for Masterminds/semver Version parsing,
// comparison and mutation. See ../PROPERTIES.md for the invariant list
// and where each was inferred from.
package pbt

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"testing"

	semver "github.com/Masterminds/semver/v3"
	"pgregory.net/rapid"
)

// V1: StrictNewVersion → String is a fixed point and re-parses equal.
func TestPBTStrictStringRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := drawVSpec(t, "v")
		str := s.String()
		v := mustStrict(t, str)
		if v.Major() != s.major || v.Minor() != s.minor || v.Patch() != s.patch {
			t.Fatalf("getters mismatch for %q: got %d.%d.%d", str, v.Major(), v.Minor(), v.Patch())
		}
		if v.Prerelease() != s.pre {
			t.Fatalf("prerelease mismatch for %q: got %q want %q", str, v.Prerelease(), s.pre)
		}
		if v.Metadata() != s.meta {
			t.Fatalf("metadata mismatch for %q: got %q want %q", str, v.Metadata(), s.meta)
		}
		if v.String() != str {
			t.Fatalf("String() not a fixed point: %q -> %q", str, v.String())
		}
		v2 := mustStrict(t, v.String())
		if !v.Equal(v2) || v.Compare(v2) != 0 {
			t.Fatalf("re-parse of %q not equal", str)
		}
		if v.Original() != str {
			t.Fatalf("Original() = %q, want the parsed string %q", v.Original(), str)
		}
	})
}

// V2: NewVersion and StrictNewVersion agree on strict-valid strings.
func TestPBTStrictAndNewAgree(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		str := drawVSpec(t, "v").String()
		a := mustStrict(t, str)
		b, err := semver.NewVersion(str)
		if err != nil {
			t.Fatalf("NewVersion rejected strict-valid %q: %v", str, err)
		}
		if a.String() != b.String() || a.Prerelease() != b.Prerelease() || a.Metadata() != b.Metadata() {
			t.Fatalf("NewVersion/StrictNewVersion disagree on %q: %q vs %q", str, a.String(), b.String())
		}
	})
}

// V3: NewVersion coercion of documented loose forms.
func TestPBTNewVersionCoercion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := drawVSpec(t, "v")
		// Build a coerced spelling of the same version.
		core := fmt.Sprintf("%d.%d.%d", s.major, s.minor, s.patch)
		suffix := ""
		if s.pre != "" {
			suffix += "-" + s.pre
		}
		if s.meta != "" {
			suffix += "+" + s.meta
		}
		// Dropping a component denotes the version with that component
		// zero-filled (documented coercion: 1.2 -> 1.2.0), so the
		// expected canonical form zeroes the dropped components.
		want := s
		var loose string
		switch rapid.IntRange(0, 4).Draw(t, "form") {
		case 0: // drop patch (a suffix needs a full core to attach to)
			if suffix != "" {
				loose = core + suffix
			} else {
				loose = fmt.Sprintf("%d.%d", s.major, s.minor)
				want.patch = 0
			}
		case 1: // drop minor+patch
			if suffix != "" {
				loose = core + suffix
			} else {
				loose = fmt.Sprintf("%d", s.major)
				want.minor, want.patch = 0, 0
			}
		case 2: // v prefix
			loose = "v" + core + suffix
		case 3: // leading zeros on core segments (CalVer-style coercion)
			loose = fmt.Sprintf("%02d.%02d.%02d%s", s.major, s.minor, s.patch, suffix)
			if len(loose) > semver.MaxVersionLen {
				loose = core + suffix
			}
		default:
			loose = core + suffix
		}
		if len(loose) > semver.MaxVersionLen {
			return // parser must reject for length; covered by upstream tests
		}
		v, err := semver.NewVersion(loose)
		if err != nil {
			t.Fatalf("NewVersion(%q) errored: %v", loose, err)
		}
		if v.Original() != loose {
			t.Fatalf("Original() = %q, want input %q", v.Original(), loose)
		}
		// The coerced String() must be the canonical strict form and
		// strict-parse to an equal version.
		if v.String() != want.String() {
			t.Fatalf("NewVersion(%q).String() = %q, want canonical %q", loose, v.String(), want.String())
		}
		sv := mustStrict(t, v.String())
		if !sv.Equal(v) {
			t.Fatalf("strict re-parse of %q not equal", v.String())
		}
		if strings.HasPrefix(v.String(), "v") {
			t.Fatalf("String() kept a v prefix: %q", v.String())
		}
	})
}

// V4: parsers never panic on arbitrary strings.
func TestPBTParsersNeverPanic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.StringMatching(`[0-9a-zA-Z.\-+vV*~,^<>=!| ]{0,40}`).Draw(t, "s")
		_, _ = semver.NewVersion(s)
		_, _ = semver.StrictNewVersion(s)
		plain := rapid.String().Draw(t, "plain")
		_, _ = semver.NewVersion(plain)
		_, _ = semver.StrictNewVersion(plain)
	})
}

// V5: StrictNewVersion rejects documented-invalid mutations.
func TestPBTStrictRejectsInvalid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := drawVSpecNoMeta(t, "v")
		s.meta = ""
		core := fmt.Sprintf("%d.%d.%d", s.major, s.minor, s.patch)
		var bad string
		switch rapid.IntRange(0, 4).Draw(t, "mutation") {
		case 0: // missing patch
			bad = fmt.Sprintf("%d.%d", s.major, s.minor)
		case 1: // v prefix
			bad = "v" + core
		case 2: // leading zero in a core segment
			bad = fmt.Sprintf("0%d.%d.%d", s.major, s.minor, s.patch)
		case 3: // empty prerelease identifier
			bad = core + "-alpha..1"
		default: // numeric prerelease with leading zero
			bad = core + "-01"
		}
		if _, err := semver.StrictNewVersion(bad); err == nil {
			t.Fatalf("StrictNewVersion(%q) unexpectedly succeeded", bad)
		}
	})
}

// V6: comparison-method consistency with Compare.
func TestPBTCompareConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := mustStrict(t, drawVSpec(t, "a").String())
		b := mustStrict(t, drawVSpec(t, "b").String())
		c := a.Compare(b)
		if a.LessThan(b) != (c < 0) {
			t.Fatalf("LessThan disagrees with Compare(%q,%q)=%d", a, b, c)
		}
		if a.LessThanEqual(b) != (c <= 0) {
			t.Fatalf("LessThanEqual disagrees with Compare=%d", c)
		}
		if a.GreaterThan(b) != (c > 0) {
			t.Fatalf("GreaterThan disagrees with Compare=%d", c)
		}
		if a.GreaterThanEqual(b) != (c >= 0) {
			t.Fatalf("GreaterThanEqual disagrees with Compare=%d", c)
		}
		if a.Equal(b) != (c == 0) {
			t.Fatalf("Equal disagrees with Compare=%d", c)
		}
		if sign(b.Compare(a)) != -sign(c) {
			t.Fatalf("Compare not antisymmetric: (%q,%q)=%d, reverse=%d", a, b, c, b.Compare(a))
		}
		if a.Compare(a) != 0 || !a.Equal(a) {
			t.Fatalf("Compare not reflexive for %q", a)
		}
	})
}

// V7: Compare transitivity.
func TestPBTCompareTransitive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := mustStrict(t, drawVSpec(t, "a").String())
		b := mustStrict(t, drawVSpec(t, "b").String())
		c := mustStrict(t, drawVSpec(t, "c").String())
		if a.Compare(b) <= 0 && b.Compare(c) <= 0 && a.Compare(c) > 0 {
			t.Fatalf("transitivity violated: %q <= %q <= %q but %q > %q", a, b, c, a, c)
		}
		if a.Compare(b) >= 0 && b.Compare(c) >= 0 && a.Compare(c) < 0 {
			t.Fatalf("transitivity (reverse) violated: %q >= %q >= %q but %q < %q", a, b, c, a, c)
		}
		if a.Compare(b) == 0 && b.Compare(c) == 0 && a.Compare(c) != 0 {
			t.Fatalf("equality not transitive: %q == %q == %q", a, b, c)
		}
	})
}

// V8: Compare equals the independent SemVer §11 model.
func TestPBTCompareMatchesSpecModel(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		sa := drawVSpec(t, "a")
		sb := drawVSpec(t, "b")
		// Prerelease precedence only engages when the cores are equal,
		// which independent draws almost never produce — share the core
		// most of the time so the prerelease comparator is exercised.
		if rapid.IntRange(0, 9).Draw(t, "shareCore") < 7 {
			sb.major, sb.minor, sb.patch = sa.major, sa.minor, sa.patch
		}
		// findings/001 pattern: prerelease numeric identifiers beyond
		// uint64 are misordered by the target (confirmed bug). Skip
		// those draws here; the finding's reproducer carries them.
		if preHasOverflowNumericID(sa.pre) || preHasOverflowNumericID(sb.pre) {
			t.Skip("findings/001 pattern: overflow numeric prerelease identifier")
		}
		a := mustStrict(t, sa.String())
		b := mustStrict(t, sb.String())
		want := sign(modelCompare(sa, sb))
		if got := sign(a.Compare(b)); got != want {
			t.Fatalf("Compare(%q, %q) sign = %d, spec model = %d", sa.String(), sb.String(), got, want)
		}
	})
}

// V9: metadata is ignored by Compare.
func TestPBTMetadataIgnoredInCompare(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := drawVSpec(t, "v")
		base := mustStrict(t, s.String())
		meta := drawMetaID(t, "m1") + "." + drawMetaID(t, "m2")
		withMeta, err := base.SetMetadata(meta)
		if err != nil {
			t.Fatalf("SetMetadata(%q) errored: %v", meta, err)
		}
		if !base.Equal(&withMeta) || base.Compare(&withMeta) != 0 {
			t.Fatalf("metadata changed equality: %q vs %q", base, withMeta.String())
		}
		other := mustStrict(t, drawVSpec(t, "o").String())
		if sign(base.Compare(other)) != sign(withMeta.Compare(other)) {
			t.Fatalf("metadata changed ordering of %q vs %q", base, other)
		}
		if cleared, err := withMeta.SetMetadata(""); err != nil || !cleared.Equal(base) || cleared.Metadata() != "" {
			t.Fatalf("clearing metadata failed: %v %q", err, cleared.String())
		}
	})
}

// V10: Inc* semantics.
func TestPBTIncSemantics(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s := drawVSpec(t, "v")
		v := mustStrict(t, s.String())

		check := func(name string, got semver.Version, err error, wantCore [3]uint64, overflow bool) {
			if overflow {
				if err == nil {
					t.Fatalf("%s(%q): expected overflow error, got %q", name, s.String(), got.String())
				}
				if got.String() != v.String() {
					t.Fatalf("%s(%q): on overflow version must be unchanged, got %q", name, s.String(), got.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("%s(%q): unexpected error %v", name, s.String(), err)
			}
			if got.Major() != wantCore[0] || got.Minor() != wantCore[1] || got.Patch() != wantCore[2] {
				t.Fatalf("%s(%q) = %q, want core %v", name, s.String(), got.String(), wantCore)
			}
			if got.Prerelease() != "" || got.Metadata() != "" {
				t.Fatalf("%s(%q) kept pre/meta: %q", name, s.String(), got.String())
			}
			if !got.GreaterThan(v) {
				t.Fatalf("%s(%q) = %q is not greater than the original", name, s.String(), got.String())
			}
		}

		// IncPatch: increments only when there is no prerelease.
		wantPatch := s.patch
		patchOverflow := false
		if s.pre == "" {
			if s.patch == math.MaxUint64 {
				patchOverflow = true
			} else {
				wantPatch = s.patch + 1
			}
		}
		gotP, errP := v.IncPatchE()
		check("IncPatchE", gotP, errP, [3]uint64{s.major, s.minor, wantPatch}, patchOverflow)
		if !patchOverflow {
			if p := v.IncPatch(); p.String() != gotP.String() {
				t.Fatalf("IncPatch/IncPatchE disagree on %q", s.String())
			}
		}

		minorOverflow := s.minor == math.MaxUint64
		wantMinor := s.minor + 1
		if minorOverflow {
			wantMinor = s.minor
		}
		gotM, errM := v.IncMinorE()
		check("IncMinorE", gotM, errM, [3]uint64{s.major, wantMinor, 0}, minorOverflow)

		majorOverflow := s.major == math.MaxUint64
		wantMajor := s.major + 1
		if majorOverflow {
			wantMajor = s.major
		}
		gotJ, errJ := v.IncMajorE()
		check("IncMajorE", gotJ, errJ, [3]uint64{wantMajor, 0, 0}, majorOverflow)
	})
}

// V10b: the panicking Inc variants panic exactly on documented overflow.
func TestPBTIncPanicsOnOverflow(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		seg := rapid.IntRange(0, 2).Draw(t, "seg")
		withPre := rapid.Bool().Draw(t, "withPre")
		s := vspec{major: 1, minor: 2, patch: 3}
		switch seg {
		case 0:
			s.major = math.MaxUint64
		case 1:
			s.minor = math.MaxUint64
		default:
			s.patch = math.MaxUint64
		}
		if withPre {
			s.pre = "alpha"
		}
		v := mustStrict(t, s.String())
		// IncPatch with a prerelease never increments, hence never overflows.
		expectPanic := !(seg == 2 && withPre)
		func() {
			defer func() {
				r := recover()
				if expectPanic && r == nil {
					t.Fatalf("expected panic incrementing %q (seg %d)", s.String(), seg)
				}
				if !expectPanic && r != nil {
					t.Fatalf("unexpected panic incrementing %q: %v", s.String(), r)
				}
			}()
			switch seg {
			case 0:
				_ = v.IncMajor()
			case 1:
				_ = v.IncMinor()
			default:
				_ = v.IncPatch()
			}
		}()
	})
}

// V11: SetPrerelease / SetMetadata.
func TestPBTSetPreMeta(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v := mustStrict(t, drawVSpec(t, "v").String())
		pre := drawPreID(t, "p1") + "." + drawPreID(t, "p2")
		nv, err := v.SetPrerelease(pre)
		if err != nil {
			t.Fatalf("SetPrerelease(%q) on %q errored: %v", pre, v, err)
		}
		if nv.Prerelease() != pre {
			t.Fatalf("SetPrerelease round trip: got %q want %q", nv.Prerelease(), pre)
		}
		reparsed := mustStrict(t, nv.String())
		if reparsed.Prerelease() != pre || !reparsed.Equal(&nv) {
			t.Fatalf("SetPrerelease result %q does not re-parse faithfully", nv.String())
		}
		for _, bad := range []string{"a..b", "01", "a b", ".a", "a.", "é"} {
			if _, err := v.SetPrerelease(bad); err == nil {
				t.Fatalf("SetPrerelease(%q) unexpectedly accepted", bad)
			}
			if _, err := v.SetMetadata(bad); err == nil {
				// metadata has a different grammar: "01" and ".a" are the
				// interesting rejects; "a..b"/"a." have empty identifiers.
				if bad == "01" {
					continue
				}
				t.Fatalf("SetMetadata(%q) unexpectedly accepted", bad)
			}
		}
	})
}

// V12: JSON/Text marshal round trips.
func TestPBTMarshalRoundTrips(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		str := drawVSpec(t, "v").String()
		v := mustStrict(t, str)

		jb, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("json.Marshal(%q): %v", str, err)
		}
		var jv semver.Version
		if err := json.Unmarshal(jb, &jv); err != nil {
			t.Fatalf("json.Unmarshal(%s): %v", jb, err)
		}
		if jv.String() != str {
			t.Fatalf("JSON round trip: %q -> %s -> %q", str, jb, jv.String())
		}

		tb, err := v.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText(%q): %v", str, err)
		}
		var tv semver.Version
		if err := tv.UnmarshalText(tb); err != nil {
			t.Fatalf("UnmarshalText(%q): %v", tb, err)
		}
		if tv.String() != str {
			t.Fatalf("Text round trip: %q -> %q", str, tv.String())
		}
	})
}

// V13: Collection sorting.
func TestPBTCollectionSort(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 12).Draw(t, "n")
		vs := make(semver.Collection, n)
		counts := map[string]int{}
		for i := range vs {
			s := drawVSpec(t, fmt.Sprintf("v%d", i)).String()
			vs[i] = mustStrict(t, s)
			counts[s]++
		}
		sort.Sort(vs)
		for i := range vs {
			counts[vs[i].String()]--
			if i > 0 && vs[i-1].Compare(vs[i]) > 0 {
				t.Fatalf("sorted collection decreases at %d: %q > %q", i, vs[i-1], vs[i])
			}
		}
		for s, c := range counts {
			if c != 0 {
				t.Fatalf("sort is not a permutation: %q count off by %d", s, c)
			}
		}
	})
}
