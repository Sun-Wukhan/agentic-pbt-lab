# Properties — pytorch/pytorch

Target version: `torch 2.15.0+cpu` (pinned; exercised as installed, via the
lab venv — see `requirements.txt` and `run.sh`)
Framework: Python, [Hypothesis](https://hypothesis.readthedocs.io) 6.170.14,
NumPy 2.5.4 as differential oracle where semantics are documented to agree.
Harness layout: `harness/` is a plain pytest suite; `run.sh` is the whole
recipe. All shapes are small (sides 0..6, ≤3 dims) — the invariants are
structural, and the lab VM is memory-tight.

| # | Property (invariant) | Source (file/doc it was inferred from) | Test |
|---|----------------------|------------------------------------------|------|
| B1 | Broadcast int add/mul values equal NumPy's exactly (incl. two's-complement wrap on overflow) | torch broadcasting doc (follows NumPy); NumPy broadcasting doc | `test_b1_int_broadcast_values_match_numpy` |
| B2 | Broadcast float add/mul values equal NumPy's exactly (single IEEE op each side; no tolerance) | Same as B1; IEEE-754 elementwise semantics | `test_b2_float_broadcast_values_match_numpy` |
| B3 | Broadcast result shape equals NumPy's broadcast target shape | Broadcasting docs (structural agreement, split from values deliberately) | `test_b3_broadcast_shape_agrees` |
| B4 | **Probe:** `torch.sum` == `np.sum` bit-exactly on float32 — stated knowing neither library documents a reduction order; float associativity makes this wrong-by-design if it fails | Absence of contract in torch.sum / np.sum docs (order unspecified) | `test_b4_float_sum_exact_agreement_probe` |
| B4b | Both libraries' float32 sums stay within the standard error bound (n−1)·u·Σ\|x\| of the exactly-rounded sum — the numerical contract that actually exists | Error analysis of floating-point summation (any order satisfies the bound) | `test_b4b_float_sum_within_error_bound` |
| B5 | Incompatible broadcast shapes are rejected by torch exactly when NumPy rejects them; compatible ones give equal shapes | Broadcasting docs | `test_b5_broadcast_error_agreement` |
| D1 | `promote_types` is commutative | Type Promotion doc (binary least-upper-bound) | `test_d1_promote_types_commutative` |
| D2 | `promote_types(a, a) == a` | Type Promotion doc | `test_d2_promote_types_idempotent` |
| D3 | `result_type` of two tensors equals `promote_types` of their dtypes, commutatively | Type Promotion doc (tensor/tensor promotion is dtype-only) | `test_d3_result_type_of_tensors_is_promote_types` |
| D4 | `result_type` with a Python scalar follows the documented weak-scalar rule (scalar category ≤ tensor category → tensor dtype; above → default dtype of the scalar's category; complex scalar + float tensor → complex counterpart of the float dtype, e.g. float16 → complex32) | Type Promotion doc, "wrapped numbers" / weak dtype rules | `test_d4_result_type_with_python_scalars` |
| D5 | **Probe:** `promote_types` is associative — stated knowing the doc never promises a lattice (NumPy's docstring warns "rarely associative"); wrong-by-design if it fails | Type Promotion doc (silence on associativity) | `test_d5_promote_types_associative_probe` |
| V1 | `reshape` to any same-count shape holds exactly NumPy's reshape values (C order) | torch.reshape doc; NumPy reshape | `test_v1_reshape_values_match_numpy` |
| V2 | `permute` by p then by p⁻¹ restores the tensor exactly | torch.permute doc | `test_v2_permute_inverse_round_trip` |
| V3 | `unsqueeze(d).squeeze(d)` is identity; `flatten()` equals NumPy `ravel` | torch.squeeze/unsqueeze/flatten docs | `test_v3_squeeze_unsqueeze_and_flatten` |
| V4 | `reshape` of a non-contiguous (transposed) tensor matches NumPy value-for-value (logical order, never raw storage) | torch.reshape doc ("returns a view if possible, else a copy" — values are layout-independent) | `test_v4_transposed_reshape_matches_numpy` |
| C1 | `cat(split(t, sizes, dim), dim) == t` for every composition of sizes | torch.split doc | `test_c1_cat_of_split_restores_tensor` |
| C2 | `chunk(t, k, dim)` returns ≤ k chunks, sizes sum to the dim length and differ by ≤ 1, and `cat` restores t | torch.chunk doc (may return fewer than k chunks) | `test_c2_chunk_partitions_and_cat_restores` |
| C3 | `stack`/`unbind` are exact inverses in both directions | torch.stack / torch.unbind docs | `test_c3_stack_unbind_inverses` |
| I1 | Boolean-mask selection has `mask.sum()` elements and equals NumPy's masked selection | torch indexing doc (mask indexing) | `test_i1_boolean_mask_selection` |
| I2 | `scatter` then `gather` at the same unique indices returns the scattered values exactly | torch.scatter / torch.gather docs | `test_i2_gather_scatter_inverse_on_unique_indices` |
| I3 | Basic slicing (start/stop/step, positive steps) along axis 0 matches NumPy exactly | torch indexing doc (basic slicing follows NumPy) | `test_i3_basic_slicing_matches_numpy` |
| I3b | Characterisation: negative-step slices are rejected by torch (`ValueError`) while NumPy evaluates them; `torch.flip` is the equivalent for `[::-1]`. Originally part of I3 as stated from the indexing doc; the blanket rejection is a deliberate, long-standing limitation (negative strides unsupported; upstream feature request pytorch#59786) — see findings/003 | Observed behaviour of the pinned build + upstream issue tracker | `test_i3b_negative_step_slicing_is_a_documented_divergence` |
| S1 | `save`/`load` round trip preserves values, dtype, shape, strides and contiguity — incl. transposed layouts — via the `weights_only=True` path | torch.save / torch.load docs | `test_s1_save_load_round_trip_exact` |
| S2 | 0-dim, empty and small tensors of every dtype round-trip exactly | Same as S1 | `test_s2_round_trip_all_dtypes_and_empty` |
| O1 | `torch.sort` values on 1-D arrays equal `np.sort` exactly | torch.sort doc; NumPy sort | `test_o1_sort_values_match_numpy` |
| O2 | `torch.sort(stable=True)` indices equal NumPy's stable argsort exactly | torch.sort doc (stable preserves equal-element order) | `test_o2_stable_sort_indices_match_numpy` |
| O3 | `topk(k)` returns the k largest values as a multiset, in descending order | torch.topk doc | `test_o3_topk_is_sorted_head` |

## Notes

- **Values vs shapes are kept separate on purpose** (B1/B2 vs B3): an
  elementwise op is a single IEEE rounding in both libraries, so exact
  value equality is a fair oracle there; that argument does not extend to
  reductions (B4) or to promotion lattices (D5), which are recorded as
  deliberate probes whose failures would be wrong properties, not bugs.
- NaN is excluded from the sort properties (O1–O3): NaN has no total
  order; both libraries place it last by convention, but index-level
  agreement among multiple NaNs is not a documented contract.
- Generated floats are finite everywhere; infinities would only arise
  from overflow inside the ops under test, where both libraries are IEEE
  and agree (inf == inf). Integer overflow wraps in both (NumPy warns;
  the warning is filtered in `conftest.py`, the values are compared).
- bfloat16 and complex32 appear only in the promotion properties, which
  build 0-element tensors: NumPy cannot mirror those dtypes, and none is
  needed — the oracle there is torch's own documented promotion rule.
