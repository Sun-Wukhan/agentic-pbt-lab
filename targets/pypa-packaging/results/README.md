# Run record — pypa/packaging @ 713d0d78 (26.4.dev0)

Date: 2026-11-07 · Framework: Hypothesis 6.170.15 · pytest 9.2.0 ·
Python 3.12.3 · packaging built from the shallow clone in `../upstream`
(commit 713d0d78924ab2686c513dde058ce6aafc63e766, HEAD of `main` that
day) and installed into the target-local virtualenv `../.venv`.
Environment: runs executed sequentially (one pytest process at a
time). Version/specifier structures are small; wall-clock is
Hypothesis generation and marker evaluation.

## Logs

| File | Contents |
|------|----------|
| `run-default.log` | Full suite, default profile (500 examples/property): 43 passed, 9 failed — the 9 failures are exactly the Finding 001 (6) and Finding 002 (3) tests, which encode the spec-required behaviour and fail by design. |
| `run-long.log` | Full suite, long profile (2,000 examples/property): identical tally — 43 passed, 9 failed (the same finding tests), 354.96s. |
| `repro-counterexamples.log` | Plain-Python reproducers for Findings 001 and 002 (source: `../findings/reproducers.py`). |

## Notes

- R3/R4 failed in the very first default-profile run with the shrunk
  counterexample `'0===0,>= 0'` — that failure *was* Finding 001.
  After triage, the requirement strategy was scoped (=== only in last
  position; see `../PROPERTIES.md`) and both profiles above are the
  scoped runs. The unscored first run's tally was 41 passed / 2 failed
  (R3, R4) before the finding tests existed.
- Finding 002 came from a manual probe of the new validators while
  the V8 strategies were being written, not from a fuzz failure; it is
  recorded as such in its finding file.
- A scratch-venv check during triage confirmed Finding 001 also
  reproduces on packaging 25.0 (long-standing, not a 26.x regression).
