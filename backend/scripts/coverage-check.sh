#!/usr/bin/env bash
# Fails when total statement coverage in a Go cover profile is below a threshold.
# Usage: coverage-check.sh <coverprofile> <min-percent>
set -euo pipefail

profile=${1:?usage: coverage-check.sh <coverprofile> <min-percent>}
min=${2:?usage: coverage-check.sh <coverprofile> <min-percent>}

total=$(go tool cover -func="$profile" | awk '/^total:/ { sub(/%/, "", $NF); print $NF }')
if [[ -z $total ]]; then
  echo "coverage-check: no total found in $profile" >&2
  exit 1
fi

if awk -v t="$total" -v m="$min" 'BEGIN { exit !(t < m) }'; then
  echo "coverage-check: FAIL total ${total}% < ${min}%" >&2
  exit 1
fi
echo "coverage-check: OK total ${total}% >= ${min}%"
