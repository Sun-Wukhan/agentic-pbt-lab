# Shared Hypothesis strategies for the pypa-packaging PBT lab.
#
# Everything is *structured*: versions are built from PEP 440 parts
# (epoch / release / pre / post / dev / local) and rendered both in a
# canonical spelling and in a "messy" spelling that uses every legal
# alias and separator, so normalisation properties compare two
# renderings of one logical version.  Specifier tests draw spec
# versions and candidate versions from shared part pools and mutate one
# component at a time, so family collisions (same epoch + trimmed
# release -- the case specifiers.py::_fast_match carves out) are dense.
from __future__ import annotations

import dataclasses

from hypothesis import strategies as st

# ---------------------------------------------------------------- versions

SEG = st.integers(min_value=0, max_value=30)
release_tuples = st.lists(SEG, min_size=1, max_size=4).map(tuple)

PRE_SPELLINGS = {
    "a": ["a", "alpha", "A", "ALPHA"],
    "b": ["b", "beta", "B", "BETA"],
    "rc": ["rc", "c", "pre", "preview", "RC", "C", "PRE", "Preview"],
}
SEPS = ["", ".", "-", "_"]


@dataclasses.dataclass(frozen=True)
class VParts:
    epoch: int = 0
    release: tuple = (1, 0)
    pre: tuple | None = None  # (canonical letter, number)
    post: int | None = None
    dev: int | None = None
    local: tuple | None = None  # segments: str (lowercase alnum) or int


local_segments = st.one_of(
    st.integers(min_value=0, max_value=50),
    st.text(alphabet="abcdefghijklmnopqrstuvwxyz0123456789", min_size=1, max_size=5),
)

version_part_records = st.builds(
    VParts,
    epoch=st.integers(min_value=0, max_value=3),
    release=release_tuples,
    pre=st.none() | st.tuples(st.sampled_from(["a", "b", "rc"]), SEG),
    post=st.none() | SEG,
    dev=st.none() | SEG,
    local=st.none() | st.lists(local_segments, min_size=1, max_size=3).map(tuple),
)


def render_canonical(p: VParts) -> str:
    out = ""
    if p.epoch:
        out += f"{p.epoch}!"
    out += ".".join(map(str, p.release))
    if p.pre is not None:
        out += f"{p.pre[0]}{p.pre[1]}"
    if p.post is not None:
        out += f".post{p.post}"
    if p.dev is not None:
        out += f".dev{p.dev}"
    if p.local is not None:
        out += "+" + ".".join(str(x) for x in p.local)
    return out


@st.composite
def render_messy(draw, p: VParts) -> str:
    """A second, legal spelling of the same logical version."""
    out = ""
    if draw(st.booleans()):
        out += "v"
    if p.epoch:
        out += f"{p.epoch}!"
    release = list(p.release)
    if draw(st.booleans()) and release:
        release[0] = f"{release[0]:02d}"  # leading zero is legal: 01 == 1
    out += ".".join(map(str, release))
    if p.pre is not None:
        letter, n = p.pre
        out += draw(st.sampled_from(SEPS))
        out += draw(st.sampled_from(PRE_SPELLINGS[letter]))
        out += draw(st.sampled_from(SEPS))
        if n != 0 or draw(st.booleans()):
            out += str(n)  # omitting the number means an implicit 0
    if p.post is not None:
        form = draw(
            st.sampled_from(
                [
                    ".post{n}",
                    "-{n}",
                    ".rev{n}",
                    "_r{n}",
                    "post{n}",
                    ".post-{n}",
                    "-post{n}",
                ]
            )
        )
        out += form.format(n=p.post)
    if p.dev is not None:
        form = draw(st.sampled_from([".dev{n}", "dev{n}", "-dev{n}", "_dev_{n}"]))
        out += form.format(n=p.dev)
    if p.local is not None:
        segs = []
        for seg in p.local:
            if isinstance(seg, int):
                segs.append(draw(st.sampled_from([str(seg), f"{seg:02d}"])))
            else:
                segs.append(draw(st.sampled_from([seg, seg.upper()])))
        sep = draw(st.sampled_from([".", "-", "_"]))
        out += "+" + sep.join(segs)
    return out


@st.composite
def versions(draw) -> "object":
    """A Version built from structured parts (canonical spelling)."""
    from packaging.version import Version

    return Version(render_canonical(draw(version_part_records)))


