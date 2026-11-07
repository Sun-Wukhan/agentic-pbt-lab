# Properties D1-D5: the dtype promotion lattice (torch.promote_types /
# torch.result_type). Sources: the "Type Promotion" documentation page.
# promote_types is documented to compute the dtype with the smallest size
# and scalar kind meeting both inputs' kind/size requirements; result_type
# additionally defines how tensors combine with Python scalars, which are
# treated as "wrapped numbers" with a weak dtype of their category:
#   * scalar category <= tensor category  -> the tensor's dtype wins
#     (a Python int never changes any tensor's dtype, etc.)
#   * scalar category above the tensor's  -> the default dtype of the
#     scalar's category (int64 / default float / default complex),
#     EXCEPT a complex scalar meeting a float tensor yields the complex
#     counterpart of that float dtype (float16 -> complex32, ...).
# D4 encodes exactly that documented rule as a reference implementation.
import torch
from hypothesis import given
import hypothesis.strategies as st

from pbt_strategies import PROMOTION_DTYPES

_CATEGORY = {torch.bool: 0}  # everything else filled in below


def _category(dt):
    if dt == torch.bool:
        return 0
    if dt.is_complex:
        return 3
    if dt.is_floating_point:
        return 2
    return 1  # integral (signed and unsigned)


_COMPLEX_COUNTERPART = {
    torch.float16: torch.complex32,
    torch.bfloat16: torch.bcomplex32,
    torch.float32: torch.complex64,
    torch.float64: torch.complex128,
}

_SCALARS = st.sampled_from([
    (1, 1),        # python int    -> integral category
    (1.0, 2),      # python float  -> floating category
    (1j, 3),       # python complex-> complex category
])


def _expected_with_scalar(tensor_dtype, scalar_category):
    if scalar_category <= _category(tensor_dtype):
        return tensor_dtype
    if scalar_category == 1:
        return torch.int64  # default integer dtype
    if scalar_category == 2:
        return torch.get_default_dtype()  # default float dtype
    # complex scalar above the tensor's category
    if tensor_dtype.is_floating_point:
        return _COMPLEX_COUNTERPART[tensor_dtype]
    return torch.complex64  # default complex dtype


@given(a=PROMOTION_DTYPES, b=PROMOTION_DTYPES)
def test_d1_promote_types_commutative(a, b):
    """D1: promote_types(a, b) == promote_types(b, a)."""
    assert torch.promote_types(a, b) == torch.promote_types(b, a)


@given(a=PROMOTION_DTYPES)
def test_d2_promote_types_idempotent(a):
    """D2: promote_types(a, a) == a."""
    assert torch.promote_types(a, a) == a


@given(a=PROMOTION_DTYPES, b=PROMOTION_DTYPES)
def test_d3_result_type_of_tensors_is_promote_types(a, b):
    """D3: result_type of two tensors == promote_types of their dtypes,
    and is commutative (tensor/tensor promotion is dtype-only)."""
    ea, eb = torch.empty(0, dtype=a), torch.empty(0, dtype=b)
    assert torch.result_type(ea, eb) == torch.promote_types(a, b)
    assert torch.result_type(eb, ea) == torch.promote_types(a, b)


@given(dt=PROMOTION_DTYPES, scalar=_SCALARS)
def test_d4_result_type_with_python_scalars(dt, scalar):
    """D4: result_type(tensor, python scalar) follows the documented
    weak-scalar rule (reference implementation above)."""
    value, category = scalar
    got = torch.result_type(torch.empty(0, dtype=dt), value)
    expected = _expected_with_scalar(dt, category)
    assert got == expected, (
        f"result_type({dt}, {value!r}) = {got}, documented rule says {expected}")


@given(a=PROMOTION_DTYPES, b=PROMOTION_DTYPES, c=PROMOTION_DTYPES)
def test_d5_promote_types_associative_probe(a, b, c):
    """D5 (deliberate probe): promote_types(promote_types(a, b), c) ==
    promote_types(a, promote_types(b, c)).

    The promotion doc defines a binary least-upper-bound operation but
    never promises an associative lattice (NumPy's docstring for the same
    function warns it is 'rarely associative'). Stated to map the actual
    behaviour; a failure triages as wrong property by design.
    """
    lhs = torch.promote_types(torch.promote_types(a, b), c)
    rhs = torch.promote_types(a, torch.promote_types(b, c))
    assert lhs == rhs, f"promote({a},{b},{c}): {lhs} != {rhs}"
