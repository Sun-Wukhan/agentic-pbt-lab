# URL properties for the encode-httpx PBT lab. See ../PROPERTIES.md (U1-U11).
from __future__ import annotations

import pytest
from hypothesis import assume, given
from hypothesis import strategies as st

import httpx
from httpx import URL

from pbt_strategies import (
    ASCII_LABEL,
    PUNY_LABEL,
    build_url_string,
    mutated_url_strings,
    url_components,
)

DEFAULT_PORTS = {"ftp": 21, "http": 80, "https": 443, "ws": 80, "wss": 443}


def _build(c) -> URL:
    return URL(build_url_string(c))


# ---------------------------------------------------------------------------
# U1 / U2 — canonical form fixpoints
# ---------------------------------------------------------------------------


@given(c=url_components())
def test_u1_parse_print_fixpoint(c):
    try:
        u = _build(c)
    except httpx.InvalidURL:
        return
    assert str(URL(str(u))) == str(u)
    assert URL(str(u)) == u


@given(s=mutated_url_strings())
def test_u2_mutation_idempotence_and_no_crash(s):
    try:
        u = URL(s)
    except httpx.InvalidURL:
        return
    # Any other exception type escaping the constructor is a failure here.
    assert str(URL(str(u))) == str(u)


# ---------------------------------------------------------------------------
# U3 — case and default-port normalization
# ---------------------------------------------------------------------------


@given(
    scheme=st.sampled_from(["http", "https", "ws", "wss", "ftp"]),
    host=st.lists(ASCII_LABEL, min_size=1, max_size=2).map(".".join),
    use_default_port=st.booleans(),
    upper=st.booleans(),
)
def test_u3_case_and_default_port_normalization(scheme, host, use_default_port, upper):
    port = DEFAULT_PORTS[scheme] if use_default_port else None
    shown_scheme = scheme.upper() if upper else scheme
    shown_host = host.upper() if upper else host
    with_port = (
        f"{shown_scheme}://{shown_host}:{port}/a" if port else f"{shown_scheme}://{shown_host}/a"
    )
    without_port = f"{scheme}://{host}/a"
    u = URL(with_port)
    assert u.scheme == scheme
    assert u.host == host
    assert u.port is None
    assert u == URL(without_port)
    assert hash(u) == hash(URL(without_port))


# ---------------------------------------------------------------------------
# U4 / U5 — IDNA host contract and host-access safety
# ---------------------------------------------------------------------------

UNICODE_HOST_LABEL = st.text(alphabet="中国ñüßéあ漢字α", min_size=1, max_size=5)


@given(
    labels=st.lists(
        st.one_of(UNICODE_HOST_LABEL, ASCII_LABEL), min_size=2, max_size=3
    )
)
def test_u4_idna_host_any_label_position(labels):
    assume(any(any(ord(ch) > 127 for ch in label) for label in labels))
    unicode_host = ".".join(labels)
    import idna

    puny_host = idna.encode(unicode_host).decode("ascii")
    for spelling in (unicode_host, puny_host):
        u = URL(f"http://{spelling}/")
        assert u.host == unicode_host
        assert u.raw_host == puny_host.encode("ascii")
    assert URL(f"http://{unicode_host}/") == URL(f"http://{puny_host}/")


@given(
    labels=st.lists(st.one_of(ASCII_LABEL, PUNY_LABEL), min_size=1, max_size=3)
)
def test_u5_host_access_never_leaks_idna_errors(labels):
    host = ".".join(labels)
    try:
        u = URL(f"http://{host}/")
    except httpx.InvalidURL:
        return
    try:
        value = u.host
    except httpx.InvalidURL:
        return
    # Any idna/Unicode exception escaping here fails the test.
    assert isinstance(value, str)


# ---------------------------------------------------------------------------
# U6 — dot-segment removal vs RFC 3986 section 5.2.4
# ---------------------------------------------------------------------------


