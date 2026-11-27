#!/usr/bin/env bash
set -euo pipefail

base=$(git rev-parse --verify --end-of-options "${1:?fork base required}^{commit}")
upstream=$(git rev-parse --verify --end-of-options "${2:?upstream tip required}^{commit}")

# output publishes one key/value to the step outputs when running under Actions.
# A failed write aborts the script on purpose: a silently missing status would
# leave the publish step skipped while the run still looked healthy.
output() {
  if [[ -n ${GITHUB_OUTPUT:-} ]]; then
    printf '%s=%s\n' "$1" "$2" >> "$GITHUB_OUTPUT"
  fi
}

report() {
  printf '%s\n' "$*"
  if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
    printf '%s\n' "$*" >> "$GITHUB_STEP_SUMMARY" || true
  fi
  return 0
}

# fail is a real Git failure: the run must go red so it is not mistaken for a
# guard that merely needs a human.
fail() {
  report "$1"
  output status error
  exit 1
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
    fail "Git merge simulation failed (exit $merge_status)."
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

# git diff --quiet: 0 = no change, 1 = differences, anything else = a Git error.
set +e
git diff --quiet --no-ext-diff --no-renames "$base" "$tree" -- CHANGELOG.md .github/workflows .github/actions
protected_status=$?
set -e
case $protected_status in
  0) ;;
  1)
    report 'The merge changes CHANGELOG.md, .github/workflows or .github/actions.'
    needs_review 'protected paths changed; resolve and merge by hand'
    ;;
  *) fail "git diff could not check protected paths (exit $protected_status)." ;;
esac

# git diff --check: 0 = clean, 2 = whitespace errors or conflict markers.
set +e
git diff --check --no-ext-diff "$base" "$tree" > /dev/null
whitespace_status=$?
set -e
case $whitespace_status in
  0) ;;
  2)
    report 'The merge introduces whitespace errors or conflict markers.'
    needs_review 'whitespace errors or conflict markers in the merge result'
    ;;
  *) fail "git diff --check could not inspect the merge result (exit $whitespace_status)." ;;
esac

if ! names=$(git diff --name-only --diff-filter=AM --no-ext-diff --no-renames "$base" "$tree" -- .changes); then
  fail 'git diff could not list the merged changelog fragments.'
fi
if ! grep -E '^\.changes/[^/]+\.md$' <<< "$names" > /dev/null; then
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
