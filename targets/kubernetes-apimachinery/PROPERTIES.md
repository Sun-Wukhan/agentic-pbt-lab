# Properties — kubernetes/apimachinery

Target commit: `b26d01364f13520dfdecb540fc63253be5b76662` (master, cloned 2026-10-31)
Framework: Go, [pgregory.net/rapid](https://pgregory.net/rapid) v1.2.0
Harness layout: files under `harness/` mirror their in-repo paths; `run.sh`
copies them into a pristine clone of apimachinery at the pinned commit, adds
`pgregory.net/rapid` as a test dependency *in that clone only*, and runs them.

Surfaces: the serialisation/parsing family — `resource.Quantity`,
`labels` selectors, `intstr` scaling, and the `metav1` selector parser
that is documented to mirror `labels`.

| # | Property (invariant) | Source (file/doc it was inferred from) | Test |
|---|----------------------|------------------------------------------|------|
| Q1 | `ParseQuantity` never panics on arbitrary strings (error is fine) | `pkg/api/resource/quantity.go` — parses untrusted API/user input | `TestPBTParseQuantityNeverPanics` |
| Q2 | Parse→`String()`→parse preserves value (`Cmp == 0`) and `String()` is a fixed point | `quantity.go` type doc: canonical form loses "no precision"; non-canonical input "will be re-emitted in their canonical form" | `TestPBTQuantityStringRoundTrip` |
| Q3 | `CanonicalizeBytes` output re-parses to the same value, and canonicalising the re-parse is byte-identical | `quantity.go` `CanonicalizeBytes` / canonical-form doc | `TestPBTQuantityCanonicalizeRoundTrip` |
| Q4 | `Cmp` is antisymmetric and reflexive; `Sign()` agrees with `Cmp(q, 0)`; `IsZero()` agrees with `Sign()` | `quantity.go` `Cmp`/`Sign` doc comments | `TestPBTQuantityCmpAntisymmetricAndSign` |
| Q5 | `Cmp` induces a total order (transitivity) | `quantity.go` `Cmp` contract | `TestPBTQuantityCmpTransitive` |
| Q6 | `Add` is commutative: `a+b == b+a` under `Cmp` | `quantity.go` `Add` doc (plain addition of values) | `TestPBTQuantityAddCommutative` |
| Q7 | `(a+b)−b == a` and `(a−b)+b == a` whenever the documented bounded-scale rounding does not engage (Dec-scale gap ≤ 29) | `quantity.go` `Add`/`Sub` docs + `amount.go` `maxScaleGap` comment (finer operand rounded away from zero beyond a 29-order scale gap, by design — see findings/003) | `TestPBTQuantityAddSubInverse` |
| Q8 | `a+0 == a`; `a−a == 0` | `quantity.go` `Add`/`Sub` semantics | `TestPBTQuantityAddZeroSubSelf` |
| Q9 | `sign(a−b) == sign(Cmp(a,b))` | `quantity.go` `Cmp`/`Sub` docs | `TestPBTQuantityCmpVsSubSign` |
| Q10 | `ScaledValue(s) == v` exactly for a quantity built by `NewScaledQuantity(v, s)`; `AsScaledInt64` agrees and reports ok | `quantity.go` `ScaledValue` doc: "returns the value of q/10^scale" | `TestPBTQuantityScaledValueExact` |
| Q11 | `ScaledValue` rounds away from zero (milli value read at scale 0); `MilliValue()` of a milli-built quantity is exact | `quantity.go` `ScaledValue`/`MilliValue` docs: "rounded to an integer, away from zero" | `TestPBTQuantityScaledValueRounding` |
| Q12 | `ScaledValue` saturates to `MinInt64`/`MaxInt64` on overflow; `AsScaledInt64` reports ok=false | `quantity.go` `Value`/`ScaledValue` docs: "saturates to math.MinInt64 or math.MaxInt64" | `TestPBTQuantityScaledValueSaturates` |
| Q13 | `ScaledValue` at a fixed scale is monotone in `Cmp` order | `quantity.go` `ScaledValue` doc (away-from-zero rounding and saturation both preserve order) | `TestPBTQuantityScaledValueMonotonic` |
| L1 | `labels.Parse` never panics on arbitrary strings | `pkg/labels/selector.go` — parses selectors from API objects and CLI input | `TestPBTLabelsParseNeverPanics` |
| L2 | Parsed selector's `String()` re-parses to the same `String()` and identical `Matches` behaviour | `pkg/labels/selector.go` `internalSelector.String()` (sorted, canonical rendering) | `TestPBTSelectorStringRoundTrip` |
| L3 | `Selector.Matches` equals the conjunction of its `Requirements`, each matching the naive model of the documented `Requirement.Matches` contract (cases 1–5, incl. Gt/Lt) | `pkg/labels/selector.go` `Requirement.Matches` doc comment | `TestPBTSelectorMatchesModel` |
| L4 | `SelectorFromSet(ls)` matches exactly the label sets containing every entry of `ls` with equal value | `pkg/labels/selector.go` `SelectorFromSet` | `TestPBTSelectorFromSetMatches` |
| I1 | `GetScaledValueFromIntOrPercent` never panics for arbitrary `IntOrString` values and totals | `pkg/util/intstr/intstr.go` — consumes rolling-update/probe values from API objects | `TestPBTIntstrScaledNeverPanics` |
| I2 | A non-percent `IntOrString` is returned unchanged for any total and rounding mode | `pkg/util/intstr/intstr.go` `GetScaledValueFromIntOrPercent` (int values used as-is) | `TestPBTIntstrIntPassthrough` |
| I3 | Percent results equal the exact mathematical floor/ceil of `p·total/100` (within bounds where the implementation's float64 product is exact) | `pkg/util/intstr/intstr.go` doc: percentage of total, rounded per `roundUp` | `TestPBTIntstrPercentExact` |
| I4 | For p ∈ [0,100], the result lies in [0, total] and is non-decreasing in total | `pkg/util/intstr/intstr.go` semantics of a percentage of a total | `TestPBTIntstrPercentBoundsAndMonotonic` |
| M1 | `metav1.ParseToLabelSelector` never panics on arbitrary strings | `pkg/apis/meta/v1/helpers.go` — "should be kept in sync with the parser in pkg/labels/selector.go" | `TestPBTParseToLabelSelectorNeverPanics` |
| M2 | When both parsers accept a selector, `LabelSelectorAsSelector(ParseToLabelSelector(s))` matches the same label sets as `labels.Parse(s)` — except for conflicting duplicate equalities on one key, which trip the confirmed defect in findings/004 and are excluded from the generator | `pkg/apis/meta/v1/helpers.go` sync note + `labels` semantics | `TestPBTParseToLabelSelectorAgreesWithLabels` |
| M3 | `labels.Parse` and `labels.ParseToRequirements` agree: `Matches` equals the conjunction of the returned requirements' `Matches` | `pkg/labels/selector.go` — two entry points over one grammar | `TestPBTLabelsParseConsistency` |

## Notes

- Documented parse normalisations are *inputs* to Q2/Q3, not violations:
  the type doc states parsing caps BinarySI quantities silently at
  2^63−1 ("8Ei" parses as 2^63−1), keeps at most 29 significant digits,
  and rounds sub-nano precision away from zero ("0.9n" becomes "1n").
  Q2/Q3 therefore assert stability of the *parsed value's* printed form,
  never identity with the input string.
- `Add`/`Sub` mutate the receiver, so every arithmetic property works on
  copies (Quantity is a value type; copies are independent per the type
  doc).
- I3 deliberately bounds p and total so `p·total` is exactly
  representable in float64; a mismatch inside those bounds cannot be
  attributed to binary floating-point representation of the product.
