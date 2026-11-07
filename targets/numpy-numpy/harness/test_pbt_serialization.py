# Properties Z1-Z3: serialisation round trips for fixed dtypes. Sources:
# ndarray.tobytes / np.frombuffer documentation ("construct an array from
# data in a byte string") and np.save / np.load (.npy format stores dtype,
# shape and fortran-order flag in the header).
import io

import numpy as np
from hypothesis import given
import hypothesis.strategies as st

from pbt_strategies import FIXED_DTYPES, base_numeric_arrays


@given(base=base_numeric_arrays(), data=st.data())
def test_z1_tobytes_frombuffer_round_trip(base, data):
    """Z1: frombuffer(a.tobytes(), a.dtype).reshape(a.shape) == a, with the
    dtype (including byte order) preserved exactly."""
    a = base.astype(data.draw(FIXED_DTYPES))
    back = np.frombuffer(a.tobytes(), dtype=a.dtype).reshape(a.shape)
    assert back.dtype == a.dtype
    assert back.shape == a.shape
    assert np.array_equal(back, a)


@given(base=base_numeric_arrays(), data=st.data())
def test_z2_save_load_round_trip(base, data):
    """Z2: np.save -> np.load preserves dtype, shape and values; a
    Fortran-ordered array comes back Fortran-ordered."""
    a = base.astype(data.draw(FIXED_DTYPES))
    buf = io.BytesIO()
    np.save(buf, a)
    buf.seek(0)
    back = np.load(buf)
    assert back.dtype == a.dtype
    assert back.shape == a.shape
    assert np.array_equal(back, a)

    if a.ndim >= 2 and all(d > 1 for d in a.shape):
        af = np.asfortranarray(a)
        buf = io.BytesIO()
        np.save(buf, af)
        buf.seek(0)
        back_f = np.load(buf)
        assert back_f.flags["F_CONTIGUOUS"]
        assert not back_f.flags["C_CONTIGUOUS"]
        assert np.array_equal(back_f, a)


@given(base=base_numeric_arrays(), data=st.data())
def test_z3_tobytes_fortran_order_round_trip(base, data):
    """Z3: tobytes(order='F') read back with reshape(..., order='F')
    restores the array exactly."""
    a = base.astype(data.draw(FIXED_DTYPES))
    back = np.frombuffer(a.tobytes(order="F"), dtype=a.dtype).reshape(
        a.shape, order="F")
    assert back.dtype == a.dtype
    assert np.array_equal(back, a)
