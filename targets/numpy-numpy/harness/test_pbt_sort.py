# Properties S1-S5: sort / argsort / partition invariants. Sources:
# np.argsort (returns indices that would sort the array), np.sort,
# np.partition documentation: "the value in the kth position is in the
# position it would be in a sorted array; all smaller elements will be
# moved before it and all larger elements behind it".
from collections import Counter

import numpy as np
from hypothesis import given, assume
import hypothesis.strategies as st

from pbt_strategies import sortable_arrays_1d


def _is_non_decreasing(a):
    return all(a[i] <= a[i + 1] for i in range(len(a) - 1))


@given(a=sortable_arrays_1d())
def test_s1_argsort_is_permutation(a):
    """S1: argsort(a) is a permutation of range(len(a))."""
    idx = np.argsort(a)
    assert sorted(idx.tolist()) == list(range(a.size))


@given(a=sortable_arrays_1d())
def test_s2_argsort_sorts_and_matches_sort(a):
    """S2: a[argsort(a)] is non-decreasing and equals np.sort(a)."""
    gathered = a[np.argsort(a)]
    assert _is_non_decreasing(gathered.tolist())
    assert np.array_equal(gathered, np.sort(a))


@given(a=sortable_arrays_1d())
def test_s3_sort_is_permutation_of_input(a):
    """S3: np.sort(a) is a permutation (same multiset) of a, and sorted."""
    s = np.sort(a)
    assert _is_non_decreasing(s.tolist())
    assert Counter(s.tolist()) == Counter(a.tolist())


@given(a=sortable_arrays_1d(), data=st.data())
def test_s4_partition_split_property(a, data):
    """S4: np.partition(a, k): element at k equals sorted(a)[k], everything
    before is <= it, everything after is >= it, and the result is a
    permutation of the input."""
    assume(a.size > 0)
    k = data.draw(st.integers(0, a.size - 1))
    p = np.partition(a, k)
    kth_sorted = np.sort(a)[k]
    assert p[k] == kth_sorted
    assert all(x <= p[k] for x in p[:k].tolist())
    assert all(x >= p[k] for x in p[k:].tolist())
    assert Counter(p.tolist()) == Counter(a.tolist())


@given(a=sortable_arrays_1d())
def test_s5_stable_argsort_tie_order(a):
    """S5: with kind='stable', equal elements keep their original index
    order in the argsort output."""
    idx = np.argsort(a, kind="stable")
    gathered = a[idx]
    assert _is_non_decreasing(gathered.tolist())
    idx_list = idx.tolist()
    vals = gathered.tolist()
    for i in range(len(vals) - 1):
        if vals[i] == vals[i + 1]:
            assert idx_list[i] < idx_list[i + 1]
