# Finding: `TestPBTFilterAddNeverPanics` initial failure was a harness artifact, not an rclone bug

- Target / property: rclone/rclone @ `1053463755dea408a4abd42bf2ce917a3884a62a` — property L3 (`TestPBTFilterAddNeverPanics`): `Filter.Add`/`AddRule` must not panic on arbitrary patterns.
- Minimal counterexample: none against rclone. The test failed before reaching `Add`: `NewFilter(&Options{})` returns `filter: --min-age "0s" can't be larger than --max-age "0s"`, because a zero-value `Options` is not the default configuration.
- Observed vs expected: upstream's own `TestNewFilterDefault` uses `NewFilter(nil)`, which selects the package defaults. The harness was corrected to `NewFilter(nil)` (identical otherwise); the property then passed 100/100 default checks and 2000/2000 long-run checks.
- Verdict: **wrong property / harness setup** — our test scaffolding, not upstream code.
- Reasoning: kept in the record per lab policy (false positives are published too): a "failure" whose counterexample is in the harness must be labelled as such, not silently fixed and forgotten. Original failure log: `results/run-default-fs.log`; fixed run: `results/run-l3-fixed.log`, `results/run-long-fspath-filter.log`.
- Upstream issue: none — nothing to report.
- Fix PR: none.
