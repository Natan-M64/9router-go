#!/usr/bin/env bash
# Prints the packages belonging to one shard of `go list ./...`.
#
# The split is deterministic: the sorted package list is walked and every
# virtual shard takes the packages whose 1-based position leaves the same
# remainder. Every package lands in exactly one shard, and the union of all
# shards is the full list, so sharding never drops coverage of a package.
set -euo pipefail

shard=${1:?shard index required}
total=${2:?shard count required}

if ! [[ $shard =~ ^[0-9]+$ && $total =~ ^[0-9]+$ ]] || (( total < 1 || shard < 1 || shard > total )); then
  printf 'usage: %s <shard 1..total> <total>\n' "$0" >&2
  exit 2
fi

go list ./... | sort | awk -v s="$shard" -v n="$total" 'NR % n == (s - 1) % n'
