### chore(sync): merge luqman-v1 upstream through `812a8383`
- Integrate 41 upstream commits from fork base `641e842d`, keeping the fork's
  `ci-local` gate and Antigravity account-ban handling while adopting the
  upstream race timeout and context-aware connection error writes.
- Merge five commits through `cd3978fc` without conflicts: request-scoped chat
  settings reuse, OpenCode session rotation on HTTP 429, wrapped daemon sentinel
  handling, and wiki documentation and link updates.
- Merge the `1.9.18-exp.1` release commit by adopting its `VERSION` and
  `updater.CurrentVersion` bump while keeping `CHANGELOG.md` and the existing
  `.changes` fragments at the fork's version. The generated changelog fold is
  intentionally not imported: the dashboard assembles `CHANGELOG.md` plus
  fragments, so dropping the fragments without the fold would remove their
  entries, and a `CHANGELOG.md` hunk is not allowed in a PR (§7.6).
- Isolate the CLI `doctor` test from the host daemon and retain explicit
  `--port 0` handling without the unused wrapper.
- Add a scheduled and manual upstream preview. Draft-PR publication is opt-in;
  the workflow stops at conflicts and policy guards and never merges to `main`.
- Verified locally with `make ci-local`, a final full Go test run, incremental
  Go lint, `actionlint`, `shellcheck`, and merge-preview cases from real history.
  PostgreSQL smoke and hosted workflow execution remain CI gates.
