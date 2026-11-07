# Properties A1-A6: integer arithmetic identities that must hold exactly.
# Sources: NumPy integer arithmetic is modulo 2**n two's-complement
# arithmetic (overflow wraps), so ring identities hold exactly even where
# values overflow; np.floor_divide / np.remainder document Python-style
# floor semantics (remainder has the sign of the divisor).
import numpy as np
from hypothesis import given, assume
import hypothesis.strategies as st
import hypothesis.extra.numpy as hnp

from pbt_strategies import int_array_pairs, int_arrays_1d


@given(pair=int_array_pairs())
def test_a1_add_sub_inverse(pair):
    """A1: (a + b) - b == a exactly for integer dtypes (modular ring)."""
    _, a, b = pair
    with np.errstate(over="ignore", invalid="ignore"):
        assert np.array_equal((a + b) - b, a)
        assert np.array_equal((a - b) + b, a)


@given(pair=int_array_pairs())
def test_a2_distributivity(pair):
    """A2: a * (b + c) == a*b + a*c exactly (ring identity under wraparound).
    c reuses b-shaped material drawn as a second pair elementwise."""
    _, a, b = pair
    c = np.flip(b)  # same dtype/shape, different arrangement
    with np.errstate(over="ignore", invalid="ignore"):
        assert np.array_equal(a * (b + c), a * b + a * c)


@given(pair=int_array_pairs())
def test_a3_divmod_identity(pair):
    """A3: for b != 0, a == b * (a // b) + (a % b) exactly, and np.divmod
    agrees with the separate operators. Holds even for INT_MIN // -1,
    where the quotient wraps (verified against numpy semantics)."""
    _, a, b = pair
    assume(np.all(b != 0))
    with np.errstate(over="ignore", invalid="ignore", divide="ignore"):
        q, r = np.divmod(a, b)
        assert np.array_equal(q, a // b)
        assert np.array_equal(r, a % b)
        assert np.array_equal(b * q + r, a)


@given(a=int_arrays_1d(), data=st.data())
def test_a4_remainder_matches_python_model(a, data):
    """A4: numpy floor_divide/remainder match Python's floor semantics
    elementwise (|r| < |b|, r has the divisor's sign or is zero), excluding
    the single overflowing pair (dtype MIN, -1) whose quotient cannot be
    represented and wraps by design."""
    dtype = a.dtype
    b = data.draw(hnp.arrays(dtype, a.shape))
    assume(np.all(b != 0))
    info = np.iinfo(dtype)
    xs, ys = a.tolist(), b.tolist()
    pairs = [(x, y) for x, y in zip(xs, ys)
             if not (x == info.min and y == -1)]
    with np.errstate(over="ignore", invalid="ignore", divide="ignore"):
        qs, rs = (a // b).tolist(), (a % b).tolist()
    got = [(q, r) for (x, y), q, r in zip(zip(xs, ys), qs, rs)
           if not (x == info.min and y == -1)]
    expected = [(x // y, x % y) for x, y in pairs]
    assert got == expected
    for (x, y), (q, r) in zip(pairs, expected):
        assert abs(r) < abs(y)
        assert r == 0 or (r > 0) == (y > 0)


@given(pair=int_array_pairs(), data=st.data())
def test_a5_clip_bounds_and_idempotence(pair, data):
    """A5: np.clip(a, lo, hi) lies within [lo, hi] elementwise and clipping
    again changes nothing."""
    dtype, a, _ = pair
    lo = data.draw(hnp.from_dtype(dtype))
    hi = data.draw(hnp.from_dtype(dtype))
    if int(lo) > int(hi):
        lo, hi = hi, lo
    clipped = np.clip(a, lo, hi)
    assert np.all(clipped >= lo)
    assert np.all(clipped <= hi)
    assert np.array_equal(np.clip(clipped, lo, hi), clipped)
    # elements already inside the bounds are untouched
    inside = (a >= lo) & (a <= hi)
    assert np.array_equal(clipped[inside], a[inside])


@given(pair=int_array_pairs(), data=st.data())
def test_a6_where_select_consistency(pair, data):
    """A6: np.where(c, x, y) equals the elementwise model and np.select
    with a single condition and default y."""
    _, x, y = pair
    cond = data.draw(hnp.arrays(np.dtype(bool), x.shape))
    got = np.where(cond, x, y)
    model = np.array(
        [xv if c else yv for c, xv, yv in
         zip(cond.ravel().tolist(), x.ravel().tolist(), y.ravel().tolist())],
        dtype=x.dtype).reshape(x.shape)
    assert np.array_equal(got, model)
    assert np.array_equal(np.select([cond], [x], default=y), got)
