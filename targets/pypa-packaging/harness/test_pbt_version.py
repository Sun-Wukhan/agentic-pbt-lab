# Version properties V1-V10 (see ../PROPERTIES.md).
import pickle

from hypothesis import given
from hypothesis import strategies as st

from packaging.utils import canonicalize_version
from packaging.version import Version

from pbt_strategies import (
    related_parts,
    render_canonical,
    render_messy,
    version_part_records,
    versions,
)


@given(parts=version_part_records)
def test_v1_str_round_trip(parts):
    v = Version(render_canonical(parts))
    v2 = Version(str(v))
    assert v2 == v
    assert str(v2) == str(v)


@given(data=st.data())
def test_v2_messy_spelling_same_version(data):
    parts = data.draw(version_part_records)
    canonical = Version(render_canonical(parts))
    messy = Version(data.draw(render_messy(parts)))
    assert messy == canonical
    assert hash(messy) == hash(canonical)
    assert str(messy) == str(canonical)


@given(a=versions(), b=versions())
def test_v3_trichotomy_and_operator_coherence(a, b):
    outcomes = [a < b, a == b, a > b]
    assert sum(outcomes) == 1
    assert (a <= b) == (a < b or a == b)
    assert (a >= b) == (a > b or a == b)
    assert (a != b) == (not (a == b))
    # Reflected comparisons must agree.
    assert (b < a) == (a > b)
    assert (b <= a) == (a >= b)
    assert (b == a) == (a == b)


@given(a=versions(), b=versions(), c=versions())
def test_v4_transitivity_and_sorting(a, b, c):
    if a <= b and b <= c:
        assert a <= c
    if a < b and b < c:
        assert a < c
    if a == b and b == c:
        assert a == c
    ordered = sorted([c, a, b])
    assert ordered[0] <= ordered[1] <= ordered[2]
    assert sorted(ordered, reverse=True) == ordered[::-1]


@given(a=versions(), b=versions())
def test_v5_hash_eq_consistency(a, b):
    assert hash(a) == hash(a)  # stable across calls (cache-safe)
    if a == b:
        assert hash(a) == hash(b)
        assert len({a, b}) == 1


@given(parts=version_part_records)
def test_v6_pep440_family_order(parts):
    release_str = ".".join(map(str, parts.release))
    r = (f"{parts.epoch}!" if parts.epoch else "") + release_str
    base = Version(r)
    dev = Version(r + ".dev1")
    a = Version(r + "a1")
    b = Version(r + "b1")
    rc = Version(r + "rc1")
    post = Version(r + ".post1")
    assert dev < a < b < rc < base < post
    # Local segments: present sorts after absent; numeric compare
    # numerically; alphanumeric sorts before numeric (PEP 440).
    assert base < Version(r + "+1")
    assert Version(r + "+abc") < Version(r + "+abd")
    assert Version(r + "+abc") < Version(r + "+1")
    assert Version(r + "+2") < Version(r + "+10")
    # Epoch dominates everything without one.
    assert Version(f"1!{release_str}") > Version("999.999.999")


@given(parts=version_part_records)
def test_v7_attribute_coherence(parts):
    v = Version(render_canonical(parts))
    assert v.is_prerelease == (v.pre is not None or v.dev is not None)
    assert v.is_devrelease == (v.dev is not None)
    assert v.is_postrelease == (v.post is not None)
    assert v.public == str(v).split("+", 1)[0]
    assert Version(v.public) == v.__replace__(local=None)
    assert Version(v.base_version) == v.__replace__(
        pre=None, post=None, dev=None, local=None
    )
    padded = (list(v.release) + [0, 0, 0])[:3]
    assert (v.major, v.minor, v.micro) == tuple(padded)


@given(parts=version_part_records)
def test_v8_from_parts_and_replace(parts):
    v = Version(render_canonical(parts))
    built = Version.from_parts(
        epoch=parts.epoch,
        release=parts.release,
        pre=parts.pre,
        post=parts.post,
        dev=parts.dev,
        local=".".join(str(x) for x in parts.local) if parts.local else None,
    )
    assert built == v
    assert hash(built) == hash(v)
    assert str(built) == str(v)
    assert v.__replace__() is v
    moved = v.__replace__(epoch=parts.epoch + 1)
    assert moved.epoch == parts.epoch + 1
    assert moved.release == v.release
    assert Version(str(moved)) == moved


@given(v=versions())
def test_v9_pickle_round_trip(v):
    w = pickle.loads(pickle.dumps(v))
    assert w == v
    assert hash(w) == hash(v)
    assert str(w) == str(v)


@given(v=versions())
def test_v10_canonicalize_version(v):
    canon = canonicalize_version(str(v))
    assert Version(canon) == v
    assert canonicalize_version(canon) == canon  # idempotent
    trimmed = v.__replace__(release=_trimmed(v.release))
    assert canon == str(trimmed)
    assert canonicalize_version(str(v), strip_trailing_zero=False) == str(v)


def _trimmed(release):
    rel = list(release)
    while len(rel) > 1 and rel[-1] == 0:
        rel.pop()
    return tuple(rel)


@given(data=st.data())
def test_v8_related_versions_stay_comparable(data):
    # Versions from one family (the related_parts mutator) must still
    # satisfy trichotomy pairwise -- dense coverage around boundaries.
    base = data.draw(version_part_records)
    a = Version(render_canonical(base))
    b = Version(render_canonical(data.draw(related_parts(base))))
    assert sum([a < b, a == b, a > b]) == 1
    if a == b:
        assert hash(a) == hash(b)
