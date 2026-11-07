# Properties — kubernetes/apimachinery `resource.Quantity`, wave 2

Target: `k8s.io/apimachinery/pkg/api/resource` at the lab clone
(`../apimachinery`, label-selector branch HEAD `2208717e`; `pkg/api/resource`
is byte-identical to the wave-1 pin `b26d01364f13520dfdecb540fc63253be5b76662`
— verified with `git diff`).
Framework: Go, [pgregory.net/rapid](https://pgregory.net/rapid) v1.2.0, in a
separate harness module (`go.mod` here) that `replace`s
`k8s.io/apimachinery` with the local clone. The clone's `go.mod` is never
edited.

Wave 1 (`../PROPERTIES.md`, Q1–Q13) covered quantities drawn from parsed
strings: parse/print/canonical round-trips, `Cmp` order laws, `Add`/`Sub`,
`ScaledValue`. This wave covers what wave 1 did not: the **serialisation
formats** (JSON, protobuf, CBOR), **constructor-built quantities**
(`NewScaledQuantity`, `NewMilliQuantity`, `NewQuantity`,
`NewDecimalQuantity`, `SetScaled`) including extreme scales, **backend
model equivalence** (the int64 fast path vs the `inf.Dec` model the package
itself falls back to), the **`Mul`/`Neg`/`RoundUp` algebra**, **float
conversion**, and **copy independence**.

| # | Property (invariant) | Source (file/doc it was inferred from) | Test |
|---|----------------------|------------------------------------------|------|
| R1 | `MarshalJSON` → `UnmarshalJSON` preserves value (`Cmp == 0`) and the printed form is a fixed point; same via `encoding/json` | `quantity.go` `MarshalJSON`/`UnmarshalJSON` | `TestPBTQuantityJSONRoundTrip` |
| R2 | `UnmarshalJSON` accepts bare JSON numbers equal to `ParseQuantity` of the same text; `null` unmarshals to zero; arbitrary bytes never panic | `quantity.go` `UnmarshalJSON` (null special case, optional quotes) | `TestPBTQuantityJSONBareNumbersAndNull`, `TestPBTQuantityUnmarshalNeverPanics` |
| R3 | Protobuf `Marshal` → `Unmarshal` preserves value; `Size()` equals the marshalled length; marshalling is deterministic | `quantity_proto.go` (custom string-field encoding of `String()`) | `TestPBTQuantityProtoRoundTrip` |
| R4 | CBOR `MarshalCBOR` → `UnmarshalCBOR` preserves value; `null` → zero | `quantity.go` `MarshalCBOR`/`UnmarshalCBOR` | `TestPBTQuantityCBORRoundTrip` |
| R5 | Backend model equivalence: `Cmp` sign equals `AsDec().Cmp()` sign on mixed-constructor pairs; `AsDec` is a defensive copy (mutating it leaves the quantity unchanged); `NewDecimalQuantity(*q.AsDec(), q.Format)` is `Cmp`-equal to `q` | `quantity.go` `Cmp` (int64 fast path vs `cmpDec` fallback), `AsDec` doc ("returned quantity is a copy") | `TestPBTQuantityDecModelEquivalence` |
| R6 | `Mul(y)` always yields the exact mathematical product (cross-checked against a `big.Rat` model); `Mul(1)` identity, `Mul(0)` zero, `Mul(-1)` equals `Neg` | `quantity.go` `Mul` doc ("return false if the result is inexact" — the value itself is computed exactly via the Dec fallback) | `TestPBTQuantityMulExact` |
| R7 | `Neg` is an involution and reverses order; `a + Neg(a) == 0`; `a.Sub(b) == a.Add(Neg(b))` exactly | `quantity.go` `Neg`/`Add`/`Sub` docs (bounded-scale rule is symmetric in the operands' scales, and `Neg` preserves scale) | `TestPBTQuantityNegAlgebra` |
| R8 | After `RoundUp(s)`: `AsScale(s)` reports exact, `RoundUp` is idempotent, sign is preserved, and the value moves only away from zero | `quantity.go` `RoundUp` doc ("rounded away from zero", false on precision loss) | `TestPBTQuantityRoundUpContract` |
| R9 | Re-rendering under any of the three `Format`s (via the exported `Format` field) produces a string that re-parses to the same value | `quantity.go` type doc + `CanonicalizeBytes` format fallbacks | `TestPBTQuantityFormatRerender` |
| R10 | `CanonicalizeBytes` is a fixed point (reparse → recanonicalise is byte-identical) for constructed quantities of all formats, and a caller-supplied `out` buffer is honoured as a prefix for non-zero values | `quantity.go` `CanonicalizeBytes` doc | `TestPBTQuantityCanonicalizeConstructed` |
| R11 | Copy independence: copies made by assignment, `DeepCopy()` and `DeepCopyInto` are unaffected when the other copy is mutated via `Add`/`Sub`/`Neg`/`Mul`/`RoundUp`/`SetScaled` | `quantity.go` type doc ("Quantity is a value type. Each shallow copy is independent from its source") | `TestPBTQuantityCopyIndependence` |
| R12 | `AsFloat64Slow` returns the correctly rounded float64 of the exact value (cross-checked with a `big.Rat` model); `AsApproximateFloat64` at least agrees in sign and zero-ness in the moderate range | `quantity.go` `AsFloat64Slow` doc ("more precise than AsApproximateFloat64") | `TestPBTQuantityFloat64SlowExact` |
| R13 | `CmpInt64(v)` agrees with `Cmp(*NewQuantity(v, DecimalSI))` | `quantity.go` `CmpInt64` doc | `TestPBTQuantityCmpInt64Consistent` |
| R14 | `NewScaledQuantity(v, s)` / `SetScaled(v, s)`: `ScaledValue(s) == v` and `AsScaledInt64(s)` reports ok at every scale, including near the int32 extremes; `String()` re-parses to the same value whenever the value is an exact multiple of 1n | `quantity.go` `ScaledValue`/`AsScaledInt64` docs (incl. the documented near-MinInt32 caveat for the Dec backend, so this property uses the int64 backend as constructed) | `TestPBTQuantityExtremeScales` |
| R15 | An exponent suffix whose value falls outside the representable scale range must be rejected with an error, never silently narrowed by wraparound: `ParseQuantity("1e<k>")` for k outside `[MinInt32+1, MaxInt32]` errors | `suffix.go` `interpret` (narrowing comment) + `quantity.go` `ParseQuantity` (the `MinInt32` exponent already errors via `canInfScale`) — **FAILS: finding 001** | `TestPBTQuantityExponentRange` (directed) |
| R16 | `String()` of any constructed quantity (any format) re-parses to the same value, under the documented precondition that the value is an exact multiple of 1n (parsing floors finer precision at nano by design) | `quantity.go` type doc (canonical form loses "no precision"; parse nano floor) | `TestPBTQuantityStringRoundTripConstructed` |
| R17 | The zero value is well-behaved: `String() == "0"`, JSON `"0"`, proto/CBOR round-trips, `Sign() == 0`, `Cmp` equal to `ParseQuantity("0")` | `quantity.go` zero handling (`CanonicalizeBytes` zero case, `UnmarshalJSON` null) | `TestPBTQuantityZeroValue` |
| R18 | Adding to / subtracting from a zero quantity adopts the other operand's `Format` | `quantity.go` `Add`/`Sub` docs ("If the current value is zero, the format of the quantity will be updated to the format of y") | `TestPBTQuantityZeroFormatAdoption` |

## Notes

- The parse nano floor is *documented* (type doc: parsing "preserves a
  value only down to nano, rounding a finer non-zero value away from
  zero"), so every round-trip-through-parse property (R9, R10, R14, R16)
  carries the precondition `q.AsScale(Nano)` reports exact. A
  sub-nano constructed quantity (e.g. `NewScaledQuantity(12345, -30)`)
  serialises to `"12345e-30"` and re-parses as `1n`; that asymmetry is
  intended, recorded as finding 002 (wrong property, not a bug).
- The BinarySI parse cap (2^63−1) and the 29-significant-digit rule are
  likewise documented inputs, as in wave 1. The cap also bounds
  serialisation round-trips: a constructed BinarySI quantity above the
  cap (e.g. `9765625000000000Ki`) prints faithfully but re-parses capped
  (finding 003, documented behaviour, not a bug). R1/R3/R4/R9/R10 carry
  that precondition.
- Canonical form is a byte fixed point only *within* a format: a
  BinarySI quantity whose base-1024 exponent is 0 (e.g. the value 10000)
  canonicalises to a suffix-less `"10000"`, which re-parses as DecimalSI
  and re-renders as `"10k"`. The value round-trips; the format does not.
  R10 asserts byte identity only when the format survives.
- `Add`/`Sub`/`Neg`/`Mul`/`RoundUp` mutate the receiver; every property
  works on copies.
- R12's exactness model bounds the decimal exponent to ±300 so the
  `big.Rat` cross-check stays cheap; float conversion of astronomically
  scaled values saturates to ±Inf/0 by design.
