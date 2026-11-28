#!/usr/bin/env bash
# Runs vet and the test suite for one shard of the Go packages.
#
#   ci-go-shard.sh <shard> <total> test   vet + tests, with a coverage profile
#   ci-go-shard.sh <shard> <total> race   tests under the race detector
#
# The coverage profile is written to coverage-<shard>.out in the working
# directory; the coverage job concatenates them (the shards are disjoint, so a
# block never appears twice).
set -euo pipefail

shard=${1:?shard index required}
total=${2:?shard count required}
mode=${3:?mode required (test|race)}

here=$(cd -- "$(dirname -- "$0")" && pwd)

pkgs=()
while IFS= read -r pkg; do
  [[ -n $pkg ]] && pkgs+=("$pkg")
done < <(bash "$here/ci-shard.sh" "$shard" "$total")

if (( ${#pkgs[@]} == 0 )); then
  # An empty shard would make go test fall back to "." and look green.
  printf 'shard %s/%s is empty\n' "$shard" "$total" >&2
  exit 1
fi

printf 'shard %s/%s: %d packages\n' "$shard" "$total" "${#pkgs[@]}"

case $mode in
  test)
    go vet "${pkgs[@]}"
    go test -count=1 -covermode=atomic -coverprofile="coverage-${shard}.out" "${pkgs[@]}"
    ;;
  race)
    go test -race -count=1 -timeout 20m "${pkgs[@]}"
    ;;
  *)
    printf 'unknown mode %q (want test or race)\n' "$mode" >&2
    exit 2
    ;;
esac
