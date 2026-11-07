# Results — pytorch/pytorch

Runs executed 2026-10-31 with `../run.sh`, against torch 2.15.0+cpu,
Hypothesis 6.170.14, NumPy 2.5.4, pytest 9.2.0, in the lab venv.

- `run-default.log` — Hypothesis profile `default`, 200 examples per
  property: **26 passed, 1 failed** (the deliberate B4 probe,
  Finding 001).
- `run-long.log` — Hypothesis profile `long`, 2,000 examples per
  property: **26 passed, 1 failed** (same probe).

Earlier exploratory runs during harness bring-up (not archived as logs)
produced the counterexamples behind Findings 002 and 004, and the first
long run — before Finding 005's precondition was added — additionally
failed B4b on an overflowing input; that failure is transcribed in full
in `../findings/005-b4b-error-bound-needs-no-overflow-precondition.md`,
and the final `run-long.log` above is the post-fix run.

Hypothesis runs with `database=None` (see `harness/conftest.py`), so
runs are hermetic: every counterexample relied on in `findings/` is
transcribed there verbatim and also lives in `reproducers.py.txt`.
