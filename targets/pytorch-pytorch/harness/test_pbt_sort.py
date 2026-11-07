# Properties O1-O3: sorting. Sources: torch.sort docs (ascending order;
# stable=True preserves the order of equal elements), torch.topk docs
# (returns the k largest elements), NumPy sort/argsort as the oracle the
# semantics are documented to match for 1-D inputs.
from collections import Counter

import numpy as np
import torch
from hypothesis import given, assume
import hypothesis.strategies as st
import hypothesis.extra.numpy as hnp

from pbt_strategies import NP_SORT_DTYPES, np_arrays, to_torch


def _one_dim(data):
    return data.draw(np_arrays(
        NP_SORT_DTYPES,
        shapes=hnp.array_shapes(min_dims=1, max_dims=1, max_side=12)))


@given(data=st.data())
def test_o1_sort_values_match_numpy(data):
    """O1: torch.sort values on a 1-D array equal np.sort's exactly
    (ints and finite floats; NaN excluded — see PROPERTIES.md)."""
    a = _one_dim(data)
    t = to_torch(a)
    values, _ = torch.sort(t)
    assert np.array_equal(values.numpy(), np.sort(a))


@given(data=st.data())
def test_o2_stable_sort_indices_match_numpy(data):
    """O2: with stable=True, the returned indices equal NumPy's stable
    argsort exactly (ties keep original order, so the index sequence is
    fully determined)."""
    a = _one_dim(data)
    t = to_torch(a)
    values, indices = torch.sort(t, stable=True)
    expected_idx = np.argsort(a, kind="stable")
    assert np.array_equal(indices.numpy(), expected_idx)
    assert np.array_equal(values.numpy(), a[expected_idx])


@given(data=st.data())
def test_o3_topk_is_sorted_head(data):
    """O3: topk(k) values are exactly the k largest values (the head of
    the descending sort, as multisets), in descending order."""
    a = _one_dim(data)
    assume(a.size > 0)
    k = data.draw(st.integers(1, a.size))
    t = to_torch(a)
    values, _ = torch.topk(t, k)
    vals = values.numpy()
    assert all(vals[i] >= vals[i + 1] for i in range(len(vals) - 1))
    descending = np.sort(a)[::-1]
    assert Counter(vals.tolist()) == Counter(descending[:k].tolist())
