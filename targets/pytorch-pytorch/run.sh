#!/usr/bin/env bash
# Reproduce the PyTorch property-based testing runs from the Agentic PBT Lab.
#
#   ./run.sh [default|long] [pytest args...]
#
# Runs the Hypothesis harness in ./harness against the pinned torch build
# in the lab venv. Profiles: default = 200 examples per property,
# long = 2,000 (see harness/conftest.py).
#
# The target is the installed library, not a clone: PyTorch is exercised
# exactly as users import it. run.sh verifies the pinned version before
# running; set PYTORCH_PBT_VENV to point at a different virtualenv.
set -euo pipefail

PIN="2.15.0+cpu"
PROFILE="${1:-default}"
if [ $# -gt 0 ]; then shift; fi
HERE="$(cd "$(dirname "$0")" && pwd)"
VENV="${PYTORCH_PBT_VENV:-$HOME/workspace/.venv-torch}"
PY="$VENV/bin/python"

if [ ! -x "$PY" ]; then
  echo "error: no python at $PY (set PYTORCH_PBT_VENV)" >&2
  exit 1
fi

echo ">> verifying pinned torch==$PIN in $VENV"
"$PY" - "$PIN" <<'EOF'
import sys, torch, hypothesis, numpy
pin = sys.argv[1]
assert torch.__version__ == pin, f"torch {torch.__version__} != pinned {pin}"
print(f"   torch {torch.__version__}, hypothesis {hypothesis.__version__}, "
      f"numpy {numpy.__version__}")
EOF

cd "$HERE/harness"
echo ">> running property tests (HYPOTHESIS_PROFILE=$PROFILE)"
HYPOTHESIS_PROFILE="$PROFILE" "$PY" -m pytest -q -p no:cacheprovider "$@" .
