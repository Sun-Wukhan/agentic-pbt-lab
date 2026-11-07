# Properties R1-R4: reshape / transpose round trips. Sources:
# ndarray.reshape and ndarray.transpose documentation (a reshape with the
# same size and a transpose by the inverse permutation are exact inverses;
# no data is reinterpreted, only the view).
import numpy as np
from hypothesis import given, assume
import hypothesis.strategies as st
import hypothesis.extra.numpy as hnp


def _arrays():
    return hnp.arrays(
        np.dtype("i8"), hnp.array_shapes(min_dims=1, max_dims=4, max_side=4))


@given(a=_arrays())
def test_r1_reshape_flat_round_trip(a):
    """R1: reshape to (size,) and back restores shape, dtype and values."""
    back = a.reshape(a.size).reshape(a.shape)
    assert back.shape == a.shape
    assert back.dtype == a.dtype
    assert np.array_equal(back, a)


@given(a=_arrays(), data=st.data())
def test_r2_transpose_inverse_permutation(a, data):
    """R2: transpose by p then by argsort(p) is the identity."""
    p = data.draw(st.permutations(list(range(a.ndim))))
    moved = a.transpose(p)
    assert moved.shape == tuple(a.shape[i] for i in p)
    back = moved.transpose(np.argsort(p))
    assert back.shape == a.shape
    assert np.array_equal(back, a)


@given(a=_arrays())
def test_r3_double_transpose(a):
    """R3: a.T.T == a for any dimensionality."""
    assert np.array_equal(a.T.T, a)
    assert a.T.T.shape == a.shape


@given(a=_arrays(), data=st.data())
def test_r4_reshape_with_inferred_dim(a, data):
    """R4: reshape(k, -1) for a divisor k of size preserves all values and
    reshapes back exactly."""
    assume(a.size > 0)
    divisors = [d for d in range(1, a.size + 1) if a.size % d == 0]
    k = data.draw(st.sampled_from(divisors))
    wide = a.reshape(k, -1)
    assert wide.shape == (k, a.size // k)
    assert np.array_equal(wide.reshape(a.shape), a)
    assert np.array_equal(wide.reshape(-1), a.reshape(-1))
