#!/usr/bin/env bash
# Run every built lab in this repo and refresh the scoreboard inputs.
# Usage: ./run-all.sh [slug ...]   (default: all targets with a run.sh)
set -uo pipefail
cd "$(dirname "$0")"
mkdir -p results
targets=("$@")
if [ ${#targets[@]} -eq 0 ]; then
  for d in targets/*/; do
    [ -f "$d/run.sh" ] && targets+=("$(basename "$d")")
  done
fi
overall=0
for slug in "${targets[@]}"; do
  [ "$slug" = "_template" ] && continue
  echo "=== $slug ==="
  if ( cd "targets/$slug" && bash run.sh ) > "results/$slug.log" 2>&1; then
    echo "$slug: OK (log: results/$slug.log)"
  else
    echo "$slug: FAILED or found counterexamples (log: results/$slug.log)"
    overall=1
  fi
done
exit $overall
