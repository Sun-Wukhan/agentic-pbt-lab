#!/usr/bin/env bash
# Reproduce the encode/httpx property-based testing runs from the Agentic PBT Lab.
#
#   ./run.sh [EXAMPLES]
#
# httpx is tested from a shallow clone of upstream `main`, pinned at
# b5addb64f0161ff6bfe94c124ef76f6a1fba5254 (post-0.28.1 development tree),
# installed into the lab's shared virtualenv. Nothing upstream is patched;
# the harness only exercises the public URL / QueryParams / Headers models.
# All properties are hermetic — no transports, no network test suites.
#
# EXAMPLES is the Hypothesis profile size per property: 500 (default
# profile) or 2000 (long profile). The venv is shared by the lab's Python
# targets at ~/workspace/.venv-pbt; override with PBT_VENV.
#
# Expected result: U1, U2, U3, U4, U5, U6, U7, U11 and H2 FAIL — those
# are Findings 001-006 in ./findings (several share the Finding 001 root
# cause). Every other property passes, including the full RFC 3986
# section 5.4 join table. The pytest exit code reflects the known
# failures.
set -euo pipefail

HTTPX_COMMIT="b5addb64f0161ff6bfe94c124ef76f6a1fba5254"
HYPOTHESIS_VERSION="6.170.14"
PYTEST_VERSION="9.2.0"

EXAMPLES="${1:-500}"
case "$EXAMPLES" in
  500)  PROFILE=default ;;
  2000) PROFILE=long ;;
  *) echo "usage: ./run.sh [500|2000]" >&2; exit 2 ;;
esac

HERE="$(cd "$(dirname "$0")" && pwd)"
VENV="${PBT_VENV:-$HOME/workspace/.venv-pbt}"

if [ ! -x "$VENV/bin/python" ]; then
  echo ">> creating virtualenv at $VENV"
  python3 -m venv "$VENV"
fi

if [ ! -d "$HERE/httpx" ]; then
  echo ">> cloning encode/httpx (shallow) at $HTTPX_COMMIT"
  git clone --depth 1 https://github.com/encode/httpx "$HERE/httpx"
fi
ACTUAL_COMMIT="$(git -C "$HERE/httpx" rev-parse HEAD)"
if [ "$ACTUAL_COMMIT" != "$HTTPX_COMMIT" ]; then
  echo ">> warning: clone is at $ACTUAL_COMMIT, expected $HTTPX_COMMIT" >&2
fi

echo ">> installing pinned packages (hypothesis==$HYPOTHESIS_VERSION pytest==$PYTEST_VERSION) and the httpx clone"
"$VENV/bin/pip" install --quiet \
  "hypothesis==$HYPOTHESIS_VERSION" "pytest==$PYTEST_VERSION" "$HERE/httpx"

echo ">> httpx under test: $("$VENV/bin/python" -c 'import httpx; print(httpx.__version__)') (clone $ACTUAL_COMMIT)"
echo ">> running property tests (profile=$PROFILE, $EXAMPLES examples per property)"
cd "$HERE"
HYPOTHESIS_PROFILE="$PROFILE" "$VENV/bin/python" -m pytest harness/ -q -p no:cacheprovider
