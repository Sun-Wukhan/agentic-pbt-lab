# Target: pypa/packaging

Property-based testing of **pypa/packaging** (the library under pip)
with Hypothesis — Wave 3 of the Agentic PBT Lab (the Python-ecosystem
wave, following the Maaz et al. method). The code under test is a
shallow clone of upstream `main` pinned at
**713d0d78924ab2686c513dde058ce6aafc63e766** (packaging 26.4.dev0),
installed into this target's virtualenv; nothing is patched and no
upstream contact was made (local-only run by instruction).

The 26.x line made this target timely: version ordering is a new
hand-rolled comparison key, specifier matching is a new interval-range
engine with a fast path that must agree with it, and 26.3 added a
public range set-algebra (`VersionRange`) plus `is_subset` /
`is_disjoint` relations — so most properties below are
cross-implementation agreements, the class of invariant a rewrite can
break while each half still looks right in isolation.

## Layout

- `PROPERTIES.md` — the 36 inferred invariants, each with the code/doc
  it came from and the test that encodes it.
- `harness/` — the pytest suite (`test_pbt_*.py`), shared structured
  strategies (`pbt_strategies.py`), Hypothesis profiles
  (`conftest.py`: `default` = 500 examples/property, `long` = 2,000),
  and the two finding test files (`test_finding_001.py`,
  `test_finding_002.py`) that encode the behaviour the specs require
  and currently fail.
- `findings/` — one file per counterexample with triage verdict, plus
  `reproducers.py` (plain-Python, no Hypothesis) and an index README.
- `results/` — run logs and the run record.
- `upstream/` — the shallow clone under test.
- `run.sh` — the whole recipe.

## Running

```sh
./run.sh        # default profile, 500 examples per property
./run.sh 2000   # long profile, 2,000 examples per property
```

Expected: all `test_pbt_*` properties pass; the 9 finding tests fail
by design (Findings 001 and 002).
