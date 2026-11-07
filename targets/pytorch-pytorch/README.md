# Lab target: pytorch/pytorch

Property-based testing of PyTorch's Python-visible tensor semantics with
Hypothesis, following the lab method (see the repo root README). NumPy is
the differential oracle wherever the two libraries' semantics are
documented to agree; elsewhere the oracle is PyTorch's own documented
rules (type promotion) or intrinsic invariants (inverses, round trips).

- **Target**: `torch 2.15.0+cpu`, pinned in `requirements.txt`, exercised
  as installed in the lab venv (`~/workspace/.venv-torch`; override with
  `PYTORCH_PBT_VENV`). Unlike the Go targets there is no clone step:
  PyTorch is tested exactly as users import it.
- **Properties**: 25 in `PROPERTIES.md` — broadcasting values/shapes,
  dtype promotion (incl. Python-scalar weak promotion), view/reshape,
  split/join inverses, indexing, serialisation round trips, sorting.
- **Run**: `./run.sh` (200 examples/property) or `./run.sh long`
  (2,000/property). All shapes are deliberately small (sides 0..6);
  the invariants are structural and the lab VM is memory-tight.
- **Findings**: `findings/` — every counterexample, triaged, including
  the wrong properties. Run logs: `results/`.