@st.composite
def related_parts(draw, base: VParts) -> VParts:
    """Mutate one component of *base* -- dense coverage of the boundary
    cases around a given version (its own family, neighbours, locals)."""
    parts = draw(version_part_records) if draw(st.integers(0, 4)) == 0 else base
    choice = draw(st.integers(0, 8))
    if choice == 0:  # bump / drop a release segment
        rel = list(parts.release)
        i = draw(st.integers(0, len(rel) - 1))
        rel[i] = max(0, rel[i] + draw(st.sampled_from([-1, 1])))
        parts = dataclasses.replace(parts, release=tuple(rel))
    elif choice == 1 and len(parts.release) > 1:
        parts = dataclasses.replace(parts, release=parts.release[:-1])
    elif choice == 2:
        parts = dataclasses.replace(parts, release=parts.release + (draw(SEG),))
    elif choice == 3:
        parts = dataclasses.replace(
            parts, pre=draw(st.none() | st.tuples(st.sampled_from(["a", "b", "rc"]), SEG))
        )
    elif choice == 4:
        parts = dataclasses.replace(parts, post=draw(st.none() | SEG))
    elif choice == 5:
        parts = dataclasses.replace(parts, dev=draw(st.none() | SEG))
    elif choice == 6:
        parts = dataclasses.replace(
            parts, local=draw(st.none() | st.lists(local_segments, min_size=1, max_size=2).map(tuple))
        )
    elif choice == 7:
        parts = dataclasses.replace(parts, epoch=draw(st.integers(0, 3)))
    else:  # trailing zero on the release: same version, different spelling
        parts = dataclasses.replace(parts, release=parts.release + (0,))
    return parts


# -------------------------------------------------------------- specifiers

PLAIN_OPS = [">=", "<=", ">", "<"]  # no local allowed in the spec version
EQ_OPS = ["==", "!="]  # local and .* allowed
ALL_OPS = PLAIN_OPS + EQ_OPS + ["~=", "==="]


@st.composite
def specifier_builds(draw, *, allow_arbitrary: bool = True):
    """One syntactically valid specifier, as (text, operator, base parts).

    The base parts are the VParts the specifier was built from (for a
    wildcard, the prefix parts), so tests can generate candidate
    versions densely around the specifier's own family.
    """
    ops = ALL_OPS if allow_arbitrary else ALL_OPS[:-1]
    op = draw(st.sampled_from(ops))
    parts = draw(version_part_records)
    if op in PLAIN_OPS or op == "~=":
        parts = dataclasses.replace(parts, local=None)
    if op == "~=" and len(parts.release) < 2:
        parts = dataclasses.replace(parts, release=parts.release + (1,))
    if op in EQ_OPS and draw(st.booleans()):
        # Wildcard form: epoch + release prefix + .*, nothing else.
        prefix = parts.release[: draw(st.integers(1, len(parts.release)))]
        parts = dataclasses.replace(
            parts, release=prefix, pre=None, post=None, dev=None, local=None
        )
        base = f"{parts.epoch}!" if parts.epoch else ""
        return f"{op}{base}{'.'.join(map(str, prefix))}.*", op, parts
    if op == "===":
        text = draw(
            st.sampled_from(
                [
                    render_canonical(parts),
                    render_canonical(parts).upper(),
                    "1.0",
                    "1.0.0",
                    "wat",
                    "1.0a1",
                ]
            )
        )
        return f"==={text}", op, parts
    gap = draw(st.sampled_from(["", " "]))
    return f"{op}{gap}{render_canonical(parts)}", op, parts


@st.composite
def specifier_strings(draw, *, allow_arbitrary: bool = True) -> str:
    """One syntactically valid specifier string."""
    text, _op, _parts = draw(specifier_builds(allow_arbitrary=allow_arbitrary))
    return text


@st.composite
def specifier_set_strings(draw, *, allow_arbitrary: bool = True, max_specs: int = 3) -> str:
    n = draw(st.integers(0, max_specs))
    specs = [draw(specifier_strings(allow_arbitrary=allow_arbitrary)) for _ in range(n)]
    return ",".join(specs)


# ------------------------------------------------------------------- names

name_segment = st.text(alphabet="abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", min_size=1, max_size=6)
project_names = st.lists(name_segment, min_size=1, max_size=3).map(
    lambda segs: "-".join(segs)
)


@st.composite
def messy_names(draw, name: str) -> str:
    """Same project, different legal spelling (case + separator runs)."""
    out = []
    for ch in name:
        if ch == "-":
            out.append(draw(st.sampled_from(["-", "_", ".", "--", "-_", "._"])))
        elif ch.isalpha():
            out.append(draw(st.sampled_from([ch.lower(), ch.upper()])))
        else:
            out.append(ch)
    return "".join(out)


# ----------------------------------------------------------------- markers

