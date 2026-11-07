#!/usr/bin/env bash
# Reproduce the pypa/packaging property-based testing runs from the
# Agentic PBT Lab.
#
#   ./run.sh [EXAMPLES]
#
# packaging is tested from a shallow clone of the upstream repo
# (./upstream, pinned at 713d0d78924ab2686c513dde058ce6aafc63e766):
# the clone is installed into this target's own virtualenv (./.venv),
# so the code under test is byte-for-byte the pinned commit's
# src/packaging/.  Nothing is patched.
#
# EXAMPLES is the Hypothesis profile size per property: 500 (default
# profile) or 2000 (long profile).
#
# Expected result: every test_pbt_* property passes at both profiles.
# The 9 tests in harness/test_finding_001.py and
# harness/test_finding_002.py FAIL by design -- they encode the
# behaviour PEP 508 / the Version round-trip contract requires, and
# are the executable form of Findings 001 and 002 in ./findings.
# The pytest exit code reflects those known failures.
set -euo pipefail

PINNED_COMMIT="713d0d78924ab2686c513dde058ce6aafc63e766"
HYPOTHESIS_VERSION="6.170.15"
PYTEST_VERSION="9.2.0"

EXAMPLES="${1:-500}"
case "$EXAMPLES" in
  500)  PROFILE=default ;;
  2000) PROFILE=long ;;
  *) echo "usage: ./run.sh [500|2000]" >&2; exit 2 ;;
esac

HERE="$(cd "$(dirname "$0")" && pwd)"
VENV="$HERE/.venv"
cd "$HERE"

if [ ! -d upstream/.git ]; then
  echo ">> cloning pypa/packaging (shallow) into ./upstream"
  git clone --depth 1 https://github.com/pypa/packaging upstream
fi
ACTUAL_COMMIT="$(git -C upstream rev-parse HEAD)"
if [ "$ACTUAL_COMMIT" != "$PINNED_COMMIT" ]; then
  echo ">> WARNING: upstream is at $ACTUAL_COMMIT, runs were recorded against $PINNED_COMMIT" >&2
fi

if [ ! -x "$VENV/bin/python" ]; then
  echo ">> creating virtualenv at $VENV"
  python3 -m venv "$VENV"
fi

echo ">> installing pinned packages (hypothesis==$HYPOTHESIS_VERSION pytest==$PYTEST_VERSION) and the cloned packaging"
"$VENV/bin/pip" install --quiet \
  "hypothesis==$HYPOTHESIS_VERSION" "pytest==$PYTEST_VERSION" ./upstream

echo ">> packaging under test: $("$VENV/bin/python" -c 'import packaging; print(packaging.__version__)')"
echo ">> running property tests (profile=$PROFILE, $EXAMPLES examples per property)"
HYPOTHESIS_PROFILE="$PROFILE" "$VENV/bin/python" -m pytest harness/ -q -p no:cacheprovider
