# Findings index — pypa/packaging @ 713d0d78 (26.4.dev0)

Runs: 2026-11-07, Hypothesis 6.170.15, 500 examples per property
(default profile), then 2,000 examples per property (long profile) —
see `../results/`.
36 properties (43 test functions): all passed clean at both 500 and
2,000 examples. Two counterexamples survived triage as confirmed bugs;
both are also encoded as deliberately-failing tests
(`../harness/test_finding_001.py`, `../harness/test_finding_002.py`).

| # | Finding | Properties | Verdict |
|---|---------|------------|---------|
| 001 | `Requirement` rejects PEP 508-valid strings when a `===` specifier is followed by another specifier containing a space (`x===0,>= 0`): the tokenizer's SPECIFIER rule reuses the standalone-specifier regex whose `===` version class `[^\s;)]+` swallows the comma and next operator. The no-space form parses only because `SpecifierSet` later re-splits the swallowed token on commas. Also present in packaging 25.0 — long-standing, not a 26.x regression. | R3, R4 | **Confirmed bug** — acceptance flips on insignificant whitespace, contradicting PEP 508's grammar (`version` excludes `,`/operators; `version_many` is comma-separated) and the parser's own documented grammar. |
| 002 | `Version.from_parts` / `__replace__` validators accept `bool` parts: `from_parts(epoch=True, release=(1,))` yields a version with `str() == 'True!1'`, which cannot be re-parsed; `release=(True, 2)` gives `str() == 'True.2'` while comparing equal to `Version('1.2')`. | V8 (manual probe) | **Confirmed bug (low severity)** — the one wrong type the isinstance-based validators structurally miss; the product violates the documented str round-trip invariant instead of raising `InvalidVersion`. Reachability is programmatic, stated plainly in the finding. |

Passed clean (2,000 examples each): V1–V10 (str round-trip, messy
spelling normalisation, trichotomy/operator coherence, transitivity +
sorting, hash/eq, PEP 440 family order incl. local-segment rules,
attribute coherence, from_parts/__replace__ agreement on genuine ints,
pickle round-trip, canonicalize_version), S1–S12 (constructed
witnesses for every operator, Specifier vs SpecifierSet agreement,
contains-vs-filter agreement, the PEP 440 filter list rule incl. the
autodetect-True exception, `&` combination, str round-trips,
pre-release policy, `is_unsatisfiable` soundness, `===` literal
equality, VersionRange algebra vs membership incl.
`to_specifier_set()` round-trip, subset/disjoint sampling soundness,
canonical equality), R1–R4 as scoped (see below), M1–M7 (marker
round-trip, & / | algebra vs evaluate, tautology/contradiction,
version-key comparisons vs Specifier, membership, PEP 685 extra
normalisation, default-environment sanity), U1–U2 (canonicalize_name
vs the PEP 503 regex model + idempotence, is_normalized_name), N1
(parsers raise only their documented exceptions).

Scoping note: after Finding 001 was triaged, the
`requirement_strings` strategy generates `===` only in last position
for R1–R4 (the bug class has its own tests); everything else in the
requirement space runs unscoped. Documented quirks deliberately not
reported: marker ordering comparisons on non-version keys degrading
to False/equality (`markers._operators`, long-standing); PEP 440
local ordering; the int-digit-limit `ValueError` the version parser
intentionally propagates (code comment in `Version.__init__`).
