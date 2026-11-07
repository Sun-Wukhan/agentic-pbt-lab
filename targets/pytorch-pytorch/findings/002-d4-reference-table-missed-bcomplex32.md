# Finding: D4's first failure was in our reference table — bfloat16's complex counterpart is `bcomplex32`, not `complex32`

- Target / property: pytorch/pytorch @ torch 2.15.0+cpu — property D4
  (`test_d4_result_type_with_python_scalars`): `result_type` with a
  Python scalar follows the documented weak-scalar rule, checked against
  a hand-written reference implementation of that rule.
- Minimal counterexample (shrunk by Hypothesis):
  - `result_type(torch.empty(0, dtype=torch.bfloat16), 1j)` →
    `torch.bcomplex32`
  - our reference table said `torch.complex32`.
- Observed vs expected: the documented rule is "a complex scalar meeting
  a float tensor yields the complex counterpart of that float dtype".
  For float16 that counterpart is complex32; our table *guessed* the
  same for bfloat16. Torch 2.15 in fact has a dedicated `bcomplex32`
  dtype and returns it — torch is following its own rule more faithfully
  than our transcription of it.
- Verdict: **harness artifact** — the bug was in the lab's reference
  implementation, not in torch. Fixed by correcting the table
  (`bfloat16 → bcomplex32`); D4 then passes at every check count,
  covering all 13 promotion dtypes × {int, float, complex} Python
  scalars, including the other two sharp edges the probe verified:
  `result_type(bool, 1) == int64` (bool sits *below* the integral
  category, so the scalar's default dtype wins) and
  `result_type(float16, 1j) == complex32`.
- Reasoning: recorded in the open, per lab practice, because it is the
  interesting kind of false positive: the differential oracle itself was
  the least faithful implementation in the room. The weak-scalar rule as
  documented holds everywhere we can test it.
- Upstream issue: none — not a torch bug.
- Fix PR: none (lab-side fix, in the harness commit). Reproducer:
  `reproducers.py.txt`.
