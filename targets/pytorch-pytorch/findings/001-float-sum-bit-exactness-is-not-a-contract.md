# Finding: `torch.sum` and `np.sum` disagree bit-for-bit on float32 — summation order is not a contract

- Target / property: pytorch/pytorch @ torch 2.15.0+cpu — property B4
  (`test_b4_float_sum_exact_agreement_probe`): `torch.sum(a) == np.sum(a)`
  bit-exactly for 1-D float32 arrays.
- Minimal counterexample (shrunk by Hypothesis):
  - input (float32): `[0.0, 0.0, 3.0, 67108864.0, 3.0]`
  - `torch.sum` → `67108872.0`; `np.sum` → `67108864.0`; the exactly
    rounded sum (math.fsum) is `67108870.0` — both implementations are
    wrong, in opposite directions.
  - A second, prettier witness: nine copies of `0.1f` give
    `0.9000000953674316` (torch) vs `0.9000000357627869` (numpy).
- Observed vs expected: at magnitude 2^26 the float32 ulp is 8, so adding
  a lone `3.0` rounds away entirely. NumPy's pairwise order adds the
  large value early and swallows both 3s; torch's blocked order sums the
  two 3s together first (3+3=6, which then rounds *up* to +8). Neither
  result is an implementation error: they are two legal evaluation
  orders of a non-associative operation.
- Verdict: **wrong property** — as designed. Neither torch.sum nor
  np.sum documents an evaluation order, so bit-exact agreement between
  two independent reduction implementations is not a contract anyone
  promised. The property was stated deliberately to map where the
  implementations diverge.
- Reasoning: the defensible contract is the numerical one, and it holds:
  property B4b checks that *each* library's sum stays within the
  standard (n−1)·u·Σ|x| error bound of the exactly-rounded sum, and —
  once given its no-overflow precondition (see Finding 005, where the
  long run caught the first version of B4b being sloppy at exactly that
  corner) — it passes at 200 and 2,000 checks. A failure of B4b within
  its precondition would have been a real bug; B4's failure is
  floating-point arithmetic being itself.
- Upstream issue: none — not a bug.
- Fix PR: none. Reproducer: `reproducers.py.txt`.
