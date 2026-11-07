# Finding: B4b's error bound was vacuous at overflow — the property was missing its no-overflow precondition (torch's `inf` was IEEE-correct)

- Target / property: pytorch/pytorch @ torch 2.15.0+cpu — property B4b
  (`test_b4b_float_sum_within_error_bound`), found only in the long
  (2,000-example) run; the 200-example run had not generated values this
  large.
- Minimal counterexample (shrunk by Hypothesis):
  - input (float32): `[8.507059e37, 8.507059e37, 8.507059e37, 8.507059e37]`
  - exactly-rounded sum: `3.402823669209385e38` — just above float32 max
    (`3.4028234663852886e38`)
  - `torch.sum` → `inf`; our bound said the result had to sit within
    ~6.1e31 of the exact sum.
- Observed vs expected: once the true sum leaves the float32 range,
  `inf` *is* the correctly rounded float32 answer, and an absolute
  error bound against a real-number exact sum stops meaning anything —
  the standard summation error analysis (n−1)·u·Σ|x| presumes no
  overflow. NumPy's sum of the same input overflows identically.
- Verdict: **wrong property** — a missing precondition in our
  supposedly-defensible form of B4, not a torch defect. (An instructive
  one: B4 was the naive probe, B4b was the "correct" version, and B4b
  still had a hole — at exactly the extreme-value corner a PBT run
  exists to visit.)
- Fix: B4b now assumes Σ|x| fits in float32 before asserting the bound.
  That precondition also keeps every partial sum of every evaluation
  order in range, so the bound again applies to both libraries
  unconditionally. B4b then passes at 200 and 2,000 checks.
- Upstream issue: none. Fix PR: none (lab-side fix). Reproducer: the
  input above; `torch.tensor([8.507059e37]*4).sum()` → `tensor(inf)`,
  correctly.
