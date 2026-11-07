#!/usr/bin/env bash
# Reproduce the kubernetes/apimachinery property-based testing runs from
# the Agentic PBT Lab.
#
#   ./run.sh [CHECKS]
#
# Clones apimachinery at the pinned commit into a scratch directory,
# copies the pristine harness test files from ./harness into it, adds
# pgregory.net/rapid as a test dependency in the clone ONLY (the lab
# never modifies upstream files), and runs every TestPBT* property.
#
# CHECKS is rapid's -rapid.checks value (default: 100, rapid's own
# default). Set GO to override the go binary. Scratch space defaults to
# ~/workspace/.gotmp because /tmp is a small tmpfs on the lab machine;
# override with PBT_SCRATCH.
set -euo pipefail

PIN="b26d01364f13520dfdecb540fc63253be5b76662"
CHECKS="${1:-100}"
GO="${GO:-go}"
HERE="$(cd "$(dirname "$0")" && pwd)"
SCRATCH_BASE="${PBT_SCRATCH:-$HOME/workspace/.gotmp}"
mkdir -p "$SCRATCH_BASE"
SCRATCH="$(mktemp -d "$SCRATCH_BASE/apimachinery-pbt.XXXXXX")"
trap 'rm -rf "$SCRATCH"' EXIT

export GOCACHE="${GOCACHE:-$HOME/workspace/.gocache}"
export GOTMPDIR="${GOTMPDIR:-$HOME/workspace/.gotmp}"

echo ">> cloning kubernetes/apimachinery @ $PIN into $SCRATCH/apimachinery"
git clone --quiet https://github.com/kubernetes/apimachinery "$SCRATCH/apimachinery"
git -C "$SCRATCH/apimachinery" checkout --quiet "$PIN"

echo ">> copying harness files"
cp "$HERE/harness/pkg/api/resource/pbt_resource_test.go" "$SCRATCH/apimachinery/pkg/api/resource/"
cp "$HERE/harness/pkg/labels/pbt_labels_test.go"         "$SCRATCH/apimachinery/pkg/labels/"
cp "$HERE/harness/pkg/util/intstr/pbt_intstr_test.go"    "$SCRATCH/apimachinery/pkg/util/intstr/"
cp "$HERE/harness/pkg/apis/meta/v1/pbt_metav1_test.go"   "$SCRATCH/apimachinery/pkg/apis/meta/v1/"

cd "$SCRATCH/apimachinery"
echo ">> adding pgregory.net/rapid v1.2.0 as a test dependency (clone only)"
GOFLAGS=-mod=mod "$GO" get pgregory.net/rapid@v1.2.0

echo ">> running property tests with -rapid.checks=$CHECKS"
"$GO" test \
  ./pkg/api/resource/ ./pkg/labels/ ./pkg/util/intstr/ ./pkg/apis/meta/v1/ \
  -run 'TestPBT' -count=1 -timeout 30m -rapid.checks="$CHECKS" -v
