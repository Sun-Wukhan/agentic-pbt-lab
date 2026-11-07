# Properties I1-I3: indexing. Sources: torch boolean-mask indexing
# (selects elements where the mask is True, in order), gather/scatter
# docs (scatter writes src at index; gather with the same index reads
# those values back — an exact inverse when the indices are unique
# along each gather line), and basic slicing, which the torch indexing
# doc defines to follow NumPy semantics.
import numpy as np
import torch
from hypothesis import given
import hypothesis.strategies as st
import hypothesis.extra.numpy as hnp

from pbt_strategies import np_arrays, slices_1d, to_torch


@given(data=st.data())
def test_i1_boolean_mask_selection(data):
    """I1: t[mask] has exactly mask.sum() elements and equals NumPy's
    masked selection element-for-element."""
    a = data.draw(np_arrays([np.dtype("i8"), np.dtype("f8")]))
    mask = data.draw(hnp.arrays(np.dtype(bool), a.shape))
    t, m = to_torch(a), to_torch(mask)
    selected = t[m]
    assert selected.numel() == int(mask.sum())
    assert torch.equal(selected, to_torch(a[mask]))


@given(data=st.data())
def test_i2_gather_scatter_inverse_on_unique_indices(data):
    """I2: scatter zeros at unique indices from src, then gather at the
    same indices, returns src exactly (both dims of a 2-D tensor)."""
    rows = data.draw(st.integers(1, 4))
    cols = data.draw(st.integers(1, 5))
    dim = data.draw(st.sampled_from([0, 1]))
    if dim == 1:
        k = data.draw(st.integers(0, cols))
        idx = np.empty((rows, k), dtype=np.int64)
        for r in range(rows):
            idx[r] = data.draw(st.permutations(range(cols)))[:k]
    else:
        k = data.draw(st.integers(0, rows))
        idx = np.empty((k, cols), dtype=np.int64)
        for c in range(cols):
            idx[:, c] = data.draw(st.permutations(range(rows)))[:k]
    src = data.draw(hnp.arrays(np.dtype("i8"), idx.shape))
    base = torch.zeros((rows, cols), dtype=torch.int64)
    idx_t = torch.from_numpy(idx)
    scattered = base.scatter(dim, idx_t, to_torch(src))
    assert torch.equal(scattered.gather(dim, idx_t), to_torch(src))


@given(data=st.data())
def test_i3_basic_slicing_matches_numpy(data):
    """I3: a basic slice (start/stop/step, positive steps) along axis 0
    selects exactly what NumPy selects. Negative steps are excluded:
    torch rejects them wholesale (see I3b / findings/003)."""
    a = data.draw(np_arrays([np.dtype("i8"), np.dtype("f8")],
                            shapes=hnp.array_shapes(min_dims=1, max_dims=3,
                                                    max_side=6)))
    s = data.draw(slices_1d(a.shape[0]))
    t = to_torch(a)
    assert torch.equal(t[s], to_torch(a[s]))


# Fixed characterisation cases for I3b (not a @given property): slices
# NumPy evaluates happily, every one of which torch rejects.
_NEGATIVE_STEP_SLICES = [
    slice(None, None, -1), slice(2, 0, -1), slice(0, 0, -3),
    slice(5, 1, -2), slice(-1, None, -1), slice(3, None, -2),
    slice(None, 1, -1), slice(-2, -5, -1),
]


def test_i3b_negative_step_slicing_is_a_documented_divergence():
    """I3b (characterisation): for every negative-step slice, torch
    raises ValueError('step must be greater than zero') while NumPy
    returns values; torch.flip is the sanctioned equivalent of [::-1].
    Pins the divergence triaged in findings/003 so a future torch that
    grows negative-step support fails THIS test loudly."""
    a = np.arange(5)
    t = to_torch(a)
    for s in _NEGATIVE_STEP_SLICES:
        _ = a[s]  # NumPy evaluates all of these without error
        try:
            t[s]
        except ValueError as e:
            assert "step must be greater than zero" in str(e)
        else:
            raise AssertionError(f"torch unexpectedly accepted {s}")
    assert torch.equal(torch.flip(t, [0]), to_torch(a[::-1]))
