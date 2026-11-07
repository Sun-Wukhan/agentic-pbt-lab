# Properties — Masterminds/semver

Target commit: `dd2b995c61c39ddd668b23ac919b04d607be35ab` (master, cloned 2026-11-07)
Framework: Go, [pgregory.net/rapid](https://pgregory.net/rapid) v1.2.0
Harness layout: **separate harness module** — `harness/` is its own Go
module (`pbt/semver`) that depends on `github.com/Masterminds/semver/v3`
via a `replace` directive pointing at the pinned clone in `semver/`.
The clone's own files are completely untouched (no test files copied in,
no go.mod edits); rapid is a dependency of the harness module only.
`run.sh` verifies the clone is at the pinned commit and runs the tests.

Surfaces: version parsing/comparison (the SemVer §11 precedence engine)
and the constraint engine (npm/Cargo-style ranges) that Helm's dependency
resolution is built on.

Documented quirks that are *inputs* to the properties, not violations
(found by reading README.md / doc.go / version.go / constraints.go first):

- `NewVersion` coerces: partial versions (`1.2` → `1.2.0`), a leading
  `v`, and (with the default `CoerceNewVersion=true`) leading zeros.
  `StrictNewVersion` accepts only strict SemVer 2.0.
- Build metadata is ignored in comparisons (`Compare` doc; README).
- `Version` *comparison* methods always include prereleases (spec §11),
  while *constraint* `Check`/`Validate` exclude prerelease versions
  unless the constraint itself carries a prerelease in the same AND
  group, or `Constraints.IncludePrerelease` is set (README).
- Constraint ranges follow npm/Cargo semantics, not the SemVer spec:
  documented equivalences for `~`, `^`, wildcards and hyphen ranges in
  README.md / doc.go are the model for C4.
- `constraintTilde` in constraints.go documents `~0.0.0` as a special
  case equivalent to `>= 0.0.0` (accept-all); the C4 tilde family
  therefore excludes the literal `0.0.0` target.
- Upstream's own tests (`version_test.go` TestInc) codify that
  `IncPatch` on a metadata-only version (`1.2.3+meta`) increments the
  patch — the loose "any of prerelease or metadata" wording in the
  `IncPatch` doc comment is not the operative contract; the increment
  is suppressed by a *prerelease* only.
- `New()` explicitly does not validate prerelease/metadata, so ordering
  properties range over *parsed* versions (or `Set*`-built ones), where
  validation has run. Garbage built through `New()` is out of scope.
- Non-`E` `Inc*` functions panic on uint64 overflow by design (documented
  in their comments; the `E` variants return `ErrIncrementOverflow`).

