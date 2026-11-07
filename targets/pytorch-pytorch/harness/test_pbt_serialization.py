# Properties S1-S2: serialisation round trips. Source: torch.save /
# torch.load docs — a saved tensor reloads with the same values, dtype,
# shape and layout (strides), and the weights_only=True load path (the
# default since 2.6) must accept plain tensors.
import io

import numpy as np
import torch
from hypothesis import given
import hypothesis.strategies as st
import hypothesis.extra.numpy as hnp

from pbt_strategies import np_arrays, to_torch

NP_SER_DTYPES = [np.dtype(c) for c in
                 ["?", "i1", "i2", "i4", "i8", "u1", "f4", "f8", "c8", "c16"]]


def _round_trip(t, weights_only=True):
    buf = io.BytesIO()
    torch.save(t, buf)
    buf.seek(0)
    return torch.load(buf, weights_only=weights_only)


@given(data=st.data())
def test_s1_save_load_round_trip_exact(data):
    """S1: save/load preserves values, dtype, shape, strides and
    contiguity exactly — including non-contiguous (transposed) layouts —
    through the weights_only load path."""
    a = data.draw(np_arrays(NP_SER_DTYPES,
                            shapes=hnp.array_shapes(min_dims=0, max_dims=3,
                                                    max_side=4)))
    t = to_torch(a)
    if a.ndim >= 2 and data.draw(st.booleans()):
        t = t.transpose(0, 1)  # non-contiguous layout
    loaded = _round_trip(t, weights_only=True)
    assert loaded.dtype == t.dtype
    assert loaded.shape == t.shape
    assert loaded.stride() == t.stride()
    assert loaded.is_contiguous() == t.is_contiguous()
    assert torch.equal(loaded, t)


@given(dt=st.sampled_from([torch.bool, torch.uint8, torch.int8,
                           torch.int64, torch.float16, torch.bfloat16,
                           torch.float32, torch.float64, torch.complex64,
                           torch.complex128]),
       shape=hnp.array_shapes(max_dims=2, max_side=3))
def test_s2_round_trip_all_dtypes_and_empty(dt, shape):
    """S2: 0-dim, empty and small tensors of every dtype (incl. the ones
    NumPy cannot mirror, like bfloat16) round-trip exactly."""
    values = torch.arange(int(np.prod(shape)) if shape else 1,
                          dtype=torch.float64).reshape(shape) if shape else \
        torch.tensor(7, dtype=torch.float64)
    t = values.to(dt)
    loaded = _round_trip(t, weights_only=True)
    assert loaded.dtype == t.dtype
    assert loaded.shape == t.shape
    assert torch.equal(loaded, t)
