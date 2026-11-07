#!/usr/bin/env bash
# Reproduce the numpy property-based testing runs from the Agentic PBT Lab.
#
#   ./run.sh [EXAMPLES]
#
# NumPy is tested as an installed binary package (the Maaz et al. paper's
# Python-ecosystem setup): there is no upstream clone and nothing is
# patched. This script ensures the pinned virtualenv exists and runs the
# Hypothesis suite in ./harness against it.
#
# EXAMPLES is the Hypothesis profile size per property: 200 (default
# profile) or 2000 (long profile). The venv is shared by the lab's Python
# targets at ~/workspace/.venv-pbt; override with PBT_VENV.
#
# Expected result: 24 of 25 properties pass. Property P3
# (promote_types associativity) FAILS by design — its counterexample is
# Finding 001 in ./findings, triaged as a wrong property (NumPy documents
# promote_types as "symmetric, but rarely associative"). The pytest exit
# code reflects that known failure.
set -euo pipefail

NUMPY_VERSION="2.5.4"
HYPOTHESIS_VERSION="6.170.14"
PYTEST_VERSION="9.2.0"

EXAMPLES="${1:-200}"
case "$EXAMPLES" in
  200)  PROFILE=default ;;
  2000) PROFILE=long ;;
  *) echo "usage: ./run.sh [200|2000]" >&2; exit 2 ;;
esac

HERE="$(cd "$(dirname "$0")" && pwd)"
VENV="${PBT_VENV:-$HOME/workspace/.venv-pbt}"

if [ ! -x "$VENV/bin/python" ]; then
  echo ">> creating virtualenv at $VENV"
  python3 -m venv "$VENV"
fi

echo ">> installing pinned packages (numpy==$NUMPY_VERSION hypothesis==$HYPOTHESIS_VERSION pytest==$PYTEST_VERSION)"
"$VENV/bin/pip" install --quiet \
  "numpy==$NUMPY_VERSION" "hypothesis==$HYPOTHESIS_VERSION" "pytest==$PYTEST_VERSION"

echo ">> numpy under test: $("$VENV/bin/python" -c 'import numpy; print(numpy.__version__)')"
echo ">> running property tests (profile=$PROFILE, $EXAMPLES examples per property)"
cd "$HERE"
HYPOTHESIS_PROFILE="$PROFILE" "$VENV/bin/python" -m pytest harness/ -q -p no:cacheprovider
