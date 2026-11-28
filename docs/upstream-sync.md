# Upstream fork sync

This fork tracks `luqman-v1/9router-go:main`. `decolua/9router` stays the separate
behavioral parity reference; the automation only mirrors the fork's own upstream.

## What runs

`.github/workflows/upstream-fork-sync.yml` runs twice a day, at 03:00 and 15:00
UTC (midnight and noon at UTC-3), and on manual dispatch. It never checks out or
executes upstream code: it fetches the upstream tip, simulates the merge with
`git merge-tree`, and inspects the resulting tree.

`scripts/upstream-sync-preview.sh <base> <upstream>` does the work and reports a
status. Guards exit 0 with a `::warning::` annotation so a scheduled preview that
needs a human does not paint the run red.

| status | meaning |
| --- | --- |
| `up_to_date` | upstream is already contained in the fork |
| `needs_review` | a guard stopped the preview; see the run summary |
| `ready` | clean candidate; publication is allowed when opted in |
| `error` | a real Git failure in the simulation or in a guard; the run fails |

Guards that produce `needs_review`:

- merge conflicts;
- changes to `CHANGELOG.md`, `.github/workflows` or `.github/actions`;
- whitespace errors or conflict markers in the merged tree;
- a candidate without a top-level `.changes/*.md` fragment (required by the CI
  `Changelog fragment` job).

## Publication

Draft-PR creation is opt-in and never automatic on merge:

- manual run: dispatch with `publish=true`;
- scheduled run: repository variable `UPSTREAM_SYNC_PUBLISH=true`.

It also requires the repository secret `UPSTREAM_SYNC_TOKEN`, a fine-grained PAT
with `Contents: read and write` and `Pull requests: read and write` on this repo.
The default `GITHUB_TOKEN` is not enough: a PR it creates does not trigger
`pull_request` workflows, so the draft would arrive without CI.

The job refuses to publish while an open same-repo PR from a `sync/` branch
exists, and re-checks that `origin/main` still matches the previewed base.

Nothing merges to `main` automatically. Merge with a merge commit so upstream
ancestry is preserved.

## Release commits

An upstream release commit (for example `1.9.18-exp.1`) folds every fragment into
`CHANGELOG.md` and deletes them. It stops the automation: the simulated merge
conflicts in `CHANGELOG.md`, and resolving it by taking upstream's side would also
change the protected paths, so the preview reports `needs_review` either way.

Resolve it the same way every time:

1. Merge the release commit and keep the fork's `CHANGELOG.md`
   (`git checkout HEAD -- CHANGELOG.md`).
2. Keep the fragments (`git checkout HEAD -- .changes`) so the entries stay
   visible: `GET /api/changelog` assembles `CHANGELOG.md` plus fragments, and
   dropping the fragments without the fold would delete those entries.
3. Adopt the `VERSION` and `internal/updater/updater.go` bump.
4. Record the decision in the sync's `.changes` fragment.

A `CHANGELOG.md` hunk must not appear in a PR (AGENTS §7.6); the file is written
by `make changelog-merge` at release time only.

## Verifying the automation

```bash
# local guards, against real history
GITHUB_OUTPUT=/dev/stdout bash scripts/upstream-sync-preview.sh <base> <upstream>

# hosted preview (no publication)
gh workflow run upstream-fork-sync.yml --repo <owner>/9router-go --ref main
gh run view --repo <owner>/9router-go --log
```

The publication path is only proven by one opted-in run that produces a draft and
triggers its `pull_request` CI.
