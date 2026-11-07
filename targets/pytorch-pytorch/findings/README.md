# Findings index — pytorch/pytorch @ torch `2.15.0+cpu`

Runs: 2026-10-31, Hypothesis 6.170.14, NumPy 2.5.4. Profiles: default
200 examples per property, then long 2,000 examples per property for
every property (see `../results/`).
27 properties: 26 pass at both check counts; the single failure is the
deliberate B4 probe. Five findings — and notably, **zero confirmed
bugs in torch**: every counterexample triaged to a wrong property, a
documented divergence, or a defect in our own harness. That is a
result, and it is recorded with the same prominence a bug would get.

| # | Finding | Properties | Verdict |
|---|---------|------------|---------|
| 001 | `torch.sum` vs `np.sum` disagree bit-for-bit on float32 (`[0,0,3,2^26,3]`: 67108872 vs 67108864, exact 67108870) — summation order is not a contract; the error-bound form B4b holds | B4 (probe), B4b | Wrong property (by design) |
| 002 | D4's reference table missed `bcomplex32`: `result_type(bfloat16, 1j)` is bfloat16's true complex counterpart — torch followed its documented rule more faithfully than our transcription | D4 | Harness artifact (our reference implementation) |
| 003 | Torch rejects *every* negative-step slice NumPy evaluates, incl. `t[::-1]` — deliberate, long-standing (negative strides unsupported; upstream feature request pytorch#59786; use `torch.flip`) | I3 → I3 + I3b | Wrong property / intended behaviour (documented divergence) |
| 004 | Two harness artifacts in the lab's broadcast strategies: pre-drawn broadcast target was wrong when both sides dropped a dim; 0-d arrays kept mutating (NumPy scalars, `ascontiguousarray` promoting 0-d to (1,)) | B1, B2, B3 | Harness artifacts (our strategies) |
| 005 | B4b's error bound was vacuous once the true sum overflows float32 (`[8.507059e37]×4` → `inf` is IEEE-correct); the property was missing its no-overflow precondition | B4b | Wrong property (missing precondition; lab-side fix) |

Passed clean (200 and 2,000 checks each): B1, B2, B3, B4b (after
Finding 005's precondition), B5, D1, D2, D3, D4 (after Finding 002's
table fix), D5 — note D5 is worth singling out: `promote_types` *is*
associative on all 13 dtypes at 2,000 random triples, unlike NumPy's —
V1, V2, V3, V4, C1, C2, C3, I1, I2, I3 (positive steps), I3b
(characterisation), S1, S2, O1, O2, O3.

Reproduction code for Findings 001–003: `reproducers.py.txt`
(plain torch/numpy, no Hypothesis). Full logs: `../results/`.
