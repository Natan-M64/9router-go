#!/usr/bin/env bash
# Concatenates the per-shard coverage profiles into one profile.
#
# The shards partition the package list, so no block appears in two profiles and
# concatenating the bodies is an exact union — no counter arithmetic needed.
set -euo pipefail

dir=${1:?profile directory required}
out=${2:?output path required}

shopt -s nullglob
files=("$dir"/coverage-*.out)
if (( ${#files[@]} == 0 )); then
  printf 'no coverage profiles matching %s/coverage-*.out\n' "$dir" >&2
  exit 1
fi

sorted=()
while IFS= read -r f; do
  [[ -n $f ]] && sorted+=("$f")
done < <(printf '%s\n' "${files[@]}" | sort)

header=$(head -n 1 "${sorted[0]}")
case $header in
  mode:*) ;;
  *)
    printf 'unexpected coverage header in %s: %q\n' "${sorted[0]}" "$header" >&2
    exit 1
    ;;
esac

{
  printf '%s\n' "$header"
  for f in "${sorted[@]}"; do
    tail -n +2 "$f"
  done
} > "$out"

printf 'merged %d profiles into %s\n' "${#sorted[@]}" "$out"
