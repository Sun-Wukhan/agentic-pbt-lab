# Properties B1-B3: broadcasting. Sources: np.broadcast_shapes docstring
# (returns the broadcasted shape, ValueError when shapes are incompatible)
# and the "Broadcasting" rules in the NumPy user guide.
import numpy as np
from hypothesis import given
import hypothesis.strategies as st

from pbt_strategies import BROADCAST_SHAPES


def _try_broadcast_shapes(*shapes):
    try:
        return np.broadcast_shapes(*shapes)
    except ValueError:
        return None


@given(s1=BROADCAST_SHAPES, s2=BROADCAST_SHAPES)
def test_b1_broadcast_shapes_agrees_with_broadcast(s1, s2):
    """B1: broadcast_shapes(s1, s2) succeeds exactly when np.broadcast of
    two real arrays with those shapes succeeds, with the same result."""
    expected = _try_broadcast_shapes(s1, s2)
    try:
        got = np.broadcast(np.empty(s1), np.empty(s2)).shape
    except ValueError:
        got = None
    assert expected == got


@given(s1=BROADCAST_SHAPES, s2=BROADCAST_SHAPES)
def test_b2_broadcast_shapes_commutative(s1, s2):
    """B2: broadcast_shapes(s1, s2) == broadcast_shapes(s2, s1), including
    agreement on failure."""
    assert _try_broadcast_shapes(s1, s2) == _try_broadcast_shapes(s2, s1)


@given(s1=BROADCAST_SHAPES, s2=BROADCAST_SHAPES, s3=BROADCAST_SHAPES,
       data=st.data())
def test_b3_broadcast_shapes_associative(s1, s2, s3, data):
    """B3: broadcasting is associative: bs(bs(s1, s2), s3) == bs(s1, bs(s2, s3)),
    with None (incompatible) propagating on both sides."""
    left_inner = _try_broadcast_shapes(s1, s2)
    right_inner = _try_broadcast_shapes(s2, s3)
    lhs = _try_broadcast_shapes(left_inner, s3) if left_inner is not None else None
    rhs = _try_broadcast_shapes(s1, right_inner) if right_inner is not None else None
    assert lhs == rhs
    # and the flat 3-way call agrees with both groupings
    assert _try_broadcast_shapes(s1, s2, s3) == lhs
