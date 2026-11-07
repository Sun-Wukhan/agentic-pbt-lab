# Finding 001 -- desired behaviour, currently FAILING on the pinned
# build (see ../findings/001-requirement-triple-equal-swallows-separators.md).
# These tests encode what PEP 508 requires; they will pass when the
# tokenizer stops swallowing separators after a === specifier.
import pytest

from packaging.requirements import Requirement

SPEC_VALID = [
    "x===0,>= 0",
    "x===0, >=0",
    "x===0,!= 1.0",
    "x===0,< 2",
    "x (===0,>= 0)",
    "x===0,>= 0; python_version>'3'",
]


@pytest.mark.parametrize("text", SPEC_VALID)
def test_finding_001_spec_valid_requirements_parse(text):
    req = Requirement(text)
    assert "===0" in str(req.specifier)
