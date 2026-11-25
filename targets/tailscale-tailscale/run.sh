#!/usr/bin/env bash
# Reproduce the tailscale/tailscale property-based testing runs from the
# Agentic PBT Lab.
#
#   ./run.sh [CHECKS]
#
# The harness is a SEPARATE Go module (./harness) that depends on the
# target through a replace directive pointing at the pinned clone in
# ./tailscale — the clone's own files are never modified. run.sh clones
# the target at the pinned commit if needed, verifies the pin, and runs
# every TestPBT* property.
#
# CHECKS is rapid's -rapid.checks value (default: 100, rapid's own
# default). Set GO to override the go binary.
set -euo pipefail

PIN="445df50dcf76b6172dc4688c2097c47878f41641"
CHECKS="${1:-100}"
GO="${GO:-$HOME/workspace/tools/go/bin/go}"
HERE="$(cd "$(dirname "$0")" && pwd)"

export GOCACHE="${GOCACHE:-$HOME/workspace/.gocache}"
export GOTMPDIR="${GOTMPDIR:-$HOME/workspace/.gotmp}"
mkdir -p "$GOCACHE" "$GOTMPDIR"

if [ ! -d "$HERE/tailscale/.git" ]; then
  echo ">> cloning tailscale/tailscale into $HERE/tailscale"
  git clone --depth 1 https://github.com/tailscale/tailscale.git "$HERE/tailscale"
fi
HEAD="$(git -C "$HERE/tailscale" rev-parse HEAD)"
if [ "$HEAD" != "$PIN" ]; then
  echo ">> checking out pinned commit $PIN (was $HEAD)"
  git -C "$HERE/tailscale" fetch --quiet --depth 1 origin "$PIN" || git -C "$HERE/tailscale" fetch --quiet
  git -C "$HERE/tailscale" checkout --quiet "$PIN"
fi
if [ -n "$(git -C "$HERE/tailscale" status --porcelain)" ]; then
  echo "!! warning: tailscale clone is not pristine" >&2
  git -C "$HERE/tailscale" status --porcelain >&2
fi

cd "$HERE/harness"
echo ">> running property tests with -rapid.checks=$CHECKS"
"$GO" test ./... -run 'TestPBT' -count=1 -timeout 30m -rapid.checks="$CHECKS" -v
