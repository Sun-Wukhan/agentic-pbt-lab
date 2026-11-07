#!/usr/bin/env bash
# Reproduce the Masterminds/semver property-based testing runs from the
# Agentic PBT Lab.
#
#   ./run.sh [CHECKS]
#
# The harness is a SEPARATE Go module (./harness) that depends on the
# target through a replace directive pointing at the pinned clone in
# ./semver — the clone's own files are never modified. run.sh clones
# the target at the pinned commit if needed, verifies the pin, and runs
# every TestPBT* property.
#
# CHECKS is rapid's -rapid.checks value (default: 100, rapid's own
# default). Set GO to override the go binary.
set -euo pipefail

PIN="dd2b995c61c39ddd668b23ac919b04d607be35ab"
CHECKS="${1:-100}"
GO="${GO:-$HOME/workspace/tools/go/bin/go}"
HERE="$(cd "$(dirname "$0")" && pwd)"

export GOCACHE="${GOCACHE:-$HOME/workspace/.gocache}"
export GOTMPDIR="${GOTMPDIR:-$HOME/workspace/.gotmp}"
mkdir -p "$GOCACHE" "$GOTMPDIR"

if [ ! -d "$HERE/semver/.git" ]; then
  echo ">> cloning Masterminds/semver into $HERE/semver"
  git clone --depth 1 https://github.com/Masterminds/semver.git "$HERE/semver"
fi
HEAD="$(git -C "$HERE/semver" rev-parse HEAD)"
if [ "$HEAD" != "$PIN" ]; then
  echo ">> checking out pinned commit $PIN (was $HEAD)"
  git -C "$HERE/semver" fetch --quiet --depth 1 origin "$PIN" || git -C "$HERE/semver" fetch --quiet
  git -C "$HERE/semver" checkout --quiet "$PIN"
fi
if [ -n "$(git -C "$HERE/semver" status --porcelain)" ]; then
  echo "!! warning: semver clone is not pristine" >&2
  git -C "$HERE/semver" status --porcelain >&2
fi

cd "$HERE/harness"
echo ">> running property tests with -rapid.checks=$CHECKS"
"$GO" test ./... -run 'TestPBT' -count=1 -timeout 30m -rapid.checks="$CHECKS" -v
