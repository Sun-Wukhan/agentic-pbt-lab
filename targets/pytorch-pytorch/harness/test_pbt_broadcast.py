# Properties B1-B5: broadcasting, values and shapes, torch vs NumPy.
#
# Sources: torch's broadcasting semantics doc ("the tensors are
# broadcastable if ... aligned to the right ... each dimension either
# equal or one of them is 1") explicitly follows NumPy; the NumPy
# broadcasting doc is the shared oracle. Elementwise + and * on identical
# inputs are single IEEE operations in both libraries, so exact value
# equality is the right oracle THERE. Reductions are different: neither
# library documents a summation order, which is why B4 is stated
# deliberately as a probe expected to triage as a wrong property.
import numpy as np
import torch
from hypothesis import given
import hypothesis.strategies as st
import hypothesis.extra.numpy as hnp

from pbt_strategies import (NP_FLOAT_DTYPES, NP_INT_DTYPES,
                            broadcastable_np_pairs, to_torch)


@given(pair=broadcastable_np_pairs(NP_INT_DTYPES))
def test_b1_int_broadcast_values_match_numpy(pair):
    """B1: broadcast int add/mul values equal NumPy's exactly (incl.
    wrap-around on overflow: both libraries wrap two's-complement)."""
    _target, a, b = pair
    ta, tb = to_torch(a), to_torch(b)
    assert torch.equal(ta + tb, to_torch(a + b))
    assert torch.equal(ta * tb, to_torch(a * b))


@given(pair=broadcastable_np_pairs(NP_FLOAT_DTYPES))
def test_b2_float_broadcast_values_match_numpy(pair):
    """B2: broadcast float add/mul values equal NumPy's exactly.
    Elementwise ops are one IEEE rounding each in both libraries, so
    no tolerance is needed or permitted."""
    _target, a, b = pair
    ta, tb = to_torch(a), to_torch(b)
    assert np.array_equal((ta + tb).numpy(), a + b)
    assert np.array_equal((ta * tb).numpy(), a * b)


@given(pair=broadcastable_np_pairs(NP_INT_DTYPES + NP_FLOAT_DTYPES))
def test_b3_broadcast_shape_agrees(pair):
    """B3: the broadcast result SHAPE agrees with NumPy's broadcast
    target shape (structural agreement, values irrelevant)."""
    target, a, b = pair
    ta, tb = to_torch(a), to_torch(b)
    assert tuple((ta + tb).shape) == target == np.broadcast_shapes(a.shape, b.shape)


@given(data=st.data())
def test_b5_broadcast_error_agreement(data):
    """B5: for arbitrary small shape pairs, torch raises exactly when
    NumPy raises (incompatible broadcasting is rejected by both)."""
    shape_a = data.draw(hnp.array_shapes(max_dims=3, max_side=4))
    shape_b = data.draw(hnp.array_shapes(max_dims=3, max_side=4))
    ea = np.empty(shape_a, dtype=np.int64)
    eb = np.empty(shape_b, dtype=np.int64)
    np_failed = False
    try:
        np_shape = np.broadcast_shapes(shape_a, shape_b)
    except ValueError:
        np_failed = True
    ta = torch.empty(shape_a, dtype=torch.int64)
    tb = torch.empty(shape_b, dtype=torch.int64)
    torch_failed = False
    try:
        t_shape = tuple((ta + tb).shape)
    except RuntimeError:
        torch_failed = True
    assert np_failed == torch_failed
    if not np_failed:
        assert t_shape == np_shape
    _ = (ea, eb)


@given(a=hnp.arrays(np.dtype("f4"), hnp.array_shapes(max_dims=1, max_side=12),
                    elements=st.floats(allow_nan=False, allow_infinity=False,
                                       width=32)))
def test_b4b_float_sum_within_error_bound(a):
    """B4b (the defensible form of B4): although torch's and NumPy's
    float32 sums need not be bit-equal, EACH stays within the standard
    summation error bound (n-1)*u*sum|x| of the exactly-rounded sum
    (math.fsum over the float64 images, which are exact). Any summation
    order satisfies this bound; violating it would be a real bug.

    Precondition (added after findings/005): the error-bound statement
    is vacuous once the true sum overflows float32 — inf is then the
    IEEE-correct result. We therefore require sum|x| itself to fit in
    float32, which also keeps every partial sum of every order in
    range."""
    import math

    from hypothesis import assume

    exact = math.fsum(a.astype(np.float64).tolist())
    assume(float(np.sum(np.abs(a.astype(np.float64))))
           <= float(np.finfo(np.float32).max))
    u = 2.0 ** -24  # float32 unit roundoff
    bound = max(a.size - 1, 0) * u * float(np.sum(np.abs(a.astype(np.float64))))
    # slack for float64 evaluation of the bound itself
    bound = bound * (1 + 1e-9) + 1e-30
    assert abs(to_torch(a).sum().item() - exact) <= bound
    assert abs(np.sum(a).item() - exact) <= bound


@given(a=hnp.arrays(np.dtype("f4"), hnp.array_shapes(max_dims=1, max_side=12),
                    elements=st.floats(allow_nan=False, allow_infinity=False,
                                       width=32)))
def test_b4_float_sum_exact_agreement_probe(a):
    """B4 (deliberate probe): torch.sum == np.sum bit-exactly on float32.

    Neither library documents a reduction order, and pairwise/vectorised
    summation makes float addition non-associative, so exact agreement is
    NOT a contract. Stated and run to map where the two implementations
    actually diverge; a failure triages as wrong property by design.
    """
    t = to_torch(a)
    assert t.sum().item() == np.sum(a).item()
