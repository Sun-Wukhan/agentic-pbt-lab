# Findings index — Masterminds/semver @ `dd2b995c61c39ddd668b23ac919b04d607be35ab`

Runs: 2026-11-07, rapid v1.2.0, 100 checks per property, then 2,000
checks for every property (see `../results/`). Harness: separate Go
module in `../harness/` (replace directive → pinned clone; the clone
is pristine). The target's own test suite also passes at the pin.
22 properties (V1–V13 incl. V10b, C1–C8): all pass at 100 and at
2,000 checks in their final form. One confirmed upstream bug; two
harness defects found and fixed during bring-up. Per this run's
instructions, nothing was filed upstream.

| # | Finding | Properties | Verdict |
|---|---------|------------|---------|
| 001 | Prerelease numeric identifiers larger than uint64 are compared lexically instead of numerically (`1.0.0-99999999999999999999` vs `1.0.0-100000000000000000000` orders backwards; a huge numeric identifier also beats an alphanumeric one) — `comparePrePart` in version.go classifies numeric identifiers via `strconv.ParseUint` success | V8 | **Confirmed bug** — violates the README-promised SemVer §11 precedence on strict-parser-accepted input. V8/C3/C8 now skip exactly this pattern (overflow numeric prerelease identifiers) so the suites stay green while the finding stands. |
| 002 | Harness/generator blind spots: (a) V3 asserted a patch-dropped spelling denotes the same version (it zero-fills); (b) V8's independent core draws almost never aligned, so prerelease precedence went untested in the first 2,000-check run | V3, V8 | Wrong property / harness artifact — both fixed in the harness; the V8 fix (shared cores) is what exposed finding 001. |

Passed clean at 2,000 checks each (final form): V1–V7, V9–V13
(strict/NewVersion round trips and agreement; coercion + Original;
parser robustness; strict rejection of invalid mutations; Compare
wrapper consistency, antisymmetry, transitivity; metadata ignored;
Inc* semantics incl. overflow errors/panics; SetPrerelease/SetMetadata;
JSON/Text marshal round trips; Collection sorting), V8 within the
uint64-fitting identifier domain, C1–C8 (constraint no-panic;
Check ≡ Validate with message-shape invariants; basic operators
following the Compare model with the documented prerelease gate;
documented ~, ^, wildcard and hyphen-range equivalences; boundary
self-consistency; constraint String round trips; AND/OR composition;
IncludePrerelease semantics).

Reproduction code for finding 001: `reproducers_test.go.txt`
(plain Go, no rapid; verified against the pinned commit — exactly the
3 overflow cases fail, 6 spec controls pass).
Full logs: `../results/` — including
`run-2000-checks-v8-failure.log`, the run that caught finding 001.
