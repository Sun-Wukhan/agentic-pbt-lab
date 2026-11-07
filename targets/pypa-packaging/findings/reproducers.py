# Plain-Python reproducers for the pypa-packaging findings.
# No Hypothesis: each function demonstrates one triaged counterexample
# against the pinned packaging build (26.4.dev0 @ 713d0d78).
#
# Run with the target venv:  ../.venv/bin/python reproducers.py
from packaging.requirements import InvalidRequirement, Requirement
from packaging.specifiers import SpecifierSet

print("packaging under test:", __import__("packaging").__version__)


def finding_001_requirement_triple_equal() -> None:
    """Requirement() rejects spec-valid strings when a === specifier is
    followed by another specifier that contains a space."""
    print("\n--- Finding 001: Requirement parser vs === specifiers ---")
    cases = [
        # (requirement string, PEP 508-valid?)
        ("x===0,>=0", True),  # parses -- but only by accident (see finding)
        ("x===0,>= 0", True),  # REJECTED (bug)
        ("x===0, >=0", True),  # REJECTED (bug)
        ("x===0,!= 1.0", True),  # REJECTED (bug)
        ("x===0,< 2", True),  # REJECTED (bug)
        ("x (===0,>= 0)", True),  # REJECTED (bug)
        ("x===0,>= 0; python_version>'3'", True),  # REJECTED (bug)
        ("x==0,>= 0", True),  # parses: only === triggers it
        ("x>=0,===0", True),  # parses: === in last position is fine
    ]
    for text, _valid in cases:
        try:
            req = Requirement(text)
        except InvalidRequirement as exc:
            print(f"  InvalidRequirement  {text!r}: {str(exc).splitlines()[0]}")
        else:
            print(f"  parsed              {text!r} -> specifier={str(req.specifier)!r}")
    # The same specifier list is accepted by SpecifierSet directly:
    print("  SpecifierSet('===0,>= 0') ->", SpecifierSet("===0,>= 0"))


def finding_002_from_parts_bool() -> None:
    """Version.from_parts / __replace__ validators accept bool (an int
    subclass) and manufacture Versions whose str() cannot be re-parsed."""
    print("\n--- Finding 002: bool parts pass the from_parts validators ---")
    from packaging.version import InvalidVersion, Version

    v = Version.from_parts(epoch=True, release=(1,))
    print(f"  from_parts(epoch=True, release=(1,)): epoch={v.epoch!r} str={str(v)!r}")
    try:
        Version(str(v))
        print("  reparse: OK")
    except InvalidVersion as exc:
        print(f"  reparse: InvalidVersion ({exc})")
    v2 = Version.from_parts(release=(True, 2))
    print(
        f"  from_parts(release=(True, 2)): str={str(v2)!r} "
        f"== Version('1.2'): {v2 == Version('1.2')}"
    )
    v3 = Version("1.2").__replace__(pre=("alpha", True))
    print(f"  __replace__(pre=('alpha', True)): str={str(v3)!r}")


if __name__ == "__main__":
    finding_001_requirement_triple_equal()
    finding_002_from_parts_bool()
