#!/usr/bin/env bash
# Reproduce the rclone property-based testing runs from the Agentic PBT Lab.
#
#   ./run.sh [CHECKS]
#
# Clones rclone at the pinned commit into a scratch directory, copies the
# pristine harness test files from ./harness into it, adds pgregory.net/rapid
# as a test dependency in the clone ONLY (the lab never modifies upstream
# files), and runs every TestPBT* property.
#
# CHECKS is rapid's -rapid.checks value (default: 100, rapid's own default).
# Set GO to override the go binary.
set -euo pipefail

PIN="1053463755dea408a4abd42bf2ce917a3884a62a"
CHECKS="${1:-100}"
GO="${GO:-go}"
HERE="$(cd "$(dirname "$0")" && pwd)"
SCRATCH="$(mktemp -d "${TMPDIR:-/tmp}/rclone-pbt.XXXXXX")"
trap 'rm -rf "$SCRATCH"' EXIT

echo ">> cloning rclone @ $PIN into $SCRATCH/rclone"
git clone --quiet https://github.com/rclone/rclone "$SCRATCH/rclone"
git -C "$SCRATCH/rclone" checkout --quiet "$PIN"

echo ">> copying harness files"
cp "$HERE/harness/fs/pbt_fs_test.go"               "$SCRATCH/rclone/fs/"
cp "$HERE/harness/fs/fspath/pbt_fspath_test.go"    "$SCRATCH/rclone/fs/fspath/"
cp "$HERE/harness/fs/filter/pbt_filter_test.go"    "$SCRATCH/rclone/fs/filter/"
cp "$HERE/harness/backend/crypt/pbt_cipher_test.go"     "$SCRATCH/rclone/backend/crypt/"
cp "$HERE/harness/backend/chunker/pbt_chunker_test.go"  "$SCRATCH/rclone/backend/chunker/"

cd "$SCRATCH/rclone"
echo ">> adding pgregory.net/rapid v1.2.0 as a test dependency (clone only)"
GOFLAGS=-mod=mod "$GO" get pgregory.net/rapid@v1.2.0

echo ">> running property tests with -rapid.checks=$CHECKS"
RCLONE_CONFIG=/notfound "$GO" test \
  ./fs/ ./fs/fspath/ ./fs/filter/ ./backend/crypt/ ./backend/chunker/ \
  -run 'TestPBT' -count=1 -timeout 30m -rapid.checks="$CHECKS" -v
