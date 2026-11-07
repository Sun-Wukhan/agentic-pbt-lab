# Shared Hypothesis strategies for the pytorch-pytorch PBT lab.
#
# Box constraint: shared VM memory is tight, so every shape here is small
# (sides 0..6, at most 3 dims, 1-D arrays at most 12 long). Tensors are
# built from NumPy arrays (torch.from_numpy) so the same generated values
# feed both sides of every differential property.
import numpy as np
import torch
import hypothesis.strategies as st
import hypothesis.extra.numpy as hnp

# All dtypes torch.promote_types knows on every platform. bfloat16 and
# complex32 have no NumPy array counterpart; the promotion properties
# only ever build 0-element tensors, so that is fine.
PROMOTION_DTYPES = st.sampled_from([
    torch.bool, torch.uint8, torch.int8, torch.int16, torch.int32,
    torch.int64, torch.float16, torch.bfloat16, torch.float32,
    torch.float64, torch.complex32, torch.complex64, torch.complex128,
])

NP_INT_DTYPES = [np.dtype(c) for c in ["i1", "i2", "i4", "i8", "u1"]]
NP_FLOAT_DTYPES = [np.dtype("f4"), np.dtype("f8")]
NP_SORT_DTYPES = NP_INT_DTYPES + NP_FLOAT_DTYPES

SMALL_SHAPES = hnp.array_shapes(max_dims=3, max_side=4)


def _float_elements(dtype):
    width = 32 if dtype == np.dtype("f4") else 64
    return st.floats(allow_nan=False, allow_infinity=False, width=width)


@st.composite
def np_arrays(draw, dtypes, shapes=None):
    """A small NumPy array over the given dtypes (finite floats)."""
    dtype = draw(st.sampled_from(dtypes))
    shape = draw(shapes if shapes is not None else SMALL_SHAPES)
    if dtype.kind == "f":
        arr = draw(hnp.arrays(dtype, shape, elements=_float_elements(dtype)))
    elif dtype.kind == "c":
        arr = draw(hnp.arrays(dtype, shape,
                              elements=st.complex_numbers(
                                  allow_nan=False, allow_infinity=False)))
    else:
        arr = draw(hnp.arrays(dtype, shape))
    # hnp.arrays with shape () yields a NumPy scalar, not a 0-d array.
    return np.asarray(arr)


@st.composite
def broadcastable_np_pairs(draw, dtypes):
    """Two NumPy arrays whose shapes always broadcast together.

    Draws a target shape, then for each side independently drops leading
    dims and collapses some dims to 1 (classic broadcasting inputs).
    Returns (target_shape, a, b).
    """
    ndim = draw(st.integers(0, 3))
    target = tuple(draw(st.integers(1, 4)) for _ in range(ndim))

    def side_shape():
        dims = list(target)
        for i in range(len(dims)):
            if draw(st.booleans()):
                dims[i] = 1
        drop = draw(st.integers(0, len(dims)))
        return tuple(dims[drop:])

    dtype = draw(st.sampled_from(dtypes))
    sa, sb = side_shape(), side_shape()
    if dtype.kind == "f":
        a = draw(hnp.arrays(dtype, sa, elements=_float_elements(dtype)))
        b = draw(hnp.arrays(dtype, sb, elements=_float_elements(dtype)))
    else:
        a = draw(hnp.arrays(dtype, sa))
        b = draw(hnp.arrays(dtype, sb))
    # The drawn target is only scaffolding: if BOTH sides drop a leading
    # dim, that dim is gone from the result. The real broadcast target is
    # whatever the two side shapes actually broadcast to.
    return np.broadcast_shapes(sa, sb), np.asarray(a), np.asarray(b)


@st.composite
def factorisations(draw, n):
    """A shape (tuple of ints >= 0) whose element count is exactly n."""
    if n == 0:
        return draw(st.sampled_from([(0,), (2, 0), (0, 3), (1, 0, 4)]))
    divisors = [d for d in range(1, n + 1) if n % d == 0]
    kind = draw(st.sampled_from(["flat", "pair", "padded"]))
    if kind == "flat":
        return (n,)
    if kind == "pair":
        d = draw(st.sampled_from(divisors))
        return (d, n // d)
    # padded: a divisor pair with 1s inserted around it
    d = draw(st.sampled_from(divisors))
    base = [d, n // d]
    pos = draw(st.integers(0, 2))
    base.insert(pos, 1)
    return tuple(base)


@st.composite
def slices_1d(draw, n):
    """A slice object valid (in the Python sense) for a length-n axis."""
    def bound():
        if draw(st.booleans()):
            return None
        return draw(st.integers(-n - 2, n + 2))

    # Positive steps only: torch does not support negative-step slicing
    # at all (ValueError: step must be greater than zero) — a deliberate,
    # long-standing divergence from NumPy, triaged in findings/003 and
    # pinned by test_i3b. The original version of this strategy drew
    # negative steps and I3 failed on exactly that.
    step = draw(st.sampled_from([None, 1, 2, 3]))
    return slice(bound(), bound(), step)


def to_torch(np_array):
    # NumPy 2 returns scalars (not 0-d arrays) from 0-d ufunc results;
    # normalise those back to 0-d ndarrays first.
    np_array = np.asarray(np_array)
    # NOTE: np.ascontiguousarray would promote a 0-d array to shape (1,),
    # so only normalise when the array is actually non-contiguous.
    if not np_array.flags.c_contiguous:
        np_array = np.ascontiguousarray(np_array)
    return torch.from_numpy(np_array)
