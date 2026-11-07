# Findings index — numpy 2.5.4

Runs: 2026-10-31, Hypothesis 6.170.14, 200 examples per property
(default profile), then 2,000 examples per property (long profile) —
see `../results/`.
25 properties: 24 passed clean at both 200 and 2,000 examples;
1 failed, triaged below.

| # | Finding | Properties | Verdict |
|---|---------|------------|---------|
| 001 | `promote_types` is not associative: `(int8, uint8, float16)` → `float32` vs `float16` by grouping; 28 of 2,744 triples affected | P3 | Wrong property — the docstring states the function is "symmetric, but rarely associative". Not a bug, not reportable. |

Passed clean (2,000 examples each): P1, P2, P4 (promotion commutativity,
idempotence, `result_type` agreement — the documented half of the
lattice), B1–B3 (`broadcast_shapes` agrees with `np.broadcast` on success
and failure, commutative, associative), R1–R4 (reshape/transpose round
trips), S1–S5 (argsort permutation, sort permutation + order, partition
split property verbatim from the docs, stable tie order), Z1–Z3
(`tobytes`/`frombuffer`, `save`/`load` incl. Fortran-order flag, F-order
bytes — all byte-exact for every fixed dtype incl. both byte orders),
A1–A6 (integer ring identities under wraparound, divmod identity
including the `INT_MIN // -1` corner, remainder matching Python floor
semantics, clip bounds/idempotence, `where`/`select` model agreement).

Run-to-run note: P3's failure is seed-dependent at 200 examples — the
non-associative triples are ~1% of the triple space, and one default-seed
run passed P3 by chance before a second seed and the 2,000-example run
both caught the same shrunk triple. Both default-profile outcomes are in
`../results/`.

Reproduction code for Finding 001: `reproducers.py` (plain Python, no
Hypothesis). Full logs: `../results/`.
