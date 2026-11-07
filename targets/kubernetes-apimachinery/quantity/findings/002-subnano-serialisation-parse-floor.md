# Finding 002 — Sub-nano constructed quantities do not survive serialisation round-trips — documented parse floor, NOT a bug

- Target / property: kubernetes/apimachinery, `pkg/api/resource` at the
  wave-1 pin — wave-2 round-trip properties R1/R3/R4/R9/R10/R14/R16 as
  first stated (unconditional `String()`/JSON/proto/CBOR → parse
  preserves value).
- Minimal counterexample (deterministic, found in pre-harness probes):
  `NewScaledQuantity(12345, -30)` — the exact value 1.2345e-26 —
  serialises to `"12345e-30"`, which re-parses to `1n` (1e-9), a value
  ~17 orders of magnitude larger. Same shape at scales −100 and
  −2147483646.
- Observed vs expected: the re-parse rounds the sub-nano value *away
  from zero* to the minimum representable parsed value, 1n.
- Verdict: **wrong property — intended, documented behaviour.**
- Reasoning: the type doc in `quantity.go` states it outright: parsing
  "preserves a value only down to nano, rounding a finer non-zero value
  away from zero, so `0.9n` becomes `1n`". Constructed quantities (via
  `NewScaledQuantity`, `SetScaled`, `NewDecimalQuantity`, or Dec
  arithmetic) may legitimately hold finer precision in memory; the
  *parser* never produces it, and every serialisation decode path goes
  through the parser. The round-trip properties now carry the
  precondition `q.AsScale(Nano)` reports exact, under which all of them
  pass at 100 and 2,000 checks. Not reportable.
- Upstream issue: none. Fix PR: none.