def remove_dot_segments(path: str) -> str:
    """Faithful transcription of the RFC 3986 5.2.4 algorithm."""
    output = ""
    buf = path
    while buf:
        if buf.startswith("../"):
            buf = buf[3:]
        elif buf.startswith("./"):
            buf = buf[2:]
        elif buf.startswith("/./"):
            buf = "/" + buf[3:]
        elif buf == "/.":
            buf = "/"
        elif buf.startswith("/../"):
            buf = "/" + buf[4:]
            output = output[: output.rfind("/")] if "/" in output else ""
        elif buf == "/..":
            buf = "/"
            output = output[: output.rfind("/")] if "/" in output else ""
        elif buf in (".", ".."):
            buf = ""
        else:
            idx = buf.find("/", 1) if buf.startswith("/") else buf.find("/")
            if idx == -1:
                output += buf
                buf = ""
            else:
                output += buf[:idx]
                buf = buf[idx:]
    return output


DOT_SEGMENTS = st.sampled_from(["a", "b", ".", "..", "", "x-y", "seg_1"])


@given(segments=st.lists(DOT_SEGMENTS, min_size=0, max_size=5))
def test_u6_dot_segments_match_rfc3986(segments):
    path = "/" + "/".join(segments)
    u = URL(f"http://e.org{path}")
    assert u._uri_reference.path == remove_dot_segments(path)


# ---------------------------------------------------------------------------
# U7 — copy_with component fidelity
# ---------------------------------------------------------------------------


@given(c=url_components())
def test_u7_copy_with_component_fidelity(c):
    try:
        u = _build(c)
    except httpx.InvalidURL:
        return
    assume(u.is_absolute_url)
    assert u.copy_with() == u

    changed = u.copy_with(path="/other")
    assert changed._uri_reference.path == "/other"
    for field in ("scheme", "userinfo", "host", "port", "query", "fragment"):
        assert getattr(changed._uri_reference, field) == getattr(
            u._uri_reference, field
        )

    changed = u.copy_with(fragment="newfrag")
    assert changed._uri_reference.fragment == "newfrag"
    for field in ("scheme", "userinfo", "host", "port", "path", "query"):
        assert getattr(changed._uri_reference, field) == getattr(
            u._uri_reference, field
        )

    changed = u.copy_with(query=None)
    assert changed._uri_reference.query is None
    assert changed._uri_reference.path == u._uri_reference.path


# ---------------------------------------------------------------------------
# U8 — join semantics
# ---------------------------------------------------------------------------


@given(c=url_components(), frag=st.text(alphabet="abc", min_size=1, max_size=4))
def test_u8_join_metamorphic(c, frag):
    try:
        base = _build(c)
    except httpx.InvalidURL:
        return
    assume(base.is_absolute_url)
    # urljoin only resolves references for schemes in its uses_relative /
    # uses_netloc tables; for unknown schemes it returns the reference
    # unresolved. httpx is an HTTP client, so the metamorphic contract is
    # stated for the web schemes (Finding 007). Mixed-case schemes are
    # also excluded: they trip Finding 001 (the port survives in the base
    # but not in the re-parsed join result), which U3 owns.
    assume(c["scheme"] in ("http", "https", "ws", "wss", "ftp"))

    # An absolute reference replaces the base entirely (modulo normalization).
    other = URL("http://other.example/x?y#z")
    assert base.join(other) == other

    # Empty reference: urljoin returns the base unchanged, fragment
    # included — a deviation from RFC 3986 (which drops the fragment),
    # inherited by delegation and recorded in Finding 007. The property
    # encodes the actual delegation contract.
    assert base.join("") == base

    # Fragment-only reference changes only the fragment.
    # Carve-out (Finding 007): with a present-but-empty query ("?"),
    # urllib.parse.urljoin drops the "?" instead of inheriting the
    # (empty) query, so the query component is only checked otherwise.
    joined = base.join(f"#{frag}")
    if base._uri_reference.query == "":
        assert joined._uri_reference.query is None
        joined = joined.copy_with(query=b"")
    else:
        assert joined._uri_reference.query == base._uri_reference.query
    assert joined.copy_with(fragment=None) == base.copy_with(fragment=None)
    assert joined._uri_reference.fragment == frag

    # Query-only reference changes only query and fragment.
    joined = base.join("?newq=1")
    assert joined._uri_reference.query == "newq=1"
    assert joined._uri_reference.fragment is None
    for field in ("scheme", "userinfo", "host", "port", "path"):
        assert getattr(joined._uri_reference, field) == getattr(
            base._uri_reference, field
        )


