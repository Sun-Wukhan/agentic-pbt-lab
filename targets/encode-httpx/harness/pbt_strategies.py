# Structured strategies for the encode-httpx PBT lab.
#
# URLs are built from components (so the generator spends its budget on
# combinations httpx claims to normalize, not on random byte soup), and a
# mutation strategy derives near-valid URL strings from valid ones.
# QueryParams and Headers get delimiter-heavy alphabets, because that is
# where parse/serialize asymmetries would live.
from __future__ import annotations

import string

from hypothesis import strategies as st

# ---------------------------------------------------------------------------
# URL components
# ---------------------------------------------------------------------------

SCHEMES = ["http", "https", "ws", "wss", "ftp", "HTTP", "Https", "a+b-c.d"]

ASCII_LABEL = st.text(
    alphabet="abcdefghijklmnopqrstuvwxyz0123456789-",
    min_size=1,
    max_size=10,
).filter(lambda s: not s.startswith("-") and not s.endswith("-"))

UNICODE_LABEL = st.text(
    alphabet="中国ñüßéあ漢字α",
    min_size=1,
    max_size=6,
)

# A pre-encoded punycode label (valid or deliberately malformed).
PUNY_LABEL = st.sampled_from(
    ["xn--fiqs8s", "xn--fa-hia", "xn--nxasmm1c", "xn--a", "xn--", "xn--bcher-kva"]
)

HOST_LABEL = st.one_of(ASCII_LABEL, UNICODE_LABEL, PUNY_LABEL)

HOSTS = st.one_of(
    st.lists(HOST_LABEL, min_size=1, max_size=3).map(".".join),
    st.sampled_from(
        ["127.0.0.1", "0.0.0.0", "[::1]", "[2001:db8::1]", "localhost", ""]
    ),
)

PORTS = st.one_of(
    st.none(),
    st.integers(min_value=0, max_value=65535),
    st.sampled_from([21, 80, 443]),
)

PATH_SEGMENTS = st.sampled_from(
    [
        "a",
        "b",
        "path",
        "seg-ment",
        "un_reserved~1",
        ".",
        "..",
        "",
        "%20",
        "%7E",
        "%7e",
        "%zz",
        "a b",
        "üñï",
        "a;b",
        "a=b",
        "x%2Fy",
    ]
)

QUERY_KEYS = st.text(alphabet="ab=%+&; ", min_size=1, max_size=4)
QUERY_VALS = st.text(alphabet="ab=%+&; ", min_size=0, max_size=4)
QUERY_PAIRS = st.lists(
    st.tuples(
        st.text(alphabet="abc", min_size=1, max_size=3),
        st.text(alphabet="xyz=+&%; ", min_size=0, max_size=4),
    ),
    min_size=0,
    max_size=4,
)

FRAGMENTS = st.text(alphabet="abc #%ü", min_size=0, max_size=6)

USERINFOS = st.one_of(
    st.none(),
    st.tuples(
        st.text(alphabet="user@%: ", min_size=1, max_size=5),
        st.one_of(st.none(), st.text(alphabet="pass@%: ", min_size=1, max_size=5)),
    ),
)


@st.composite
def url_components(draw):
    return {
        "scheme": draw(st.sampled_from(SCHEMES)),
        "host": draw(HOSTS),
        "port": draw(PORTS),
        "userinfo": draw(USERINFOS),
        "path_segments": draw(st.lists(PATH_SEGMENTS, min_size=0, max_size=4)),
        "trailing_slash": draw(st.booleans()),
        "query_pairs": draw(QUERY_PAIRS),
        "query_absent": draw(st.booleans()),
        "fragment": draw(st.one_of(st.none(), FRAGMENTS)),
    }


def build_url_string(c) -> str:
    scheme = c["scheme"]
    out = f"{scheme}://" if scheme else "//"
    if c["userinfo"] is not None:
        user, password = c["userinfo"]
        out += user
        if password is not None:
            out += f":{password}"
        out += "@"
    out += c["host"]
    if c["port"] is not None:
        out += f":{c['port']}"
    path = "/" + "/".join(c["path_segments"])
    if c["trailing_slash"] and not path.endswith("/"):
        path += "/"
    out += path
    if not c["query_absent"]:
        out += "?" + "&".join(f"{k}={v}" for k, v in c["query_pairs"])
    if c["fragment"] is not None:
        out += f"#{c['fragment']}"
    return out


MUTATION_CHARS = "%?#@:./&=+ü~ "


@st.composite
def mutated_url_strings(draw):
    """Take a valid component-built URL string and apply 1-3 small edits."""
    base = build_url_string(draw(url_components()))
    s = list(base)
    for _ in range(draw(st.integers(min_value=1, max_value=3))):
        if not s:
            break
        op = draw(st.sampled_from(["insert", "replace", "delete"]))
        pos = draw(st.integers(min_value=0, max_value=len(s) - 1))
        if op == "insert":
            s.insert(pos, draw(st.sampled_from(list(MUTATION_CHARS))))
        elif op == "replace":
            s[pos] = draw(st.sampled_from(list(MUTATION_CHARS)))
        else:
            del s[pos]
    return "".join(s)


# ---------------------------------------------------------------------------
# QueryParams
# ---------------------------------------------------------------------------

QP_KEYS = st.text(alphabet="abc", min_size=1, max_size=3)
QP_VALS = st.text(alphabet="xyz=+&%; ", min_size=0, max_size=4)
QP_PAIR_LISTS = st.lists(
    st.tuples(QP_KEYS, QP_VALS), min_size=0, max_size=5
)
# Raw query strings over a delimiter-heavy alphabet (no '#': that starts a
# fragment and is not part of a query string handed to QueryParams... except
# when it is; parse_qs treats it as data, so include it via the raw alphabet
# below minus '#' to keep the strategy focused on & = + % ; asymmetries).
QP_RAW_STRINGS = st.text(alphabet="ab=&+%; ", min_size=0, max_size=12)

# ---------------------------------------------------------------------------
# Headers
# ---------------------------------------------------------------------------

HEADER_KEY_BASES = st.sampled_from(
    [
        "Content-Type",
        "X-Custom-Header",
        "Accept",
        "Set-Cookie",
        "ETag",
        "a",
        "x-a",
    ]
)


@st.composite
def header_keys(draw):
    base = draw(HEADER_KEY_BASES)
    # Random case permutation.
    return "".join(
        draw(st.sampled_from([ch.lower(), ch.upper()])) if ch.isalpha() else ch
        for ch in base
    )


HEADER_VALS = st.text(
    alphabet=string.ascii_letters + string.digits + ",; =",
    min_size=0,
    max_size=10,
).filter(lambda s: s == s.strip())

HEADER_PAIR_LISTS = st.lists(
    st.tuples(header_keys(), HEADER_VALS), min_size=0, max_size=5
)
