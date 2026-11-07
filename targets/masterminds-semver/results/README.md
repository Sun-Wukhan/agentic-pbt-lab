# Results — Masterminds/semver

All runs: 2026-11-07, Go 1.25.1, rapid v1.2.0, pinned commit
`dd2b995c61c39ddd668b23ac919b04d607be35ab`, harness module in
`../harness/` (see `../run.sh`).

- `run-100-checks.log` — final harness, `-rapid.checks=100`:
  22/22 properties pass.
- `run-2000-checks.log` — final harness, `-rapid.checks=2000`:
  22/22 properties pass (2,000 checks per property).
- `run-2000-checks-v8-failure.log` — the V8-only run (2,000 checks)
  that caught findings/001, before the confirmed-defect pattern was
  quarantined in the V8/C3/C8 generators: rapid shrank to
  `Compare("0.0.0-100000000000000000000.a.0+a", "0.0.0-20000000000000000000") = -1`
  against the spec model's `+1`.
- Baseline: the target's own test suite (`go test ./...` in a scratch
  copy of the clone) passes at the pinned commit.
