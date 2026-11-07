# Upstream issue draft — kubernetes/apimachinery (prepared Oct 31, 2026)

STATUS: NOT filed. The lab finding records "reports only on Navid's
instruction" — this draft waits for Navid's go. Source finding:
findings/004-parsetolabelselector-drops-conflicting-equalities.md
(apimachinery @ b26d01364f13520dfdecb540fc63253be5b76662).

VERIFIED independently Nov 1, 2026 (standalone reproducer, not the lab
harness): at the exact pinned commit (module version
v0.36.0-alpha.2.0.20261031004447-b26d01364f13) AND at the latest
release v0.37.2 — ParseToLabelSelector("tier=db,tier=web") →
MatchLabels{tier: web}, no error; labels.Parse matches nothing on
{tier: web} while the converted selector matches it; reverse order
keeps tier=db; same-value duplicates convert faithfully. So the bug
is confirmed as originally found and still unfixed in v0.37.2.
Scratch reproducer: ~/workspace/.gotmp/apiverify/main.go.

## Suggested title
metav1.ParseToLabelSelector silently drops conflicting equality requirements on the same key

## Body (paste-ready)

**What happened**

`metav1.ParseToLabelSelector("tier=db,tier=web")` returns
`MatchLabels{tier: web}` with no error. Converted back via
`LabelSelectorAsSelector`, that selector matches `{tier: web}`.

`labels.Parse("tier=db,tier=web")` keeps both requirements, so the
selector is a contradiction and matches nothing.

The conversion therefore silently *widens* what the selector selects
for. Reversing the input order (`tier=web,tier=db`) keeps `tier=db`
instead — the survivor is whichever equality comes last.

**Why**

`ParseToLabelSelector` (pkg/apis/meta/v1/helpers.go) routes
Equals/DoubleEquals requirements into the `MatchLabels` map with a
plain assignment, so a second conflicting equality for the same key
overwrites the first. The function already returns errors for other
unrepresentable inputs ("equals operator must have exactly one
value"), and its doc comment says it "should be kept in sync with
the parser in pkg/labels/selector.go" — it just never checks for a
conflicting duplicate.

Verified non-issues: duplicate equalities with the *same* value, and
equality combined with `in`/`notin`/exists requirements on the same
key, all convert faithfully. Only conflicting Equals/DoubleEquals
pairs collide.

**Expected behaviour**

Either reject a conflicting duplicate equality with an error (matching
how the function already treats unrepresentable input), or preserve
the semantics losslessly — e.g. route the subsequent equality for an
already-mapped key into `MatchExpressions` as an `In` requirement.

**Severity / context**

Low severity, degenerate input (a repeated equality key with
different values, most likely from programmatically merged
requirement lists rather than hand-written selectors) — but the
failure mode is silent widening in a conversion every Kubernetes
tool depends on, found via property-based differential testing
against `labels.Parse`.

## Fix options (for the follow-up PR, Navid's call)
1. Error on conflicting duplicate — smallest change, matches existing
   error behaviour for unrepresentable input.
2. Lossless conversion via MatchExpressions `In` — preserves all
   inputs, slightly larger change.
Lab reproducer: harness `reproducers_test.go.txt` in this target dir.
