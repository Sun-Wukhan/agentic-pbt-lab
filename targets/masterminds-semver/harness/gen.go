// Structured generators and the independent SemVer spec model for the
// Masterminds/semver property tests. Lab: Agentic PBT Lab
// (targets/masterminds-semver). This is a separate module; the target
// clone is exercised through a replace directive and stays pristine.
package pbt

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"pgregory.net/rapid"
)

// vspec is a structured version under construction: by construction it
// is always a strict SemVer 2.0 string once rendered.
type vspec struct {
	major, minor, patch uint64
	pre                 string // "" = no prerelease
	meta                string // "" = no metadata
}

func (s vspec) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d.%d.%d", s.major, s.minor, s.patch)
	if s.pre != "" {
		b.WriteString("-")
		b.WriteString(s.pre)
	}
	if s.meta != "" {
		b.WriteString("+")
		b.WriteString(s.meta)
	}
	return b.String()
}

func drawCoreNum(t *rapid.T, label string) uint64 {
	switch rapid.IntRange(0, 9).Draw(t, label+"Kind") {
	case 0, 1, 2, 3, 4, 5:
		return rapid.Uint64Range(0, 30).Draw(t, label+"Small")
	case 6, 7, 8:
		return rapid.Uint64Range(0, 100000).Draw(t, label+"Med")
	default:
		return rapid.Uint64Range(0, ^uint64(0)).Draw(t, label+"Big")
	}
}

// drawPreID draws one valid prerelease identifier: small numeric, huge
// numeric (up to 26 digits — deliberately beyond uint64), or
// alphanumeric per the spec grammar (leading zeros only when the
// identifier is not purely numeric).
func drawPreID(t *rapid.T, label string) string {
	switch rapid.IntRange(0, 11).Draw(t, label+"Kind") {
	case 0, 1, 2, 3:
		return strconv.FormatUint(rapid.Uint64Range(0, 999).Draw(t, label+"Num"), 10)
	case 4, 5:
		n := rapid.IntRange(17, 26).Draw(t, label+"Len")
		var b strings.Builder
		b.WriteByte(byte('1' + rapid.IntRange(0, 8).Draw(t, label+"First")))
		for i := 1; i < n; i++ {
			b.WriteByte(byte('0' + rapid.IntRange(0, 9).Draw(t, label+strconv.Itoa(i))))
		}
		return b.String()
	case 6, 7, 8:
		return rapid.SampledFrom([]string{
			"alpha", "beta", "rc", "dev", "a", "B", "Z", "x-y-z", "0a", "a0", "-", "--", "pre-release",
		}).Draw(t, label+"Word")
	default:
		n := rapid.IntRange(1, 6).Draw(t, label+"Alen")
		const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-"
		var b strings.Builder
		for i := 0; i < n; i++ {
			b.WriteByte(alphabet[rapid.IntRange(0, len(alphabet)-1).Draw(t, label+"A"+strconv.Itoa(i))])
		}
		s := b.String()
		if isNumericID(s) {
			s += "a" // keep it alphanumeric; numeric shape is covered above
		}
		return s
	}
}

func drawMetaID(t *rapid.T, label string) string {
	n := rapid.IntRange(1, 6).Draw(t, label+"Len")
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-"
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteByte(alphabet[rapid.IntRange(0, len(alphabet)-1).Draw(t, label+strconv.Itoa(i))])
	}
	return b.String()
}

func drawVSpec(t *rapid.T, label string) vspec {
	s := vspec{
		major: drawCoreNum(t, label+"Maj"),
		minor: drawCoreNum(t, label+"Min"),
		patch: drawCoreNum(t, label+"Pat"),
	}
	if rapid.IntRange(0, 9).Draw(t, label+"HasPre") < 6 {
		n := rapid.IntRange(1, 3).Draw(t, label+"PreN")
		ids := make([]string, n)
		for i := range ids {
			ids[i] = drawPreID(t, label+"Pre"+strconv.Itoa(i))
		}
		s.pre = strings.Join(ids, ".")
	}
	if rapid.IntRange(0, 9).Draw(t, label+"HasMeta") < 3 {
		n := rapid.IntRange(1, 2).Draw(t, label+"MetaN")
		ids := make([]string, n)
		for i := range ids {
			ids[i] = drawMetaID(t, label+"Meta"+strconv.Itoa(i))
		}
		s.meta = strings.Join(ids, ".")
	}
	return s
}

// drawVSpecNoMeta is drawVSpec with metadata suppressed (constraint
// targets and boundary properties do not use metadata).
func drawVSpecNoMeta(t *rapid.T, label string) vspec {
	s := drawVSpec(t, label)
	s.meta = ""
	return s
}

// preHasOverflowNumericID reports whether a prerelease string contains
// a purely numeric identifier too large for uint64. That is the exact
// pattern of findings/001 (confirmed upstream defect in
// comparePrePart); properties that assert spec-model ordering skip
// those draws so the suites stay green while the finding stands.
func preHasOverflowNumericID(pre string) bool {
	if pre == "" {
		return false
	}
	for _, id := range strings.Split(pre, ".") {
		if isNumericID(id) {
			if _, err := strconv.ParseUint(id, 10, 64); err != nil {
				return true
			}
		}
	}
	return false
}

func mustStrict(t *rapid.T, s string) *semver.Version {
	v, err := semver.StrictNewVersion(s)
	if err != nil {
		t.Fatalf("harness bug: constructed strict version %q rejected: %v", s, err)
	}
	return v
}

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

// --- Independent SemVer 2.0 §11 precedence model (shares no code with
// the target; numeric identifiers compared with math/big). ---

func isNumericID(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func modelComparePreID(a, b string) int {
	if a == b {
		return 0
	}
	an, bn := isNumericID(a), isNumericID(b)
	switch {
	case an && bn:
		ai, _ := new(big.Int).SetString(a, 10)
		bi, _ := new(big.Int).SetString(b, 10)
		return ai.Cmp(bi)
	case an:
		return -1 // numeric < alphanumeric
	case bn:
		return 1
	default:
		if a < b {
			return -1
		}
		return 1
	}
}

func modelComparePre(a, b string) int {
	if a == b {
		return 0
	}
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	for i := 0; i < len(ap) || i < len(bp); i++ {
		if i >= len(ap) {
			return -1 // shorter prefix list is less
		}
		if i >= len(bp) {
			return 1
		}
		if d := modelComparePreID(ap[i], bp[i]); d != 0 {
			return d
		}
	}
	return 0
}

// modelCompare orders two vspec values by SemVer §11, ignoring metadata.
func modelCompare(a, b vspec) int {
	if a.major != b.major {
		return cmpUint(a.major, b.major)
	}
	if a.minor != b.minor {
		return cmpUint(a.minor, b.minor)
	}
	if a.patch != b.patch {
		return cmpUint(a.patch, b.patch)
	}
	if a.pre == "" && b.pre == "" {
		return 0
	}
	if a.pre == "" {
		return 1 // release > prerelease
	}
	if b.pre == "" {
		return -1
	}
	return modelComparePre(a.pre, b.pre)
}

func cmpUint(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
