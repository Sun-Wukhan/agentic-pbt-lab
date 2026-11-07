# Shared Hypothesis strategies for the numpy-numpy PBT lab.
import numpy as np
import hypothesis.strategies as st
import hypothesis.extra.numpy as hnp

# Dtypes over which promote_types is exercised: bool, all int/uint widths,
# the IEEE float widths available on every platform, and complex64/128.
# (longdouble / complex256 are platform-dependent and are excluded so the
# runs reproduce identically everywhere.)
PROMOTION_DTYPES = st.sampled_from(
    [np.dtype(c) for c in ["?", "b", "B", "h", "H", "i", "I", "q", "Q",
                           "e", "f", "d", "F", "D"]]
)

INT_DTYPES = [np.dtype(c) for c in ["b", "B", "h", "H", "i", "I", "q", "Q"]]

# Fixed-width dtypes (incl. both byte orders) for serialisation round trips.
FIXED_DTYPES = st.sampled_from(
    [np.dtype(c) for c in ["?", "i1", "u1", "<i2", ">i2", "<i4", ">i4",
                           "<i8", ">u2", "<u4", ">u8", "<f2", "<f4", ">f4",
                           "<f8", ">f8", "<c8", ">c8", "<c16", ">c16"]]
)

SMALL_SHAPES = hnp.array_shapes(max_dims=3, max_side=4)
BROADCAST_SHAPES = hnp.array_shapes(max_dims=3, max_side=4)


@st.composite
def int_array_pairs(draw):
    """Two same-dtype, same-shape integer arrays (full value range)."""
    dtype = draw(st.sampled_from(INT_DTYPES))
    shape = draw(hnp.array_shapes(max_dims=2, max_side=5))
    a = draw(hnp.arrays(dtype, shape))
    b = draw(hnp.arrays(dtype, shape))
    return dtype, a, b


@st.composite
def int_arrays_1d(draw):
    """One 1-D integer array, length 0..12."""
    dtype = draw(st.sampled_from(INT_DTYPES))
    return draw(hnp.arrays(dtype, hnp.array_shapes(min_dims=1, max_dims=1,
                                                   max_side=12)))


@st.composite
def sortable_arrays_1d(draw):
    """One 1-D integer or finite-float array, length 0..12."""
    kind = draw(st.sampled_from(["int", "float"]))
    if kind == "int":
        dtype = draw(st.sampled_from(INT_DTYPES))
        return draw(hnp.arrays(dtype, hnp.array_shapes(min_dims=1, max_dims=1,
                                                       max_side=12)))
    return draw(hnp.arrays(
        np.dtype("f8"), hnp.array_shapes(min_dims=1, max_dims=1, max_side=12),
        elements=st.floats(allow_nan=False, allow_infinity=False,
                           width=64)))


@st.composite
def base_numeric_arrays(draw):
    """A small integer/float array used as raw material for astype draws."""
    dtype = draw(st.sampled_from([np.dtype("i8"), np.dtype("f8")]))
    shape = draw(SMALL_SHAPES)
    if dtype == np.dtype("f8"):
        return draw(hnp.arrays(dtype, shape,
                               elements=st.floats(allow_nan=False,
                                                  allow_infinity=False)))
    return draw(hnp.arrays(dtype, shape))
