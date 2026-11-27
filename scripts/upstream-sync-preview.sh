#!/usr/bin/env bash
set -euo pipefail

base=$(git rev-parse --verify --end-of-options "${1:?fork base required}^{commit}")
upstream=$(git rev-parse --verify --end-of-options "${2:?upstream tip required}^{commit}")

# output publishes one key/value to the step outputs when running under Actions.
output() {
  if [[ -n ${GITHUB_OUTPUT:-} ]]; then
    printf '%s=%s\n' "$1" "$2" >> "$GITHUB_OUTPUT"
  fi
  return 0
}

report() {
  printf '%s\n' "$*"
  if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
    printf '%s\n' "$*" >> "$GITHUB_STEP_SUMMARY"
  fi
  return 0
}

# needs_review ends a preview that found work for a human. It exits 0 on
# purpose: a scheduled preview stopping on a guard is a normal outcome, and a
# red run every time upstream touches the changelog or the actions config would
# train the operator to ignore it. Publication is gated on the status output,
# and only a real Git failure exits non-zero.
needs_review() {
  output status needs_review
  printf '::warning title=Upstream sync needs review::%s\n' "$1"
  exit 0
}

report "Fork base: $base; upstream tip: $upstream."
if git merge-base --is-ancestor "$upstream" "$base"; then
  report 'Already synchronized.'
  output status up_to_date
  exit 0
fi

if preview=$(git merge-tree --write-tree --name-only --no-messages "$base" "$upstream"); then
  tree=$preview
else
  merge_status=$?
  if [[ $merge_status != 1 ]]; then
    report "Git merge simulation failed (exit $merge_status)."
    output status error
    exit 1
  fi
  report 'Merge conflicts require manual resolution:'
  conflicts=${preview#*$'\n'}
  if [[ -z $conflicts ]]; then
    report '- (no path list returned; resolve the merge locally)'
  else
    while IFS= read -r path; do
      # Quote paths before reporting upstream-controlled names.
      [[ -z $path ]] || report "- $(printf '%q' "$path")"
    done <<< "$conflicts"
  fi
  needs_review 'merge conflicts require manual resolution'
fi

if ! git diff --quiet --no-ext-diff --no-renames "$base" "$tree" -- CHANGELOG.md .github/workflows .github/actions; then
  report 'The merge changes CHANGELOG.md, .github/workflows or .github/actions.'
  needs_review 'protected paths changed; resolve and merge by hand'
fi

if ! git diff --check --no-ext-diff "$base" "$tree" > /dev/null; then
  report 'The merge introduces whitespace errors or conflict markers.'
  needs_review 'whitespace errors or conflict markers in the merge result'
fi

if ! git diff --name-only --diff-filter=AM --no-ext-diff --no-renames "$base" "$tree" -- .changes |
  grep -E '^\.changes/[^/]+\.md$' > /dev/null; then
  report 'The sync candidate carries no changelog fragment.'
  needs_review 'add a .changes fragment for the sync PR before publishing'
fi

count=$(git rev-list --count "$base..$upstream")
report "Clean merge candidate: $count incoming commits."
output base "$base"
output upstream "$upstream"
output tree "$tree"
output count "$count"
output status ready