VERSION_KEYS = ["python_version", "python_full_version", "implementation_version", "platform_release"]
STRING_KEYS = ["os_name", "sys_platform", "platform_machine", "platform_system", "implementation_name", "platform_python_implementation"]
STRING_VALUES = {
    "os_name": ["posix", "nt", "java"],
    "sys_platform": ["linux", "win32", "darwin"],
    "platform_machine": ["x86_64", "arm64", "AMD64"],
    "platform_system": ["Linux", "Windows", "Darwin"],
    "implementation_name": ["cpython", "pypy"],
    "platform_python_implementation": ["CPython", "PyPy"],
}
SHORT_VERSIONS = ["3.8", "3.9", "3.10", "3.11", "3.12", "3.13", "3.14"]
FULL_VERSIONS = ["3.9.0", "3.10.4", "3.11.0", "3.12.3", "3.13.0a2", "7.3.13"]


@st.composite
def marker_atoms(draw) -> str:
    kind = draw(st.sampled_from(["version", "string", "membership", "extra"]))
    if kind == "version":
        key = draw(st.sampled_from(VERSION_KEYS))
        value = draw(st.sampled_from(SHORT_VERSIONS + FULL_VERSIONS))
        op = draw(st.sampled_from(["<", "<=", "==", "!=", ">=", ">", "~="]))
        if op == "~=" and value.count(".") < 1:
            value += ".0"
        if op == "==" and draw(st.booleans()):
            prefix = value.rsplit(".", 1)[0]
            return f'{key} == "{prefix}.*"'
        return f'{key} {op} "{value}"'
    if kind == "string":
        key = draw(st.sampled_from(STRING_KEYS))
        value = draw(st.sampled_from(STRING_VALUES[key] + ["other", ""]))
        op = draw(st.sampled_from(["==", "!="]))
        return f'{key} {op} "{value}"'
    if kind == "membership":
        key = draw(st.sampled_from(STRING_KEYS))
        needle = draw(st.sampled_from(["lin", "win", "64", "posix", "x", "CP", ""]))
        op = draw(st.sampled_from(["in", "not in"]))
        return f'"{needle}" {op} {key}'
    value = draw(st.sampled_from(["foo", "Foo_Bar", "foo-bar", "baz.qux", ""]))
    op = draw(st.sampled_from(["==", "!="]))
    return f'extra {op} "{value}"'


@st.composite
def marker_strings(draw) -> str:
    """and-groups joined by or, one group optionally parenthesised."""
    groups = []
    for _ in range(draw(st.integers(1, 3))):
        atoms = [draw(marker_atoms()) for _ in range(draw(st.integers(1, 2)))]
        groups.append(" and ".join(atoms))
    if len(groups) > 1 and draw(st.booleans()):
        i = draw(st.integers(0, len(groups) - 1))
        groups[i] = f"({groups[i]})"
    return " or ".join(groups)


@st.composite
def marker_environments(draw) -> dict:
    env = {}
    for key in STRING_KEYS:
        env[key] = draw(st.sampled_from(STRING_VALUES[key] + ["other"]))
    env["python_version"] = draw(st.sampled_from(SHORT_VERSIONS))
    env["python_full_version"] = draw(st.sampled_from(FULL_VERSIONS + SHORT_VERSIONS))
    env["implementation_version"] = draw(st.sampled_from(FULL_VERSIONS))
    env["platform_release"] = draw(st.sampled_from(["5.15.0", "6.1.0", "10.0.22621", "23.0.0"]))
    env["platform_version"] = draw(st.sampled_from(["#1 SMP", "10.0.22621", ""]))
    env["extra"] = draw(st.sampled_from(["", "foo", "Foo_Bar", "foo-bar", "baz"]))
    return env


# ------------------------------------------------------------ requirements

EXTRA_NAMES = ["foo", "Foo_Bar", "baz.qux", "test", "ALL"]


@st.composite
def requirement_strings(draw) -> str:
    name = draw(messy_names(draw(project_names)))
    out = name
    extras = draw(st.lists(st.sampled_from(EXTRA_NAMES), min_size=0, max_size=2, unique=True))
    if extras:
        out += "[" + ",".join(extras) + "]"
    use_url = draw(st.booleans()) and draw(st.booleans())
    if use_url:
        out += " @ https://example.com/pkg-1.0.tar.gz"
    else:
        # Specifiers for a *requirement*: === is only generated in last
        # position.  A === specifier followed by another specifier that
        # contains a space is PEP 508-valid but the pinned parser
        # rejects it -- that is Finding 001, so the class is excluded
        # here (it has its own tests in test_finding_001.py) to let the
        # round-trip properties run at full budget.
        pieces = [
            draw(specifier_strings(allow_arbitrary=False))
            for _ in range(draw(st.integers(0, 2)))
        ]
        if draw(st.booleans()):
            text, op, _parts = draw(specifier_builds())
            if op == "===":
                pieces.append(text)
        specs = ",".join(pieces)
        if specs:
            out += " " + specs if draw(st.booleans()) else specs
    if draw(st.booleans()):
        out += " ; " + draw(marker_strings())
    return out
