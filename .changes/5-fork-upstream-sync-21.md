### chore(sync): merge luqman-v1 upstream through `cd3978fc`
- Integrate 36 upstream commits from fork base `641e842d`, keeping the fork's
  `ci-local` gate and Antigravity account-ban handling while adopting the
  upstream race timeout and context-aware connection error writes.
- Merge five more commits through `cd3978fc` without conflicts: request-scoped
  chat settings reuse, OpenCode session rotation on HTTP 429, wrapped daemon
  sentinel handling, and wiki documentation and link updates.
- Isolate the CLI `doctor` test from the host daemon and retain explicit
  `--port 0` handling without the unused wrapper.
- Add a scheduled and manual upstream preview. Draft-PR publication is opt-in;
  the workflow stops at conflicts and policy guards and never merges to `main`.
- Verified locally with `make ci-local`, a final full Go test run, incremental
  Go lint, `actionlint`, `shellcheck`, and merge-preview cases from real history.
  PostgreSQL smoke and hosted workflow execution remain CI gates.
