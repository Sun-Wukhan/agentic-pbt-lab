# Specifier / SpecifierSet / VersionRange properties S1-S12
# (see ../PROPERTIES.md).
import dataclasses

from hypothesis import assume, given
from hypothesis import strategies as st

from packaging.specifiers import Specifier, SpecifierSet
from packaging.version import Version

from pbt_strategies import (
    related_parts,
    render_canonical,
    specifier_builds,
    specifier_set_strings,
    specifier_strings,
    version_part_records,
    versions,
)

MIN_VERSION = Version("0.dev0")


@st.composite
def spec_and_candidate(draw):
    """A specifier plus a candidate version drawn densely around it."""
    text, op, parts = draw(specifier_builds())
    roll = draw(st.integers(0, 3))
    if roll == 0:
        cand = Version(render_canonical(parts))
    elif roll == 1:
        cand = Version(render_canonical(draw(related_parts(parts))))
    else:
        cand = draw(versions())
    return text, op, parts, cand


# --------------------------------------------------------------------- S1


@given(data=st.data())
def test_s1_constructed_witnesses(data):
    text, op, parts = data.draw(specifier_builds())
    spec = Specifier(text)
    v = Version(render_canonical(parts))
    if text.endswith(".*"):
        member = Version(
            render_canonical(
                dataclasses.replace(
                    parts, release=parts.release + (5,), pre=("a", 1)
                )
            )
        )
        outsider = Version(
            render_canonical(
                dataclasses.replace(
                    parts,
                    release=parts.release[:-1] + (parts.release[-1] + 1,),
                    pre=None,
                    post=None,
                    dev=None,
                    local=None,
                )
            )
        )
        if op == "==":
            assert spec.contains(member, prereleases=True)
            assert not spec.contains(outsider, prereleases=True)
        else:  # !=
            assert not spec.contains(member, prereleases=True)
            assert spec.contains(outsider, prereleases=True)
        return
    if op == "===":
        assert spec.contains(text[3:], prereleases=True)
        return
    if op in (">=", "<=", "==", "~="):
        assert spec.contains(v, prereleases=True), text
    if op == "!=":
        assert not spec.contains(v, prereleases=True), text
    if op == ">":
        bumped = dataclasses.replace(
            parts,
            release=parts.release[:-1] + (parts.release[-1] + 1,),
            pre=None,
            post=None,
            dev=None,
            local=None,
        )
        assert spec.contains(Version(render_canonical(bumped)), prereleases=True)
        assert not spec.contains(v, prereleases=True)
    if op == "<":
        # PEP 440: <V also excludes V's own pre-release family, so the
        # effective bound is V itself (V pre-release) or V.dev0.  The
        # witness MIN_VERSION only works when that bound is above it
        # (e.g. <0 legitimately contains nothing at all).
        bound = v if v.is_prerelease else v.__replace__(dev=0, local=None)
        assume(bound > MIN_VERSION)
        assert spec.contains(MIN_VERSION, prereleases=True)
        assert not spec.contains(v, prereleases=True)


# --------------------------------------------------------------------- S2


@given(case=spec_and_candidate(), p=st.sampled_from([True, False, None]))
def test_s2_specifier_vs_set_agreement(case, p):
    text, _op, _parts, cand = case
    single = Specifier(text).contains(cand, prereleases=p)
    as_set = SpecifierSet(text).contains(cand, prereleases=p)
    assert single == as_set, (text, str(cand), p)


# --------------------------------------------------------------------- S3


@given(
    s=specifier_set_strings(allow_arbitrary=False),
    w=versions(),
    p=st.sampled_from([True, False]),
)
def test_s3_contains_vs_filter_single(s, w, p):
    ss = SpecifierSet(s)
    assert ss.contains(w, prereleases=p) == (w in list(ss.filter([w], prereleases=p)))


@given(s=specifier_set_strings(allow_arbitrary=False), w=versions())
def test_s3_contains_vs_filter_single_default(s, w):
    ss = SpecifierSet(s)
    assert ss.contains(w) == (w in list(ss.filter([w])))


# --------------------------------------------------------------------- S4


def _is_subsequence(sub, seq):
    it = iter(seq)
    return all(any(x == y for y in it) for x in sub)


@given(
    s=specifier_set_strings(allow_arbitrary=False),
    items=st.lists(versions(), min_size=0, max_size=6),
)
def test_s4_filter_list_semantics(s, items):
    ss = SpecifierSet(s)
    out = list(ss.filter(items))
    assert _is_subsequence(out, items)
    contained = [x for x in items if ss.contains(x)]
    contained_finals = [x for x in contained if not x.is_prerelease]
    if contained_finals and ss.prereleases is not True:
        expected = contained_finals
    else:
        expected = contained
    assert out == expected


# --------------------------------------------------------------------- S5


@given(
    a=specifier_set_strings(allow_arbitrary=False),
    b=specifier_set_strings(allow_arbitrary=False),
    w=versions(),
)
def test_s5_and_combination(a, b, w):
    sa, sb = SpecifierSet(a), SpecifierSet(b)
    combined = sa & sb
    assert combined.contains(w, prereleases=True) == (
        sa.contains(w, prereleases=True) and sb.contains(w, prereleases=True)
    )
    reparsed = SpecifierSet(str(combined))
    assert reparsed == combined
    assert hash(reparsed) == hash(combined)


# --------------------------------------------------------------------- S6


@given(s=specifier_set_strings())
def test_s6_str_round_trip(s):
    ss = SpecifierSet(s)
    ss2 = SpecifierSet(str(ss))
    assert ss2 == ss
    assert hash(ss2) == hash(ss)
    assert str(ss2) == str(ss)


