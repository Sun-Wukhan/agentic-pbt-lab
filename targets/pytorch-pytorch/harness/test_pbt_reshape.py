# Properties V1-V4: view / reshape semantics. Sources: torch.reshape,
# torch.permute, torch.squeeze/unsqueeze docs ("returns a view where
# possible", values in C order), and NumPy's reshape as the documented
# equivalent for the value layout of contiguous data.
import numpy as np
import torch
from hypothesis import given, assume
import hypothesis.strategies as st
import hypothesis.extra.numpy as hnp

from pbt_strategies import (NP_FLOAT_DTYPES, NP_INT_DTYPES, factorisations,
                            np_arrays, to_torch)


@given(data=st.data())
def test_v1_reshape_values_match_numpy(data):
    """V1: torch.reshape to any same-count shape holds exactly the values
    of NumPy's reshape of the same data (C-order layout)."""
    a = data.draw(np_arrays(NP_INT_DTYPES + NP_FLOAT_DTYPES))
    target = data.draw(factorisations(a.size))
    t = to_torch(a)
    assert torch.equal(t.reshape(target), to_torch(a.reshape(target)))


@given(a=np_arrays(NP_INT_DTYPES), data=st.data())
def test_v2_permute_inverse_round_trip(a, data):
    """V2: permute by p then by the inverse permutation restores the
    original tensor exactly (shape and values)."""
    assume(a.ndim >= 1)
    perm = data.draw(st.permutations(range(a.ndim)))
    inverse = [0] * a.ndim
    for i, p in enumerate(perm):
        inverse[p] = i
    t = to_torch(a)
    assert torch.equal(t.permute(perm).permute(inverse), t)
    assert t.permute(perm).shape == torch.from_numpy(
        np.ascontiguousarray(np.transpose(a, perm))).shape


@given(a=np_arrays(NP_INT_DTYPES), data=st.data())
def test_v3_squeeze_unsqueeze_and_flatten(a, data):
    """V3: unsqueeze(d).squeeze(d) is the identity; flatten() equals
    NumPy's ravel (C order)."""
    t = to_torch(a)
    d = data.draw(st.integers(0, a.ndim))
    assert torch.equal(t.unsqueeze(d).squeeze(d), t)
    assert torch.equal(t.flatten(), to_torch(a.ravel()))


@given(data=st.data())
def test_v4_transposed_reshape_matches_numpy(data):
    """V4: reshape of a NON-contiguous (transposed) tensor still matches
    NumPy value-for-value: reshape must copy in logical order, never
    reinterpret the transposed storage."""
    a = data.draw(hnp.arrays(np.dtype("i8"),
                             hnp.array_shapes(min_dims=2, max_dims=2,
                                              max_side=4)))
    axes = data.draw(st.sampled_from([(1, 0), (0, 1)]))
    target = data.draw(factorisations(a.size))
    t = to_torch(a).permute(axes)
    expected = np.transpose(a, axes).reshape(target)
    assert torch.equal(t.reshape(target), to_torch(expected))
