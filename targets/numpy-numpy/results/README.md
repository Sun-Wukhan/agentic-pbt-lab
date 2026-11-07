# Run record — numpy 2.5.4

Date: 2026-10-31 · Framework: Hypothesis 6.170.14 · pytest 9.2.0 ·
Python 3.12.3 · numpy 2.5.4 (PyPI wheel)
Environment: shared lab virtualenv `~/workspace/.venv-pbt`; runs executed
sequentially (one pytest process at a time). Arrays are tiny (≤ 5×5,
length ≤ 12 for 1-D properties), so wall-clock is dominated by
Hypothesis generation, not numpy.

## Logs

| File | Contents |
|------|----------|
| `run-default.log` | Full suite, default profile (200 examples/property), second seed: 24 passed, P3 FAILED with the shrunk `(int8, uint8, float16)` counterexample (Finding 001). |
| `run-long.log` | Full suite, long profile (2,000 examples/property): 24 passed, P3 FAILED with the same shrunk counterexample. |
| `repro-counterexamples.log` | Plain-Python triage reproducer for Finding 001 (source: `../findings/reproducers.py`): minimal triple with pairwise steps, plus exhaustive counts — 28 of 2,744 ordered triples non-associative, 0 of 196 ordered pairs non-commutative. |

## Seed note

The very first default-profile run (200 examples) passed all 25
properties, P3 included: the non-associative triples are ~1% of the
2,744-triple space, so a 200-draw sample misses them ~1 time in 8.
A second default-seed run and the 2,000-example run both caught P3 with
the identical shrunk triple. The logs above are from the catching runs;
the passing first run is recorded here for honesty (its tally line was
`25 passed in 13.18s`).
