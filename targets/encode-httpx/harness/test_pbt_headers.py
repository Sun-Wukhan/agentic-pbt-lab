# Headers properties for the encode-httpx PBT lab. See ../PROPERTIES.md (H1-H8).
from __future__ import annotations

from hypothesis import assume, given
from hypothesis import strategies as st

from httpx import Headers

from pbt_strategies import HEADER_PAIR_LISTS, header_keys


def distinct_keys(pairs):
    return {k.lower() for k, _ in pairs}


# H1 — case-insensitive lookup


@given(pairs=HEADER_PAIR_LISTS, probe=header_keys())
def test_h1_case_insensitive_lookup(pairs, probe):
    h = Headers(pairs)
    lowered = [k.lower() for k, _ in pairs]
    if probe.lower() in lowered:
        expected = ", ".join(v for k, v in pairs if k.lower() == probe.lower())
        assert h[probe] == expected
        assert h[probe.upper()] == expected
        assert h[probe.lower()] == expected
        assert probe in h
        assert h.get_list(probe) == [v for k, v in pairs if k.lower() == probe.lower()]
        del h[probe]
        assert probe.lower() not in [k.lower() for k, _ in h.multi_items()]
    else:
        assert probe not in h
        assert h.get_list(probe) == []


# H2 — Mapping length consistency


@given(pairs=HEADER_PAIR_LISTS)
def test_h2_len_matches_distinct_keys(pairs):
    h = Headers(pairs)
    assert len(h) == len(distinct_keys(pairs))
    assert len(h) == len(list(h.keys()))
    assert len(h) == len(list(iter(h)))


# H3 — concatenated reads agree


@given(pairs=HEADER_PAIR_LISTS)
def test_h3_concatenated_reads_agree(pairs):
    h = Headers(pairs)
    items = dict(h.items())
    for key in distinct_keys(pairs):
        joined = ", ".join(v for k, v in pairs if k.lower() == key)
        assert h[key] == joined
        assert items[key] == joined
    assert list(h.values()) == [items[k] for k in h.keys()]


# H4 — reconstruction fidelity


@given(pairs=HEADER_PAIR_LISTS)
def test_h4_reconstruction_fidelity(pairs):
    h = Headers(pairs)
    assert Headers(h.multi_items()) == h
    assert Headers(h.raw) == h
    assert [k for k, _ in h.raw] == [k.encode("ascii") for k, _ in pairs]


# H5 — setitem semantics


@given(pairs=HEADER_PAIR_LISTS, key=header_keys(), value=st.text(alphabet="abc XYZ012", min_size=0, max_size=5))
def test_h5_setitem_semantics(pairs, key, value):
    h = Headers(pairs)
    before_keys = [k.lower() for k, _ in h.multi_items()]
    h[key] = value
    assert h.get_list(key) == [value]
    after = h.multi_items()
    if key.lower() in before_keys:
        # Replaced key keeps its first-occurrence position.
        assert [k.lower() for k, _ in after].index(key.lower()) == before_keys.index(
            key.lower()
        )
        assert len(after) == len(pairs) - (
            before_keys.count(key.lower()) - 1
        )
    else:
        assert after[-1][0].lower() == key.lower()
        assert len(after) == len(pairs) + 1


# H6 — update semantics


@given(a=HEADER_PAIR_LISTS, b=HEADER_PAIR_LISTS)
def test_h6_update_semantics(a, b):
    h = Headers(a)
    other = Headers(b)
    h.update(other)
    for key in distinct_keys(b):
        assert h.get_list(key) == other.get_list(key)
    for key in distinct_keys(a):
        if key not in distinct_keys(b):
            assert h.get_list(key) == Headers(a).get_list(key)
    # Update with an equal instance is the identity.
    h2 = Headers(a)
    h2.update(Headers(a))
    assert h2 == Headers(a)
    assert h2.multi_items() == Headers(a).multi_items()


# H7 — get_list split_commas


@given(pairs=HEADER_PAIR_LISTS)
def test_h7_get_list_split_commas(pairs):
    h = Headers(pairs)
    for key in distinct_keys(pairs):
        expected = [
            part.strip() for v in h.get_list(key) for part in v.split(",")
        ]
        assert h.get_list(key, split_commas=True) == expected


# H8 — equality semantics


@given(pairs=HEADER_PAIR_LISTS)
def test_h8_equality_semantics(pairs):
    h = Headers(pairs)
    # Order-insensitive.
    assert Headers(list(reversed(pairs))) == h
    # Key-case-insensitive.
    recased = [(k.swapcase(), v) for k, v in pairs]
    assert Headers(recased) == h
    # Duplicate-sensitive: an extra occurrence breaks equality.
    assume(pairs)
    k, v = pairs[0]
    assert Headers(pairs + [(k, v)]) != h
