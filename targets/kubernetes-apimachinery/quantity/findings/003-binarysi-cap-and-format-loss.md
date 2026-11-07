# Finding 003 — BinarySI parse cap and format loss break naive serialisation round-trips — documented behaviour, NOT a bug

- Target / property: kubernetes/apimachinery, `pkg/api/resource` at the
  wave-1 pin — wave-2 properties R1/R3/R4/R9/R10 as first stated.
- Minimal counterexamples (shrunk by rapid at 100 checks):
  - `9765625000000000Ki` (constructed, ≈1.0e25) proto/CBOR/JSON
    round-trips to `9223372036854775807` (2^63−1).
  - `9224P` (≈9.224e18) re-rendered under `Format = BinarySI` as
    `9007812500000000Ki` re-parses to `9223372036854775807`.
  - BinarySI value `10000` canonicalises to `"10000"` (base-1024
    exponent 0 has no suffix), re-parses as DecimalSI, and
    re-canonicalises to `"10k"` — byte fixed point fails, value intact.
- Observed vs expected: re-parsing any BinarySI spelling caps the
  magnitude at 2^63−1, and a BinarySI quantity whose canonical form has
  no binary suffix loses its format on re-parse.
- Verdict: **wrong property — intended, documented behaviour.**
- Reasoning: the `quantity.go` type doc discloses the cap verbatim:
  "Parsing does cap a binarySI quantity, silently, at 2^63-1: `8Ei`
  parses as 2^63-1." Wave 1's Q2/Q3 never tripped it because parsed
  quantities are born capped; constructed quantities can exceed it, so
  the cap becomes a *serialisation* asymmetry. The format-loss case is
  the same family: `suffix.go` deliberately registers `""` for
  bePair{2, 0} ("Don't emit an error when trying to produce a suffix
  for 2^0"), and `CanonicalizeBytes` already documents emitting some
  BinarySI values "as if q.Format were DecimalSI". The properties now
  carry both preconditions (not-above-cap for BinarySI round-trips;
  byte fixed point only when the format survives) and pass at 100 and
  2,000 checks. Not reportable.
- Upstream issue: none. Fix PR: none.
