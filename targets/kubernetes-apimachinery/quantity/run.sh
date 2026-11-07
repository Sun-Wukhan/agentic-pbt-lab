#!/usr/bin/env bash
# Reproduce the wave-2 kubernetes/apimachinery resource.Quantity PBT runs.
#
#   ./run.sh [CHECKS]
#
# Runs the rapid property tests in this directory. The harness module
# (./go.mod) replaces k8s.io/apimachinery with the lab's local clone at
# ../apimachinery (whose pkg/api/resource is byte-identical to the wave-1
# pin b26d01364f13520dfdecb540fc63253be5b76662); the clone itself is never
# modified. CHECKS is rapid's -rapid.checks value (default 100).
# TestPBTQuantityExponentRange is expected to FAIL at the pinned commit —
# it encodes finding 001 (exponent suffix wraparound).
set -uo pipefail

CHECKS="${1:-100}"
GO="${GO:-$HOME/workspace/tools/go/bin/go}"
HERE="$(cd "$(dirname "$0")" && pwd)"

export GOCACHE="${GOCACHE:-$HOME/workspace/.gocache}"
export GOTMPDIR="${GOTMPDIR:-$HOME/workspace/.gotmp}"
export GOFLAGS=-mod=mod

cd "$HERE"
"$GO" test . -run 'TestPBT' -count=1 -timeout 30m -rapid.checks="$CHECKS" -v
