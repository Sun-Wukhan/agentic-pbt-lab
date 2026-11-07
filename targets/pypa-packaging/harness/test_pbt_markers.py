# Marker properties M1-M7 (see ../PROPERTIES.md).
from hypothesis import given
from hypothesis import strategies as st

from packaging.markers import Marker, default_environment
from packaging.specifiers import Specifier

from pbt_strategies import (
    FULL_VERSIONS,
    SHORT_VERSIONS,
    STRING_KEYS,
    STRING_VALUES,
    marker_atoms,
    marker_environments,
    marker_strings,
)


@given(s=marker_strings())
def test_m1_marker_round_trip(s):
    m = Marker(s)
    m2 = Marker(str(m))
    assert m2 == m
    assert hash(m2) == hash(m)
    assert str(m2) == str(m)


@given(a=marker_strings(), b=marker_strings(), env=marker_environments())
def test_m2_marker_and_or_algebra(a, b, env):
    ma, mb = Marker(a), Marker(b)
    both = ma & mb
    either = ma | mb
    assert both.evaluate(env, context="requirement") == (
        ma.evaluate(env, context="requirement")
        and mb.evaluate(env, context="requirement")
    )
    assert either.evaluate(env, context="requirement") == (
        ma.evaluate(env, context="requirement")
        or mb.evaluate(env, context="requirement")
    )
    assert Marker(str(both)) == both
    assert Marker(str(either)) == either


@given(key=st.sampled_from(STRING_KEYS), env=marker_environments())
def test_m3_marker_tautology(key, env):
    value = env[key]
    tautology = Marker(f'{key} == "{value}" or {key} != "{value}"')
    contradiction = Marker(f'{key} == "{value}" and {key} != "{value}"')
    assert tautology.evaluate(env) is True
    assert contradiction.evaluate(env) is False


@given(
    key=st.sampled_from(
        ["python_version", "python_full_version", "implementation_version"]
    ),
    op=st.sampled_from(["<", "<=", "==", "!=", ">=", ">"]),
    bound=st.sampled_from(SHORT_VERSIONS + FULL_VERSIONS),
    actual=st.sampled_from(SHORT_VERSIONS + FULL_VERSIONS),
)
def test_m4_marker_version_comparisons(key, op, bound, actual):
    marker = Marker(f'{key} {op} "{bound}"')
    env = {key: actual}
    expected = Specifier(f"{op}{bound}").contains(actual, prereleases=True)
    assert marker.evaluate(env, context="requirement") == expected


@given(
    key=st.sampled_from(STRING_KEYS),
    needle=st.text(alphabet="linwx64CPose", min_size=0, max_size=3),
    env=marker_environments(),
)
def test_m5_marker_membership(key, needle, env):
    marker = Marker(f'"{needle}" in {key}')
    assert marker.evaluate(env) == (needle in env[key])
    negated = Marker(f'"{needle}" not in {key}')
    assert negated.evaluate(env) == (needle not in env[key])


def test_m6_marker_extra_normalization():
    m1 = Marker('extra == "Foo_Bar"')
    m2 = Marker('extra == "foo-bar"')
    assert m1 == m2
    assert hash(m1) == hash(m2)
    env = {"extra": "FOO.bar"}
    assert m1.evaluate(env, context="requirement") is True
    assert m2.evaluate(env, context="requirement") is True
    assert Marker('extra == "other"').evaluate(env, context="requirement") is False


@given(atom=marker_atoms())
def test_m7_marker_default_environment(atom):
    env = default_environment()
    env["extra"] = ""
    result = Marker(atom).evaluate(env, context="requirement")
    assert isinstance(result, bool)
    # Atoms that don't reference extra also evaluate against the bare
    # default environment (metadata context supplies extra="").
    if "extra" not in atom:
        assert isinstance(Marker(atom).evaluate(), bool)
