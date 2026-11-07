# Finding: two harness/generator defects found during bring-up (no upstream impact)

- Target / property: Masterminds/semver lab harness itself, during the
  2026-11-07 bring-up runs.
- Minimal counterexample:
  1. V3 (coercion) failed at 100 checks with
     `NewVersion("0.0").String() = "0.0.0", want canonical "0.0.1"` —
     the harness's "drop patch" spelling was asserted to denote the
     *same* version, but dropping a component denotes the version with
     that component zero-filled. The library was right; the property's
     expectation was wrong. Fixed by zeroing the dropped components in
     the expected canonical form.
  2. V8 (spec model) passed its first 2,000-check run despite the
     confirmed defect in findings/001 being trivially reproducible by
     hand: two independently drawn versions almost never share all
     three core numbers, so prerelease precedence — the only place the
     defect lives — was effectively never compared. Once the generator
     shared the core between the pair 70% of the time, rapid found a
     counterexample on the first check.
- Observed vs expected: both are defects in our test rig, recorded per
  lab practice of triaging counterexamples in the open. (2) is the
  more general lesson for this lab: a model-equivalence property over
  pairs is only as strong as the generator's ability to reach the
  region where the compared functions actually diverge; "passed 2,000
  checks" meant "sampled 2,000 points, ~none in the prerelease
  comparator".
- Verdict: wrong property / harness artifact (both). Fixed in the
  harness; no upstream implication.
- Upstream issue: none. Fix PR: none (harness changes are in
  `../harness/`).
