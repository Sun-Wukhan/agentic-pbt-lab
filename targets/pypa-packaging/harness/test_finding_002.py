# Finding 002 -- desired behaviour, currently FAILING on the pinned
# build (see ../findings/002-from-parts-bool-parts.md).  The validators
# should reject bool parts with InvalidVersion instead of manufacturing
# versions whose str() cannot be re-parsed.
import pytest

from packaging.version import InvalidVersion, Version


def test_finding_002_bool_epoch_rejected():
    with pytest.raises(InvalidVersion):
        Version.from_parts(epoch=True, release=(1,))


def test_finding_002_bool_release_rejected():
    with pytest.raises(InvalidVersion):
        Version.from_parts(release=(True, 2))


def test_finding_002_bool_pre_number_rejected():
    with pytest.raises(InvalidVersion):
        Version("1.2").__replace__(pre=("a", True))
