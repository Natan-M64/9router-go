#!/usr/bin/env bash
set -euo pipefail

base=$(git rev-parse --verify --end-of-options "${1:?fork base required}^{commit}")
upstream=$(git rev-parse --verify --end-of-options "${2:?upstream tip required}^{commit}")

report() {
  printf '%s\n' "$*"
  if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
    printf '%s\n' "$*" >> "$GITHUB_STEP_SUMMARY"
  fi
}

report "Fork base: $base; upstream tip: $upstream."
if git merge-base --is-ancestor "$upstream" "$base"; then
  report 'Already synchronized.'
  exit 0
fi

if preview=$(git merge-tree --write-tree --name-only --no-messages "$base" "$upstream"); then
  tree=$preview
else
  status=$?
  if [[ $status == 1 ]]; then
    report 'Merge conflicts require manual resolution:'
    while IFS= read -r path; do
      # Quote paths before reporting upstream-controlled names.
      [[ -z $path ]] || report "- $(printf '%q' "$path")"
    done <<< "${preview#*$'\n'}"
  else
    report "Git merge simulation failed (exit $status)."
  fi
  exit "$status"
fi

if ! git diff --quiet --no-ext-diff --no-renames "$base" "$tree" -- CHANGELOG.md .github/workflows .github/actions; then
  report 'Manual review required: the merge changes CHANGELOG.md, .github/workflows or .github/actions.'
  exit 1
fi

if ! git diff --check --no-ext-diff "$base" "$tree" > /dev/null; then
  report 'Manual review required: the merge introduces whitespace errors or conflict markers.'
  exit 1
fi

if ! git diff --name-only --diff-filter=AM --no-ext-diff --no-renames "$base" "$tree" -- .changes |
  grep -E '^\.changes/[^/]+\.md$' > /dev/null; then
  report 'Manual review required: add a .changes fragment for the sync PR before publishing.'
  exit 1
fi

count=$(git rev-list --count "$base..$upstream")
report "Clean merge candidate: $count incoming commits."
if [[ -n ${GITHUB_OUTPUT:-} ]]; then
  printf 'base=%s\nupstream=%s\ntree=%s\ncount=%s\n' "$base" "$upstream" "$tree" "$count" >> "$GITHUB_OUTPUT"
fi