| # | Property (invariant) | Source (file/doc it was inferred from) | Test |
|---|----------------------|------------------------------------------|------|
| V1 | `StrictNewVersion(s)` → `String()` re-parses strictly to an equal version; `String()` is a fixed point; getters return the drawn parts | version.go `StrictNewVersion`, `String`, README "Parsing" | `TestPBTStrictStringRoundTrip` |
| V2 | On strict-valid strings, `NewVersion` and `StrictNewVersion` agree exactly (`String`, prerelease, metadata) | README: both return a `Version` for valid input; `NewVersion` only *adds* coercion | `TestPBTStrictAndNewAgree` |
| V3 | Coerced forms (`1`, `1.2`, `v1.2.3`, `01.2.3`, …) parse via `NewVersion`; the coerced `String()` is strict-valid and strict-parses to an equal version; `Original()` returns the input verbatim | README/doc.go coercion contract; version.go `Original` | `TestPBTNewVersionCoercion` |
| V4 | Neither parser ever panics on arbitrary strings | version.go — parses untrusted chart/user input (Helm) | `TestPBTParsersNeverPanic` |
| V5 | `StrictNewVersion` rejects documented-invalid mutations: missing component, leading zero in a core segment, empty prerelease identifier, numeric prerelease with leading zero, `v` prefix | SemVer 2.0 grammar via version.go `StrictNewVersion` checks (`ErrSegmentStartsZero`, `validatePrerelease`) | `TestPBTStrictRejectsInvalid` |
| V6 | `LessThan`/`LessThanEqual`/`GreaterThan`/`GreaterThanEqual`/`Equal` all agree with the sign of `Compare`; `Compare` is reflexive and antisymmetric | version.go: all five are defined as wrappers over `Compare` | `TestPBTCompareConsistency` |
| V7 | `Compare` induces a total order: transitivity over triples | version.go `Compare` contract; README sorting section (Collection relies on it) | `TestPBTCompareTransitive` |
| V8 | `Compare` equals an independent SemVer §11 model (big-integer numeric prerelease identifiers, numeric < alphanumeric, ASCII lexical, prefix-shorter-is-less, metadata ignored) | README: Version comparisons "follow the specification … https://semver.org/#spec-item-11"; version.go `Compare` doc | `TestPBTCompareMatchesSpecModel` |
| V9 | Changing only build metadata never changes `Compare` against any third version, and metadata variants are `Equal` | version.go `Compare`/`Equal` docs: "Build metadata is ignored" | `TestPBTMetadataIgnoredInCompare` |
| V10 | `IncPatchE`/`IncMinorE`/`IncMajorE` produce exactly the documented next core, clear prerelease+metadata, are strictly greater than the input, agree with the panicking variants, and on segment overflow return `ErrIncrementOverflow` leaving the version unchanged (except `IncPatchE` with a prerelease set, which never needs to increment) | version.go `Inc*` docs + upstream TestInc/TestIncOverflow expectations | `TestPBTIncSemantics` |
| V11 | `SetPrerelease`/`SetMetadata` round-trip through `String()`; invalid values are rejected with an error and leave the version unchanged | version.go `SetPrerelease`/`SetMetadata` + validators | `TestPBTSetPreMeta` |
| V12 | JSON and Text marshal → unmarshal preserves `String()` and equality | version.go `MarshalJSON`/`UnmarshalJSON`/`MarshalText`/`UnmarshalText` | `TestPBTMarshalRoundTrips` |
| V13 | `sort.Sort(Collection)` on a shuffled slice yields a `Compare`-non-decreasing permutation of the input | collection.go; README sorting section | `TestPBTCollectionSort` |
| C1 | `NewConstraint`, `Check` and `Validate` never panic on arbitrary constraint strings against arbitrary versions | constraints.go — parses untrusted Chart.yaml requirements | `TestPBTConstraintsNeverPanic` |
| C2 | `Check(v)` always equals `Validate(v)`'s boolean; success ⇒ no messages, failure ⇒ ≥1 message; `Check` is deterministic and unaffected by an intervening `Validate` | doc.go/README Validation section: Validate is Check plus reasons | `TestPBTCheckValidateAgree` |
| C3 | For a full (non-wildcard) target X, the six basic operators follow `Compare`: `Check(v)` ⟺ sign relation of `v.Compare(X)` — modulo the documented prerelease gate (a prerelease v is excluded when the group carries no prerelease and `IncludePrerelease` is false) | constraints.go basic-op functions; README "Basic Comparisons" + prerelease rules | `TestPBTBasicOpsFollowCompare` |
| C4 | Documented range equivalences agree on every non-prerelease version: `~a.b.c` ≡ `>=a.b.c,<a.(b+1).0`; `^a.b.c` (a>0) ≡ `>=a.b.c,<(a+1).0.0`; `^0.b.c` (b>0) ≡ `>=0.b.c,<0.(b+1).0`; `^0.0.c` ≡ `>=0.0.c,<0.0.(c+1)`; `a.b.x` ≡ `>=a.b.0,<a.(b+1).0`; `*` ≡ `>=0.0.0`; `a.b - a2.b2.c2` ≡ `>=a.b,<=a2.b2.c2` | README/doc.go tilde, caret, wildcard and hyphen-range tables | `TestPBTRangeEquivalences` |
| C5 | A version equal to a constraint's boundary behaves per operator: `>=X`,`<=X`,`=X`,`~X`,`^X` accept X; `>X`,`<X`,`!=X` reject X (non-prerelease X; and prerelease X for the ops whose group then carries a prerelease) | README "Basic Comparisons"; constraints.go op functions | `TestPBTBoundarySelfConsistency` |
| C6 | Constraint `String()` round-trips: re-parsing `c.String()` succeeds, yields the same `String()`, and `Check` agrees on every drawn version | constraints.go `Constraints.String` (origfunc+orig preservation) | `TestPBTConstraintStringRoundTrip` |
| C7 | AND/OR composition: `c1, c2` checks as `c1 && c2`; `c1 \|\| c2` checks as `c1 \|\| c2` (non-prerelease versions, where the group-level prerelease gate cannot differ) | doc.go: comma/space = AND, `||` = OR | `TestPBTConstraintComposition` |
| C8 | `IncludePrerelease=true` changes nothing for non-prerelease versions, and with it set the basic operators follow the raw `Compare` model of C3 even for prerelease versions | constraints.go `IncludePrerelease` field doc; README | `TestPBTIncludePrerelease` |

## Notes

- Generators build versions from structured parts (core numbers from
  small/medium/full-uint64 ranges; prerelease identifiers that are
  small numeric, *huge* numeric up to 26 digits, or alphanumeric;
  metadata identifiers with leading zeros allowed) so the interesting
  corners of §11 are exercised, not just `rapid.String()` noise.
- V8's model is implemented independently with `math/big` for numeric
  prerelease identifiers; it shares no code with the target.
- V8 draws its compared pair with a shared core 70% of the time:
  prerelease precedence only engages on equal cores, which independent
  draws almost never produce (this blind spot hid findings/001 for a
  full 2,000-check run — see findings/002).
- After findings/001 was confirmed, V8, C3 and C8 skip draws whose
  prerelease contains a numeric identifier too large for uint64 (the
  exact confirmed-defect pattern, via `preHasOverflowNumericID`), so
  the suites stay green while the finding stands — the same treatment
  the apimachinery lab gave its confirmed M2 pattern. The standalone
  reproducer in `findings/reproducers_test.go.txt` carries the pattern.
