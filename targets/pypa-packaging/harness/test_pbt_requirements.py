# Requirement properties R1-R4 (see ../PROPERTIES.md).
from hypothesis import given
from hypothesis import strategies as st

from packaging.requirements import Requirement
from packaging.utils import canonicalize_name

from pbt_strategies import (
    marker_environments,
    messy_names,
    project_names,
    requirement_strings,
)


@given(s=requirement_strings())
def test_r1_requirement_round_trip(s):
    r = Requirement(s)
    r2 = Requirement(str(r))
    assert r2 == r
    assert hash(r2) == hash(r)
    assert str(r2) == str(r)


@given(name=project_names, data=st.data())
def test_r2_requirement_spelling_invariance(name, data):
    messy = data.draw(messy_names(name))
    r1 = Requirement(name + ">=1.0")
    r2 = Requirement(messy + ">=1.0")
    assert canonicalize_name(name) == canonicalize_name(messy)
    assert r1 == r2
    assert hash(r1) == hash(r2)
    e1 = Requirement(name + "[Foo_Bar]>=1.0")
    e2 = Requirement(name + "[foo-bar]>=1.0")
    assert e1 == e2
    assert hash(e1) == hash(e2)


@given(s=requirement_strings())
def test_r3_requirement_parts_preserved(s):
    r = Requirement(s)
    r2 = Requirement(str(r))
    assert r2.name == r.name
    assert r2.url == r.url
    assert r2.extras == r.extras
    assert r2.specifier == r.specifier
    assert r2.marker == r.marker


@given(s=requirement_strings(), env=marker_environments())
def test_r4_requirement_marker_evaluation(s, env):
    r = Requirement(s)
    if r.marker is None:
        return
    r2 = Requirement(str(r))
    assert r2.marker is not None
    assert r2.marker.evaluate(env, context="requirement") == r.marker.evaluate(
        env, context="requirement"
    )
