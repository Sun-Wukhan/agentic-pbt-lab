# Utils properties U1-U2 (see ../PROPERTIES.md).
import re

from hypothesis import given
from hypothesis import strategies as st

from packaging.utils import canonicalize_name, is_normalized_name

from pbt_strategies import messy_names, name_segment, project_names

_PEP503_REFERENCE = re.compile(r"[-_.]+")


def _reference(name: str) -> str:
    return _PEP503_REFERENCE.sub("-", name).lower()


@given(data=st.data())
def test_u1_canonicalize_name_model(data):
    name = data.draw(messy_names(data.draw(project_names)))
    result = canonicalize_name(name)
    assert result == _reference(name)
    assert canonicalize_name(result) == result  # idempotent


@given(name=project_names)
def test_u2_is_normalized_name(name):
    canon = canonicalize_name(name)
    # project_names generates valid names (alnum start/end), so the
    # canonical form must be a normalized name, and a normalized name
    # must be a fixed point of canonicalize_name.
    assert is_normalized_name(canon)
    assert canonicalize_name(canon) == canon


@given(seg=name_segment, data=st.data())
def test_u2_normalized_fixed_point(seg, data):
    candidate = data.draw(messy_names(seg.lower()))
    if is_normalized_name(candidate):
        assert canonicalize_name(candidate) == candidate
