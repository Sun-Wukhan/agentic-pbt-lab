# Target: encode/httpx

Property-based testing of **httpx `main` @ `b5addb6`** (post-0.28.1
development tree) with Hypothesis — Wave 3 of the Agentic PBT Lab (the
Python-ecosystem wave, following the Maaz et al. method). The surfaces are
the pure model layer — `URL`, `QueryParams`, `Headers` — inferred from
`httpx/_urlparse.py`, `httpx/_urls.py`, `httpx/_models.py` and
`httpx/_utils.py`. Everything is hermetic: no transports, no network.

## Layout

- `PROPERTIES.md` — the 27 inferred invariants (U1–U11, Q1–Q8, H1–H8),
  each with the file/doc it came from and the test that encodes it.
- `httpx/` — the shallow upstream clone at the pinned commit (unmodified).
- `harness/` — the pytest suite (`test_pbt_*.py`), shared strategies
  (`pbt_strategies.py`) and Hypothesis profiles (`conftest.py`:
  `default` = 500 examples/property, `long` = 2,000).
- `findings/` — one file per counterexample with triage verdict, plus
  `reproducers.py` (plain Python, no Hypothesis) and an index README.
- `results/` — run logs.
- `run.sh` — the whole recipe.

## Running

```sh
./run.sh          # default profile, 500 examples per property
./run.sh 2000     # long profile, 2,000 examples per property
```

The virtualenv is shared by the lab's Python targets at
`~/workspace/.venv-pbt` (override with `PBT_VENV`); `run.sh` creates it if
needed and installs the clone into it.
