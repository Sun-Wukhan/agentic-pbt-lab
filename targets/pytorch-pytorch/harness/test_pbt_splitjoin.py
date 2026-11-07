# Properties C1-C3: split/join inverses. Sources: torch.split ("the
# tensor is split into chunks ... concatenating them along dim recovers
# the tensor"), torch.chunk (may return fewer chunks than requested),
# torch.stack / torch.unbind (documented inverses of each other).
import torch
from hypothesis import given, assume
import hypothesis.strategies as st
import hypothesis.extra.numpy as hnp
import numpy as np

from pbt_strategies import np_arrays, to_torch


@given(data=st.data())
def test_c1_cat_of_split_restores_tensor(data):
    """C1: cat(split(t, sizes, dim)) == t for any composition of sizes
    summing to the dim length."""
    a = data.draw(np_arrays([np.dtype("i8"), np.dtype("f8")],
                            shapes=hnp.array_shapes(min_dims=1, max_dims=3,
                                                    max_side=5)))
    t = to_torch(a)
    dim = data.draw(st.integers(0, a.ndim - 1))
    n = a.shape[dim]
    # a composition of n into positive parts (n == 0 handled as [ ])
    if n == 0:
        sizes = []
    else:
        cuts = sorted(data.draw(
            st.lists(st.integers(1, n - 1), unique=True)) ) if n > 1 else []
        sizes, prev = [], 0
        for c in cuts + [n]:
            sizes.append(c - prev)
            prev = c
    chunks = torch.split(t, sizes, dim=dim) if sizes else [t]
    if sizes:
        assert torch.equal(torch.cat(chunks, dim=dim), t)


@given(data=st.data())
def test_c2_chunk_partitions_and_cat_restores(data):
    """C2: chunk(t, k, dim) returns at most k chunks whose sizes sum to
    the dim length and differ by at most 1, and cat restores t."""
    a = data.draw(np_arrays([np.dtype("i8"), np.dtype("f8")],
                            shapes=hnp.array_shapes(min_dims=1, max_dims=2,
                                                    max_side=6)))
    t = to_torch(a)
    dim = data.draw(st.integers(0, a.ndim - 1))
    k = data.draw(st.integers(1, 6))
    chunks = torch.chunk(t, k, dim=dim)
    assert len(chunks) <= k
    sizes = [c.shape[dim] for c in chunks]
    assert sum(sizes) == a.shape[dim]
    if sizes:
        assert max(sizes) - min(sizes) <= 1
    if chunks:
        assert torch.equal(torch.cat(chunks, dim=dim), t)


@given(data=st.data())
def test_c3_stack_unbind_inverses(data):
    """C3: stack(unbind(t, dim)) == t, and unbind(stack(ts)) returns the
    original tensors element by element."""
    a = data.draw(np_arrays([np.dtype("i8"), np.dtype("f4")],
                            shapes=hnp.array_shapes(min_dims=1, max_dims=3,
                                                    max_side=4)))
    t = to_torch(a)
    dim = data.draw(st.integers(0, a.ndim - 1))
    assert torch.equal(torch.stack(torch.unbind(t, dim=dim), dim=dim), t)
    ts = list(torch.unbind(t, dim=dim))
    back = torch.unbind(torch.stack(ts, dim=dim), dim=dim)
    assert len(back) == len(ts)
    for x, y in zip(back, ts):
        assert torch.equal(x, y)
