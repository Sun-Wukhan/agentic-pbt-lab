# Finding: naive `String()` → `Set()` identity fails at value 1 for both suffix types — display form is not a parse form

- Target / property: rclone/rclone @ `1053463755dea408a4abd42bf2ce917a3884a62a` — properties F6 (`TestPBTSizeSuffixStringSetIdentity`) and F7 (`TestPBTCountSuffixStringSetIdentity`): `Set(x.String()) == x` for all `x ≥ 0`.
- Minimal counterexample (shrunk by rapid):
  - `SizeSuffix(1)` → `String()` = `"1"` → `Set` → `1024` (a bare number is read as KiB by design)
  - `CountSuffix(1)` → `String()` = `"1"` → `Set` → `1000` (a bare number is read as kilo by design)
- Observed vs expected: below the first unit threshold, `String()` emits a bare number while `Set` interprets a bare number as scaled. For `SizeSuffix` this is exactly why `stringRoundTrip()` exists (it appends `B`); `CountSuffix` has no equivalent helper. Above the threshold the same `%.3f` precision issue as Finding 002 applies, and for `CountSuffix` the below-1000 corruption family is claimed upstream (see below).
- Verdict: **wrong property** — as stated. `String()` is a human display format; neither type documents it as parseable back. Recorded deliberately: the failure map is what exposed the real, documented-contract violation in Finding 002.
- Claim status: the `CountSuffix` below-1k round-trip corruption (F7 family) is claimed by open PR rclone/rclone#9981 (by n0liu) — **not reportable by us** regardless of verdict framing.
- Reasoning: design intent is visible in the code — `Set`'s bare-number scaling is long-standing documented flag behaviour, and `stringRoundTrip`'s existence (with its `B`-suffix comment: "A bare number is read by Set as KiB, so values below 1 KiB get a B suffix") shows upstream already knows display ≠ parse form.
- Upstream issue: n/a for F6; F7 family covered by https://github.com/rclone/rclone/pull/9981
- Fix PR: not ours — do not file.
