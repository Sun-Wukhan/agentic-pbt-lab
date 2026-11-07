# Target: numpy/numpy

Property-based testing of **numpy 2.5.4** (PyPI wheel) with Hypothesis —
Wave 3 of the Agentic PBT Lab (the Python-ecosystem wave, following the
Maaz et al. method). NumPy is exercised as an installed binary package,
the same way the paper exercised Python packages: no upstream tree, no
patches, just properties inferred from the documentation and checked at
scale.

## Layout

- `PROPERTIES.md` — the 25 inferred invariants, each with the doc it came
  from and the test that encodes it.
- `harness/` — the pytest suite (`test_pbt_*.py`), shared strategies
  (`pbt_strategies.py`) and Hypothesis profiles (`conftest.py`:
  `default` = 200 examples/property, `long` = 2,000).
- `findings/` — one file per counterexample with triage verdict, plus
  `reproducers.py` (plain-Python, no Hypothesis) and an index README.
- `results/` — run logs and the run record.
- `run.sh` — the whole recipe.

## Running

```sh
./run.sh          # default profile, 200 examples per property
./run.sh 2000     # long profile, 2,000 examples per property
```

The virtualenv is shared by the lab's Python targets at
`~/workspace/.venv-pbt` (override with `PBT_VENV`); `run.sh` creates it
if missing and installs the pinned versions: numpy 2.5.4,
hypothesis 6.170.14, pytest 9.2.0.

## Outcome (2026-10-31)

25 properties, 24 pass at 2,000 examples. The one failure is property
P3 (promotion associativity), triaged as a **wrong property**: NumPy
documents `promote_types` as "symmetric, but rarely associative", and the
counterexample `(int8, uint8, float16)` is precisely that documented
behaviour. Zero confirmed bugs; nothing filed upstream. See
`findings/README.md`.
