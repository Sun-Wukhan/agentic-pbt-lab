# No-crash property N1 (see ../PROPERTIES.md): the four parsers either
# accept the input or raise exactly their documented exception type.
from hypothesis import given
from hypothesis import strategies as st

from packaging.markers import InvalidMarker, Marker
from packaging.requirements import InvalidRequirement, Requirement
from packaging.specifiers import InvalidSpecifier, Specifier, SpecifierSet
from packaging.version import InvalidVersion, Version

FRAGMENTS = [
    "1.0", "1!2.0a1", ">=2.0", "<3", "==", "!=", "~=", "===", ".*", "+local",
    "foo", "Foo_Bar", "[extra]", ";", "and", "or", "(", ")", '"3.10"',
    "python_version", "extra", "@", "https://x.co/a", ",", " ", ".", "-",
    "_", "dev", "a1", "post", "rc2", "in", "not", "0", "999", "..", "v1",
]

texts = st.one_of(
    st.text(max_size=40),
    st.lists(st.sampled_from(FRAGMENTS), min_size=1, max_size=6).map(" ".join),
    st.lists(st.sampled_from(FRAGMENTS), min_size=1, max_size=6).map("".join),
)


@given(s=texts)
def test_n1_parsers_raise_only_documented(s):
    try:
        Version(s)
    except InvalidVersion:
        pass
    try:
        Specifier(s)
    except InvalidSpecifier:
        pass
    try:
        SpecifierSet(s)
    except InvalidSpecifier:
        pass
    try:
        Requirement(s)
    except InvalidRequirement:
        pass
    try:
        Marker(s)
    except InvalidMarker:
        pass
