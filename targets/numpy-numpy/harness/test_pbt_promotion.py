# Properties P1-P4: the dtype promotion lattice (np.promote_types /
# np.result_type). Sources: the np.promote_types docstring ("This function
# is symmetric, but rarely associative") and the NEP 50 promotion rules as
# documented for np.result_type.
import numpy as np
from hypothesis import given

from pbt_strategies import PROMOTION_DTYPES


@given(a=PROMOTION_DTYPES, b=PROMOTION_DTYPES)
def test_p1_promote_types_commutative(a, b):
    """P1: promote_types(a, b) == promote_types(b, a) — docstring: symmetric."""
    assert np.promote_types(a, b) == np.promote_types(b, a)


@given(a=PROMOTION_DTYPES)
def test_p2_promote_types_idempotent(a):
    """P2: promote_types(a, a) == a (a is its own least upper bound)."""
    assert np.promote_types(a, a) == a


@given(a=PROMOTION_DTYPES, b=PROMOTION_DTYPES, c=PROMOTION_DTYPES)
def test_p3_promote_types_associative(a, b, c):
    """P3: promote_types(promote_types(a, b), c)
    == promote_types(a, promote_types(b, c)).

    Stated as the lattice property one would want; the docstring warns the
    function is "rarely associative", so a failure here is expected to
    triage as a wrong property, not a bug. Recorded deliberately.
    """
    lhs = np.promote_types(np.promote_types(a, b), c)
    rhs = np.promote_types(a, np.promote_types(b, c))
    assert lhs == rhs, f"promote({a},{b},{c}): {lhs} != {rhs}"


@given(a=PROMOTION_DTYPES, b=PROMOTION_DTYPES)
def test_p4_result_type_agrees_with_promote_types(a, b):
    """P4: result_type of two arrays equals promote_types of their dtypes
    (NEP 50: array-array promotion is dtype-only, no value-based rules)."""
    ea = np.empty(0, dtype=a)
    eb = np.empty(0, dtype=b)
    assert np.result_type(ea, eb) == np.promote_types(a, b)
    assert np.result_type(a, b) == np.promote_types(a, b)
