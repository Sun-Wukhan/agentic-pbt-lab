# Finding: `SizeSuffix.Set` / `CountSuffix.Set` silently accept NaN and overflowing values, producing `math.MinInt64`

- Target / property: rclone/rclone @ `1053463755dea408a4abd42bf2ce917a3884a62a` — properties F3 (`TestPBTSizeSuffixSetAcceptedValuesSane`) and F4 (`TestPBTCountSuffixSetAcceptedValuesSane`): any input `Set` accepts must yield a value ≥ 0 or exactly −1 (`"off"`), because `Set` itself rejects negatives ("size can't be negative").
- Minimal counterexample (shrunk by rapid, reproduced with plain Go — see `reproducers_test.go.txt` and `results/repro-counterexamples.log`):
  - `SizeSuffix.Set("NaNK")` → `nil` error, value `-9223372036854775808`
  - `CountSuffix.Set("NaNK")` → `nil` error, value `-9223372036854775808`
  - `CountSuffix.Set("infK")`, `Set("1e18")`, `Set("1e19")`, `Set("9223372036854775807")` → all `nil` error, value `-9223372036854775808`
- Observed vs expected: `strconv.ParseFloat` accepts `NaN`/`+Inf`; NaN fails both the `value < 0` and (for SizeSuffix) the `value >= MaxInt64` comparisons, and float→int64 conversion of NaN/Inf/overflowing values yields `MinInt64` on amd64. `CountSuffix.Set` additionally has **no range check at all** (its sibling `SizeSuffix.Set` errors on every overflow case above). Because a negative `SizeSuffix` prints as `"off"`, `--max-size NaNK` is silently equivalent to no limit at all.
- Verdict: **confirmed bug — already claimed upstream (rclone/rclone#10154, by kevin9327, opened 21 Oct 2026), which fixes exactly this NaN/overflow family. NOT reportable by us.**
- Reasoning: root cause read directly from `fs/sizesuffix.go` `Set` (missing NaN guard) and `fs/countsuffix.go` `Set` (missing NaN guard *and* missing `>= MaxValue` check); claim verified by the main agent via GitHub REST on 2026-10-28.
- Related: the same `ParseFloat`-without-NaN-check pattern was found by a repo-wide audit in `fs/parseduration.go`, claimed the same day by rclone/rclone#10199 — evidence the property generalises beyond these two parsers.
- Upstream issue: covered by PR https://github.com/rclone/rclone/pull/10154 (and #10199 for parseduration)
- Fix PR: not ours — do not file.