@given(s=specifier_strings())
def test_s6_specifier_str_round_trip(s):
    sp = Specifier(s)
    sp2 = Specifier(str(sp))
    assert sp2 == sp
    assert hash(sp2) == hash(sp)


# --------------------------------------------------------------------- S7


@given(s=specifier_set_strings(), w=versions())
def test_s7_prereleases_false_blocks(s, w):
    assume(w.is_prerelease)
    assert not SpecifierSet(s).contains(w, prereleases=False)
    assert not SpecifierSet(s).contains(w, prereleases=False, installed=False)


@given(parts=version_part_records)
def test_s7_autodetect_and_filter_rule(parts):
    rel = parts.release
    spec = Specifier(f">={render_canonical(dataclasses.replace(parts, pre=('a', 1), post=None, dev=None, local=None))}")
    assert spec.prereleases is True
    named = Version(
        render_canonical(dataclasses.replace(parts, pre=("a", 1), post=None, dev=None, local=None))
    )
    assert spec.contains(named)
    # Lone matching pre-release is yielded; dropped once a matching
    # final release is in the same filter input (PEP 440 rule).
    bumped = rel[:-1] + (rel[-1] + 1,)
    pre = Version(".".join(map(str, bumped)) + "a1")
    fin = Version(".".join(map(str, bumped)))
    ss = SpecifierSet(f">={'.'.join(map(str, rel))}")
    assert list(ss.filter([pre])) == [pre]
    assert list(ss.filter([pre, fin])) == [fin]
    assert list(ss.filter([fin, pre])) == [fin]


# --------------------------------------------------------------------- S8


@given(
    s=specifier_set_strings(),
    pool=st.lists(versions(), min_size=1, max_size=8),
)
def test_s8_unsatisfiable_soundness(s, pool):
    ss = SpecifierSet(s)
    contained = [w for w in pool if ss.contains(w, prereleases=True)]
    if ss.is_unsatisfiable():
        assert not contained, s
    if contained:
        assert not ss.is_unsatisfiable(), s


def test_s8_known_unsatisfiable_examples():
    assert SpecifierSet(">=2.0,<1.0").is_unsatisfiable()
    assert SpecifierSet("==1.0,!=1.0").is_unsatisfiable()
    assert not SpecifierSet(">=1.0,<2.0").is_unsatisfiable()
    assert not SpecifierSet("").is_unsatisfiable()


# --------------------------------------------------------------------- S9


@given(v=versions())
def test_s9_arbitrary_equality(v):
    spec = Specifier(f"==={v}")
    assert spec.contains(str(v), prereleases=True)
    assert spec.contains(v, prereleases=True)
    # Literal, not version, equality: a different string never matches,
    # even when it names an equal PEP 440 version.
    assert not spec.contains(str(v) + ".0", prereleases=True)
    upper = Specifier(f"==={str(v).upper()}")
    assert upper.contains(str(v), prereleases=True)


# -------------------------------------------------------------------- S10


@given(
    a=specifier_set_strings(allow_arbitrary=False, max_specs=2),
    b=specifier_set_strings(allow_arbitrary=False, max_specs=2),
    w=versions(),
)
def test_s10_range_algebra(a, b, w):
    sa = SpecifierSet(a, prereleases=True)
    sb = SpecifierSet(b, prereleases=True)
    ra, rb = sa.to_range(), sb.to_range()
    in_a = sa.contains(w, prereleases=True)
    in_b = sb.contains(w, prereleases=True)
    assert ra.contains(w) == in_a
    assert (ra & rb).contains(w) == (in_a and in_b)
    assert (ra | rb).contains(w) == (in_a or in_b)
    assert (~ra).contains(w) == (not in_a)
    assert (ra - rb).contains(w) == (in_a and not in_b)
    back = ra.to_specifier_set()
    if back is not None:
        assert back.contains(w, prereleases=True) == in_a


# -------------------------------------------------------------------- S11


@given(
    a=specifier_set_strings(allow_arbitrary=False, max_specs=2),
    b=specifier_set_strings(allow_arbitrary=False, max_specs=2),
    pool=st.lists(versions(), min_size=1, max_size=8),
)
def test_s11_relation_soundness(a, b, pool):
    sa, sb = SpecifierSet(a), SpecifierSet(b)
    assert sa.is_subset(sa)
    assert sa.is_superset(sa)
    assert sa.is_subset(sb) == sb.is_superset(sa)
    if sa.is_subset(sb):
        for w in pool:
            assert not (
                sa.contains(w, prereleases=True)
                and not sb.contains(w, prereleases=True)
            ), (a, b, str(w))
    if sa.is_disjoint(sb):
        for w in pool:
            assert not (
                sa.contains(w, prereleases=True) and sb.contains(w, prereleases=True)
            ), (a, b, str(w))


# -------------------------------------------------------------------- S12


@given(parts=version_part_records)
def test_s12_canonical_equality(parts):
    assume(parts.local is None)
    v = render_canonical(parts)
    padded = render_canonical(dataclasses.replace(parts, release=parts.release + (0,)))
    assert Specifier(f"=={v}") == Specifier(f"=={padded}")
    assert hash(Specifier(f"=={v}")) == hash(Specifier(f"=={padded}"))


@given(
    specs=st.lists(specifier_strings(allow_arbitrary=False), min_size=1, max_size=3),
    data=st.data(),
)
def test_s12_set_equality_order_insensitive(specs, data):
    ss1 = SpecifierSet(",".join(specs))
    shuffled = data.draw(st.permutations(specs))
    duplicated = list(shuffled) + specs[:1]
    ss2 = SpecifierSet(",".join(duplicated))
    assert ss1 == ss2
    assert hash(ss1) == hash(ss2)
