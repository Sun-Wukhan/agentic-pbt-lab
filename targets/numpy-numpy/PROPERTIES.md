# Properties — numpy/numpy

Target version: **numpy 2.5.4** (PyPI wheel, installed 2026-10-31) ·
Python 3.12.3 · Hypothesis 6.170.14 · pytest 9.2.0
Framework: Python, [Hypothesis](https://hypothesis.readthedocs.io/)
(+ `hypothesis.extra.numpy`)
Harness layout: plain pytest files under `harness/` run against the
installed numpy — there is no upstream tree to patch, so `run.sh` just
builds the pinned virtualenv (shared at `~/workspace/.venv-pbt`) and runs
the suite. NumPy is exercised as a binary dependency, exactly the way the
Maaz et al. paper exercised Python packages.

| # | Property (invariant) | Source (file/doc it was inferred from) | Test |
|---|----------------------|------------------------------------------|------|
| P1 | `promote_types(a, b) == promote_types(b, a)` | `np.promote_types` docstring: "This function is symmetric" | `test_p1_promote_types_commutative` |
| P2 | `promote_types(a, a) == a` | `np.promote_types` docstring: returns the smallest type both inputs safely cast to; a type is its own bound | `test_p2_promote_types_idempotent` |
| P3 | `promote_types(promote_types(a, b), c) == promote_types(a, promote_types(b, c))` | The lattice property one would want; the same docstring warns the function is "rarely associative" — stated deliberately to map where it breaks | `test_p3_promote_types_associative` |
| P4 | `result_type(arrays of dtype a, b) == promote_types(a, b)` | NEP 50 promotion rules (as documented for `np.result_type`): array–array promotion is dtype-only, no value-based casting | `test_p4_result_type_agrees_with_promote_types` |
| B1 | `broadcast_shapes(s1, s2)` succeeds exactly when `np.broadcast` of two arrays with those shapes succeeds, with the same shape | `np.broadcast_shapes` docstring + Broadcasting rules in the user guide | `test_b1_broadcast_shapes_agrees_with_broadcast` |
| B2 | `broadcast_shapes` is commutative, including agreement on failure | Broadcasting rules (dimension comparison is order-free) | `test_b2_broadcast_shapes_commutative` |
| B3 | `broadcast_shapes` is associative over triples, and the flat 3-way call agrees with both groupings | Broadcasting rules applied pairwise | `test_b3_broadcast_shapes_associative` |
| R1 | `a.reshape(a.size).reshape(a.shape)` restores shape, dtype and values | `ndarray.reshape` docs (new view with the same data) | `test_r1_reshape_flat_round_trip` |
| R2 | `a.transpose(p).transpose(argsort(p)) == a` for every axis permutation `p` | `ndarray.transpose` docs (permuted view; inverse permutation undoes it) | `test_r2_transpose_inverse_permutation` |
| R3 | `a.T.T == a` | `ndarray.T` docs | `test_r3_double_transpose` |
| R4 | `a.reshape(k, -1)` for a divisor `k` of `a.size` preserves all values and reshapes back exactly | `ndarray.reshape` docs (`-1` dimension is inferred) | `test_r4_reshape_with_inferred_dim` |
| S1 | `argsort(a)` is a permutation of `range(len(a))` | `np.argsort` docs (indices that would sort the array) | `test_s1_argsort_is_permutation` |
| S2 | `a[argsort(a)]` is non-decreasing and equals `np.sort(a)` | `np.argsort` / `np.sort` docs | `test_s2_argsort_sorts_and_matches_sort` |
| S3 | `np.sort(a)` is non-decreasing and a permutation (same multiset) of `a` | `np.sort` docs | `test_s3_sort_is_permutation_of_input` |
| S4 | `np.partition(a, k)[k] == np.sort(a)[k]`, all elements before `k` are ≤ it, all after are ≥ it, and the result is a permutation of `a` | `np.partition` docs (verbatim contract) | `test_s4_partition_split_property` |
| S5 | Stable argsort keeps equal elements in original index order | `np.argsort` docs, `kind='stable'` | `test_s5_stable_argsort_tie_order` |
| Z1 | `np.frombuffer(a.tobytes(), a.dtype).reshape(a.shape) == a`, dtype (incl. byte order) preserved | `ndarray.tobytes` / `np.frombuffer` docs | `test_z1_tobytes_frombuffer_round_trip` |
| Z2 | `np.save` → `np.load` preserves dtype, shape, values, and the Fortran-order flag | `np.save` / `np.load` docs; .npy format spec stores dtype, shape and `fortran_order` | `test_z2_save_load_round_trip` |
| Z3 | `a.tobytes(order='F')` read back with `reshape(shape, order='F')` restores `a` | `ndarray.tobytes(order=...)` docs | `test_z3_tobytes_fortran_order_round_trip` |
| A1 | `(a + b) - b == a` and `(a - b) + b == a` for integer dtypes | Integer arithmetic is modulo-2ⁿ two's-complement (overflow wraps), so ring identities hold exactly | `test_a1_add_sub_inverse` |
| A2 | `a * (b + c) == a*b + a*c` for integer dtypes | Same modular-ring reasoning as A1 | `test_a2_distributivity` |
| A3 | For `b != 0`: `a == b * (a // b) + (a % b)`, and `np.divmod` agrees with the separate operators | `np.floor_divide` / `np.remainder` / `np.divmod` docs | `test_a3_divmod_identity` |
| A4 | `//` and `%` match Python's floor semantics elementwise; `|r| < |b|` and `r` has the divisor's sign or is zero (the single overflowing pair `INT_MIN, -1` excluded: its quotient is unrepresentable and wraps by design) | `np.remainder` docs ("the result has the same sign as the divisor") + Python semantics | `test_a4_remainder_matches_python_model` |
| A5 | `np.clip(a, lo, hi)` lies within `[lo, hi]`, is idempotent, and leaves in-bounds elements untouched | `np.clip` docs | `test_a5_clip_bounds_and_idempotence` |
| A6 | `np.where(c, x, y)` equals the elementwise model and `np.select([c], [x], default=y)` | `np.where` / `np.select` docs | `test_a6_where_select_consistency` |

## Notes

- Promotion dtype set excludes `longdouble`/`complex256`: their width is
  platform-dependent, which would make the runs non-reproducible across
  machines. Everything else (bool, all int/uint widths, float16/32/64,
  complex64/128) is in.
- Sort properties run on integer arrays and finite float64 arrays (NaN is
  excluded by strategy: NaN's unordered semantics are a separate,
  documented contract and would only re-derive them).
- A3 deliberately includes the `INT_MIN // -1` corner: the quotient wraps
  to `INT_MIN` with remainder 0, and the identity `b*q + r == a` still
  holds modulo 2ⁿ (verified). A4 excludes that one pair because it compares
  against Python's unbounded integers, where the quotient is `2ⁿ⁻¹` and
  cannot be represented in the dtype at all.
- Serialisation properties build arrays by `astype` from int64/float64
  material into every fixed-width dtype incl. both byte orders; values
  that overflow or wrap during the cast are fine — the property is that
  whatever array results then round-trips byte-exactly.
