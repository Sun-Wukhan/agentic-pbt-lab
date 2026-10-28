# Findings index — rclone/rclone @ `1053463755dea408a4abd42bf2ce917a3884a62a`

Runs: 2026-10-28, rapid v1.2.0, default 100 checks per property, then
2,000 checks for every property that passed (see `../results/`).
17 properties: 12 passed clean (each at both 100 and 2,000 checks),
5 failed; the 5 failures triage into Findings 001–003 below, and
Finding 004 records a sixth initial failure that was in our harness.

| # | Finding | Properties | Verdict |
|---|---------|------------|---------|
| 001 | Suffix parsers silently accept NaN / overflowing values → `MinInt64` (`"--max-size NaNK"` ≡ off) | F3, F4 | Confirmed bug — **already claimed upstream** by PR #10154 (kevin9327). Related same-root-cause case in `fs/parseduration.go` claimed by PR #10199. Not reportable by us. |
| 002 | `SizeSuffix.stringRoundTrip()` loses precision (`%.3f`): 1046 → 1045; up to ~0.3% (≈10^11–10^12 bytes) at large scales | F5 | Confirmed bug — **reportable, unclaimed**; fix in progress by the main agent. |
| 003 | Naive `String()`→`Set()` identity fails at value 1 (bare numbers parse as Ki/kilo) | F6, F7 | Wrong property (display ≠ parse form, by design). F7's below-1k corruption family is also claimed by PR #9981 (n0liu) — not reportable by us. |
| 004 | `Filter.Add` property's first failure was in our harness (`NewFilter(&Options{})`), not rclone | L3 | Wrong property / harness setup — fixed to `NewFilter(nil)`; property passes 2,000 checks. |

Passed clean (2,000 checks each unless noted): F1, F2 (suffix `Set`
never panics), S1, S2 (`fspath.Parse` robustness + lossless split —
the invariants from the repo's own `fs/fspath/fuzz.go`), L1, L2
(glob conversion robustness; literal globs match themselves), L3
(after Finding 004's harness fix), C1, C2, C3 (crypt decrypt
robustness; encrypt→decrypt round trips for all modes/encodings;
obfuscated Unicode round trips), H1, H2 (chunker metadata decoding
robustness; marshal→unmarshal round trip).

Reproduction code for Findings 001–003: `reproducers_test.go.txt`
(plain Go, no rapid). Full logs: `../results/`.
