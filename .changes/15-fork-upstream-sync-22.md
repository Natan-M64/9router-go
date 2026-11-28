### chore(sync): merge luqman-v1 upstream through `0dfe6347`
- Integrate six upstream commits from fork base `015e417d`: the dashboard API
  client split by domain (#1534), the media handler split by endpoint family
  (#1533), provider error classification split (#1519), the keys import endpoint
  (#1577), the CodeBuddy empty-reasoning fix (#1575) and the model-deprecation
  banner on 200 streams (#1523).
- `web/src/api/client.ts` conflicted because the fork carried two edits on the
  monolith the refactor replaced. The merge keeps upstream's `endpoints/*`
  structure and ports both fork changes: `createCombo` accepts `matchRules` as
  JSON text, and the jailbreak keyword master switch is exposed from
  `endpoints/jailbreak.ts` (used by `JailbreakView.svelte`).
- The sync automation stopped at the conflict guard instead of publishing a
  broken merge (run `46157633326`).
- Verified: `go test -count=1 ./...` 11,292 passing in 85 packages, `go vet`
  clean, `bun test src scripts` 1,092 passing, `bun run build` and the
  svelte-check ratchet green; `CHANGELOG.md` untouched.
