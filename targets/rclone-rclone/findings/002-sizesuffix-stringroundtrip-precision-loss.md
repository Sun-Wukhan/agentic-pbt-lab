# Finding: `SizeSuffix.stringRoundTrip()` does not round-trip — `%.3f` display precision loses bytes, up to ~0.3% at large scales

- Target / property: rclone/rclone @ `1053463755dea408a4abd42bf2ce917a3884a62a` — property F5 (`TestPBTSizeSuffixRoundTripString`): `Set(x.stringRoundTrip()) == x` for all `x ≥ 0`, per the function's own doc comment ("turns SizeSuffix into a string which Set parses back to the same value").
- Minimal counterexample (shrunk by rapid; smallest failing value in an ordered scan of `[0, 200000)`):
  - `SizeSuffix(1046)` → `"1.021Ki"` → `Set` → `1045` (off by −1)
  - At scale (plain-Go repro, `results/repro-counterexamples.log`): `2^60 + 2^40` → `"1.000Ei"` → delta **−1,099,511,627,776** bytes (≈ −1 TiB); `123456789012345678` → `"109.652Pi"` → delta **+387,572,761,730** bytes (≈ +0.31%)
  - Failure rate in `[0, 200000)`: 98,706 of 200,007 values (≈49%) fail exact round-trip via `stringRoundTrip` (and 99,729 via plain `String()`).
- Observed vs expected: `String()` formats the scaled value with `%.3f`, so one display step is 0.1% of the unit — up to ~1 byte at Ki scale and ~10^15 bytes at Ei scale. The doc comment promises exact recovery, and two callers rely on the string form: `fs/registry.go` (flag default rendering, line 308) and `fs/bwtimetable.go` (bandwidth timetable serialisation, lines 21–24), so a serialised bandwidth limit can drift by up to ~0.3% when read back.
- Verdict: **confirmed bug — reportable; fix in progress by the main agent (separate step).** Unclaimed upstream as of the 2026-10-28 claim check.
- Reasoning: not a wrong property — the invariant is the function's documented contract, not our invention. The defect is precision, not parsing: `Set` is exact for the string it is given; the string itself cannot represent the value.
- Upstream issue: not yet filed (main agent handles filing)
- Fix PR: in progress separately — not part of this lab step.
