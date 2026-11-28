#!/usr/bin/env bash
# Runs one shard of the integration suite.
#
# The suite is a single Go package, so it cannot be split by package. Instead
# the test names are discovered with `go test -list`, sorted, and partitioned
# deterministically; the shard runs only its own names through -run. Every test
# lands in exactly one shard.
#
#   ci-integration-shard.sh <shard> <total>
#   SHARD_DRY_RUN=1 ci-integration-shard.sh <shard> <total>   prints the regex
set -euo pipefail

shard=${1:?shard index required}
total=${2:?shard count required}
pkg='./internal/integration/...'

if ! [[ $shard =~ ^[0-9]+$ && $total =~ ^[0-9]+$ ]] || (( total < 1 || shard < 1 || shard > total )); then
  printf 'usage: %s <shard 1..total> <total>\n' "$0" >&2
  exit 2
fi

tests=()
while IFS= read -r name; do
  [[ -n $name ]] && tests+=("$name")
done < <(go test -tags=integration -list '^Test' "$pkg" | grep -E '^Test' | sort)

if (( ${#tests[@]} == 0 )); then
  printf 'no integration tests discovered in %s\n' "$pkg" >&2
  exit 1
fi

mine=()
while IFS= read -r name; do
  [[ -n $name ]] && mine+=("$name")
done < <(printf '%s\n' "${tests[@]}" | awk -v s="$shard" -v n="$total" 'NR % n == (s - 1) % n')

if (( ${#mine[@]} == 0 )); then
  printf 'shard %s/%s is empty\n' "$shard" "$total" >&2
  exit 1
fi

joined=$(printf '%s|' "${mine[@]}")
regex="^(${joined%|})$"
printf 'shard %s/%s: %d of %d integration tests\n' "$shard" "$total" "${#mine[@]}" "${#tests[@]}"

if [[ -n ${SHARD_DRY_RUN:-} ]]; then
  printf '%s\n' "$regex"
  exit 0
fi

exec go test -tags=integration -race -count=1 -timeout 10m -run "$regex" -v "$pkg"
