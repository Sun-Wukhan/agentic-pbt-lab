# Finding 001 — CONFIRMED BUG: `ParseQuantity` silently wraps out-of-range exponent suffixes modulo 2^32

- Target / property: kubernetes/apimachinery, `pkg/api/resource` at the
  wave-1 pin `b26d01364f13520dfdecb540fc63253be5b76662` (lab clone HEAD
  `2208717e`; the package is byte-identical at both) — property R15
  (`TestPBTQuantityExponentRange`, directed) in `../PROPERTIES.md`.
- Verdict: **CONFIRMED — true bug.** Silent value corruption in the
  parser of untrusted input; no documentation sanctions it (see
  "Why this is not documented behaviour" below).

## Minimal reproducer

Plain Go test: `../repro_exponent_test.go` (`TestReproExponentWraparound`,
fails at the pinned commit). Core cases, all returning `err == nil`:

| Input | Denotes | Parser produces |
|---|---|---|
| `1e4294967296` | 10^4294967296 | `1` (exponent 2^32 wraps to 0) |
| `1e4294967297` | 10^(2^32+1) | `10` (wraps to 1) |
| `1e4294967295` | 10^(2^32−1) | `100m` = 0.1 (wraps to −1; direction flips) |
| `1e-4294967296` | 10^−(2^32) ≈ 0 | `1` (wraps to 0) |
| `2e8589934592` | 2·10^(2^33) | `2` (wraps to 0) |
| `3e12884901890` | 3·10^(3·2^32+2) | `300` (wraps to 2) |
| `1.5e4294967297` | 1.5·10^(2^32+1) | `15` (Dec path — not just the int64 fast path) |

Controls that behave correctly today: `1e2147483647` (MaxInt32) parses
to the right value; `1e2147483648` and `1e-2147483648` are rejected with
`ErrSuffix`, because narrowing lands exactly on `math.MinInt32`, whose
scale `ParseQuantity` refuses via `Scale.canInfScale()`.

A secondary symptom: when the wrapped exponent makes the input look
canonical to the int64 fast path, the quantity's cached `String()`
echoes the *input spelling* (e.g. `ParseQuantity("1e4294967296").String()`
is `"1e4294967296"` while the value is 1), so the corrupted value also
serialises back out under its original, wildly different spelling.

## Exact upstream location

`pkg/api/resource/suffix.go`, `suffixHandler.interpret`:

```go
parsed, err := strconv.ParseInt(string(suffix[1:]), 10, 64)
if err != nil {
    return 0, 0, DecimalExponent, false
}
// Stored objects keep the spelling they were given, so narrowing
// rather than rejecting is what keeps them decodable.
return 10, int32(parsed), DecimalExponent, true
```

`int32(parsed)` wraps modulo 2^32 for any exponent outside
`[math.MinInt32, math.MaxInt32]`. Exponents beyond int64 are fine
(`ParseInt` errors → rejected); the hole is exactly the int64-but-not-
int32 band, in both directions. Only the `e`/`E` decimal-exponent suffix
form is affected; decimal/binary SI suffixes come from fixed tables.

## Reachability

`ParseQuantity` is the single parser for `resource.Quantity`, which is
how every Kubernetes manifest expresses resource requests and limits
(`resources.requests.cpu`, `resources.limits.memory`, storage sizes,
etc.). The same parser backs `Quantity.UnmarshalJSON`,
`Quantity.UnmarshalCBOR` and the protobuf `Quantity.Unmarshal`
(`quantity_proto.go`), so the wrapped decode applies equally to API
payloads and to stored objects — the exact population the narrowing
comment was trying to keep decodable. Any client, admission path or
tool that feeds an attacker- or accident-supplied quantity string with
a crafted exponent gets a silently wrong resource value (in either
direction: `1e4294967295` shrinks an astronomical request to 0.1;
`1e-4294967296` inflates an effectively-zero one to 1).

## Violated invariant

`ParseQuantity`'s contract is error-or-faithful-value: it rejects
malformed suffixes (`ErrSuffix`) and already rejects the one
unrepresentable exponent it notices (`math.MinInt32`, via
`canInfScale`). A quantity string must never parse, without error, to a
value different from the one it denotes — the type doc's canonical-form
guarantee ("No precision is lost") presumes it. Wrapping is not a
faithful narrowing: it changes the exponent's magnitude *and can flip
its sign*, unlike the documented, deliberate approximations elsewhere
in the parser (BinarySI cap at 2^63−1, 29 significant digits, nano
floor), each of which is documented in the type doc and preserves the
value's rough magnitude and sign. The narrowing comment justifies not
*rejecting* stored spellings; it cannot justify returning a different
number — a saturating narrow (or a range check returning
`ok == false`) keeps stored objects decodable without corruption.

## Suggested fix direction (not submitted — lab is local-only)

In `interpret`, reject (return `ok == false`) when `parsed` does not
fit in an int32, or saturate to `math.MaxInt32` / `math.MinInt32+1`
before narrowing so downstream scale handling stays faithful in sign
and magnitude; `math.MinInt32` itself already errors in
`ParseQuantity`.

## Evidence

- `../results/run-100-checks.log` and `../results/run-2000-checks.log`:
  `TestPBTQuantityExponentRange` fails after 0 checks with the shrunk
  input `1e4294967296` in both runs; the other 16 properties pass at
  both check counts.
- Standalone probe (pre-harness) reproduced all table rows above plus
  the `String()` echo symptom before the property was written.

---

## TRIAGE CORRECTION (main agent, 2026-11-07) — NOT A BUG: deliberate compatibility contract

Verification overturned the CONFIRMED verdict above. The wraparound is upstream's
explicit, tested design:

- `ec9ee680` "Reject decimal exponents that do not fit the int32 scale" — upstream
  first implemented exactly the rejection this finding proposes.
- `98fee61a` "resource: keep decoding out-of-int32 exponents written before 1.38" —
  then deliberately restored wrap-on-decode: apiservers <=1.37 could write such
  spellings into etcd, and rejecting them would make stored objects undecodable
  after upgrade. The wrapped value and the original spelling are both pinned by
  `TestQuantityOutOfInt32ExponentCompatibility` (quantity_compat_test.go),
  `TestQuantityParseErrorBaseline/exponent-over-int32` (quantity_overflow_test.go)
  and `TestInterpretExponentInt32Bounds` (quantity_test.go).

A trial fix (range-check in `suffixHandler.interpret`) was implemented locally and
reverted in full: it passes the reproducer but fails those three upstream tests.
The property "out-of-range exponents must error" is the wrong property for the
decode path. Classification: DOCUMENTED BEHAVIOUR, do not file.
