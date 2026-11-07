# QueryParams properties for the encode-httpx PBT lab. See ../PROPERTIES.md (Q1-Q8).
from __future__ import annotations

from hypothesis import given

import httpx
from httpx import QueryParams, URL

from pbt_strategies import QP_PAIR_LISTS, QP_RAW_STRINGS


def coerce(value):
    """Mirror httpx._utils.primitive_value_to_str."""
    if value is True:
        return "true"
    if value is False:
        return "false"
    if value is None:
        return ""
    return str(value)


# Q1 / Q2 — string round-trip and multi-value preservation


@given(pairs=QP_PAIR_LISTS)
def test_q1_str_round_trip(pairs):
    q = QueryParams(pairs)
    assert QueryParams(str(q)) == q


@given(raw=QP_RAW_STRINGS)
def test_q1_str_round_trip_from_raw(raw):
    q = QueryParams(raw)
    assert QueryParams(str(q)) == q


@given(pairs=QP_PAIR_LISTS)
def test_q2_multi_value_preservation(pairs):
    q = QueryParams(pairs)
    rt = QueryParams(str(q))
    assert sorted(rt.multi_items()) == sorted(q.multi_items())
    for key in q.keys():
        assert rt.get_list(key) == q.get_list(key)


# Q3-Q5 — set / add / remove

from hypothesis import strategies as st

KEY = st.sampled_from(["a", "b", "c", "new"])
VAL = st.text(alphabet="xyz=+&%; ", min_size=0, max_size=4)


@given(pairs=QP_PAIR_LISTS, key=KEY, value=VAL)
def test_q3_set_semantics(pairs, key, value):
    q = QueryParams(pairs)
    before = q.multi_items()
    q2 = q.set(key, value)
    assert q.multi_items() == before  # original unchanged
    assert q2.get_list(key) == [value]
    for other in q.keys():
        if other != key:
            assert q2.get_list(other) == q.get_list(other)


@given(pairs=QP_PAIR_LISTS, key=KEY, value=VAL)
def test_q4_add_semantics(pairs, key, value):
    q = QueryParams(pairs)
    before = q.multi_items()
    q2 = q.add(key, value)
    assert q.multi_items() == before
    assert q2.get_list(key) == q.get_list(key) + [value]
    for other in q.keys():
        if other != key:
            assert q2.get_list(other) == q.get_list(other)


@given(pairs=QP_PAIR_LISTS, key=KEY)
def test_q5_remove_semantics(pairs, key):
    q = QueryParams(pairs)
    before = q.multi_items()
    q2 = q.remove(key)
    assert q.multi_items() == before
    assert key not in q2
    assert q2.get_list(key) == []
    for other in q.keys():
        if other != key:
            assert q2.get_list(other) == q.get_list(other)
    # Removing a missing key is the identity.
    assert q.remove("__missing__") == q


# Q6 — merge


@given(a=QP_PAIR_LISTS, b=QP_PAIR_LISTS, c=QP_PAIR_LISTS)
def test_q6_merge_semantics(a, b, c):
    qa, qb, qc = QueryParams(a), QueryParams(b), QueryParams(c)
    merged = qa.merge(qb)
    for key in qb.keys():
        assert merged.get_list(key) == qb.get_list(key)
    for key in qa.keys():
        if key not in qb:
            assert merged.get_list(key) == qa.get_list(key)
    # Associativity under ==, and empty-merge identity.
    assert qa.merge(qb).merge(qc) == qa.merge(qb.merge(qc))
    assert qa.merge(QueryParams("")) == qa
    assert qa.merge({}) == qa


# Q7 — primitive coercion


@given(
    value=st.one_of(
        st.booleans(),
        st.none(),
        st.integers(min_value=-1000, max_value=1000),
        st.text(alphabet="abc", min_size=0, max_size=3),
    )
)
def test_q7_primitive_coercion(value):
    expected = coerce(value)
    assert QueryParams({"k": value}).get("k") == expected
    assert QueryParams([("k", value)]).get("k") == expected
    assert QueryParams("").set("k", value).get("k") == expected
    assert QueryParams("").add("k", value).get("k") == expected
    assert QueryParams({"k": [value, value]}).get_list("k") == [expected, expected]


# Q8 — URL integration


@given(pairs=QP_PAIR_LISTS, key=KEY, value=VAL)
def test_q8_url_params_integration(pairs, key, value):
    q = QueryParams(pairs)
    u = URL("http://e.org/path", params=q)
    assert u.params == q
    assert u.copy_set_param(key, value).params == q.set(key, value)
    assert u.copy_add_param(key, value).params == q.add(key, value)
    assert u.copy_remove_param(key).params == q.remove(key)
    assert u.copy_merge_params({"zz": "1"}).params == q.merge({"zz": "1"})
    # The rest of the URL is untouched by param manipulation.
    assert u.copy_set_param(key, value).path == "/path"
