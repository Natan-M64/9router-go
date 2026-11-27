### ci(sync): report upstream preview status without failing the run
- The upstream preview now reports `up_to_date`, `needs_review`, `ready` or
  `error`. Conflicts, protected paths, whitespace defects and a missing fragment
  end with a warning annotation instead of a failed scheduled run; only a real
  Git failure exits non-zero.
- Draft-PR publication is gated on `status=ready`, so a stopped preview can never
  publish. Duplicate suppression covers every open same-repo `sync/` PR, and
  `actions/checkout` moves to v4.4.0, the latest release in the v4 major used
  across the workflows. The Node.js 20 deprecation warning that action still
  emits clears only with checkout v5 or newer.
- `docs/upstream-sync.md` records the statuses, guards, publication opt-in with
  `UPSTREAM_SYNC_TOKEN` scopes, and the release-commit resolution.
- Verified with six preview scenarios against real history plus `shellcheck` and
  `actionlint`; the publication path still needs one opted-in run with the token.