RFC_BASE = "http://a/b/c/d;p?q"
RFC_CASES = [
    # RFC 3986 section 5.4.1, normal examples.
    ("g:h", "g:h"),
    ("g", "http://a/b/c/g"),
    ("./g", "http://a/b/c/g"),
    ("g/", "http://a/b/c/g/"),
    ("/g", "http://a/g"),
    ("//g", "http://g"),
    ("?y", "http://a/b/c/d;p?y"),
    ("g?y", "http://a/b/c/g?y"),
    ("#s", "http://a/b/c/d;p?q#s"),
    ("g#s", "http://a/b/c/g#s"),
    ("g?y#s", "http://a/b/c/g?y#s"),
    (";x", "http://a/b/c/;x"),
    ("g;x", "http://a/b/c/g;x"),
    ("g;x?y#s", "http://a/b/c/g;x?y#s"),
    ("", "http://a/b/c/d;p?q"),
    (".", "http://a/b/c/"),
    ("./", "http://a/b/c/"),
    ("..", "http://a/b/"),
    ("../", "http://a/b/"),
    ("../g", "http://a/b/g"),
    ("../..", "http://a/"),
    ("../../", "http://a/"),
    ("../../g", "http://a/g"),
    # RFC 3986 section 5.4.2, abnormal examples.
    ("../../../g", "http://a/g"),
    ("../../../../g", "http://a/g"),
    ("/./g", "http://a/g"),
    ("/../g", "http://a/g"),
    ("g.", "http://a/b/c/g."),
    (".g", "http://a/b/c/.g"),
    ("g..", "http://a/b/c/g.."),
    ("..g", "http://a/b/c/..g"),
    ("./../g", "http://a/b/g"),
    ("./g/.", "http://a/b/c/g/"),
    ("g/./h", "http://a/b/c/g/h"),
    ("g/../h", "http://a/b/c/h"),
    ("g;x=1/./y", "http://a/b/c/g;x=1/y"),
    ("g;x=1/../y", "http://a/b/c/y"),
]


@pytest.mark.parametrize("rel,expected", RFC_CASES)
def test_u8_join_rfc3986_examples(rel, expected):
    assert str(URL(RFC_BASE).join(rel)) == expected


# ---------------------------------------------------------------------------
# U9 — percent-encoding round-trips
# ---------------------------------------------------------------------------


@given(
    segments=st.lists(
        st.sampled_from(["a b", "üñï", "x;y", "p=q", "tilde~", "plain"]),
        min_size=1,
        max_size=3,
    ),
    frag=st.text(alphabet="ab ü", min_size=0, max_size=5),
)
def test_u9_percent_encoding_round_trip(segments, frag):
    path = "/" + "/".join(segments)
    u = URL("http://e.org", path=path, fragment=frag)
    assert u.path == path
    assert u.fragment == frag
    reparsed = URL(str(u))
    assert reparsed.path == path
    assert reparsed.fragment == frag


# ---------------------------------------------------------------------------
# U10 — empty vs absent query
# ---------------------------------------------------------------------------


def test_u10_empty_vs_absent_query():
    with_q = URL("http://e.org/?")
    without_q = URL("http://e.org/")
    assert str(with_q) == "http://e.org/?"
    assert str(without_q) == "http://e.org/"
    assert with_q != without_q
    assert with_q.raw_path == b"/?"
    assert without_q.raw_path == b"/"
    assert with_q.copy_with(query=None) == without_q
    assert URL("http://e.org/#")._uri_reference.fragment == ""
    assert URL("http://e.org/")._uri_reference.fragment is None


# ---------------------------------------------------------------------------
# U11 — equality / hash consistency
# ---------------------------------------------------------------------------


@given(c=url_components())
def test_u11_equality_and_hash_consistency(c):
    try:
        u = _build(c)
    except httpx.InvalidURL:
        return
    assert u == str(u)
    assert u == URL(u)
    assert hash(u) == hash(URL(str(u)))
    assert hash(u) == hash(str(u))
