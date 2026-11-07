# Findings index — kubernetes/apimachinery @ `b26d01364f13520dfdecb540fc63253be5b76662`

Runs: 2026-10-31, rapid v1.2.0, default 100 checks per property, then
2,000 checks for every property (see `../results/`).
24 properties: all pass at 100 and at 2,000 checks in their final form.
Four counterexamples were found along the way and triaged below; three
were defects in our own properties/harness, one is a confirmed upstream
bug. Per this run's instructions, nothing was filed upstream.

| # | Finding | Properties | Verdict |
|---|---------|------------|---------|
| 001 | `intstr` passthrough property drew values outside the type's documented int32 domain (`FromInt(2^31)` truncates to −2^31) | I2 | Wrong property / harness artifact — `FromInt` documents the int32 caller responsibility; property re-ranged to int32. |
| 002 | Labels generator produced values the parser's label-value validation rejects (`app=-2`, `app > -1`) | L2, L3 | Wrong property / harness artifact — negative numbers are not expressible as selector values; generator fixed. |
| 003 | `(a+b)−b != a` when the operands' Dec scales are >29 apart (`9223372036854775807 + 1e30` → …810, off by +3) | Q7 | Wrong property — the finer operand is rounded away from zero by documented design (`amount.go` `maxScaleGap`); Q7 now carries the documented precondition. |
| 004 | `metav1.ParseToLabelSelector` silently drops conflicting equalities on one key: `"tier=db,tier=web"` (matches nothing) converts to `tier=web` (matches) | M2 | **Confirmed bug** — silent semantic weakening in a conversion documented to stay in sync with the labels parser. Found at 2,000 checks; not filed upstream in this run. |

Passed clean at 2,000 checks each (final form): Q1–Q6, Q8–Q13
(Quantity parse robustness; String/CanonicalizeBytes round trips; Cmp
antisymmetry/transitivity/Sign agreement; Add commutativity; Cmp vs
Sub sign; ScaledValue exactness, away-from-zero rounding, saturation,
monotonicity), L1–L4 (labels parse robustness; selector String fixed
point; Matches vs documented-requirement model incl. Gt/Lt;
SelectorFromSet semantics), I1–I4 (intstr robustness; int passthrough;
percent floor/ceil exactness; bounds and monotonicity), M1, M3
(ParseToLabelSelector robustness; labels.Parse vs ParseToRequirements
agreement). M2 passes at 2,000 checks with the findings/004 pattern
excluded from its generator.

Reproduction code for Findings 001–004: `reproducers_test.go.txt`
(plain Go, no rapid; all four verified against the pinned commit).
Full logs: `../results/` — including
`run-2000-checks-m2-failure.log`, the run that caught Finding 004.
