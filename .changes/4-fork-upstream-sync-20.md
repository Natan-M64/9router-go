### sync(upstream): merge `luqman-v1/9router-go` v1.9.17
- Merge of `origin/main` through `46a9b69d` into `sync/upstream-20`, initially
  at `67a4bc94` (`chore: bump to 1.9.17`) and then advanced by 17 unique
  commits (7 on the first-parent line).
  Upstream work carried in per-model
  capability editor (#1479), the Playground tab (#1466), per-key RPH/RPD and
  per-IP rate limits (#1476), per-model request caps for custom providers
  (#1456), the full Qoder account catalog (#1468), Claude seat tier and
  `agy-acp` rows in the quota tracker (#1486, #1497), sealed vault writes for
  OAuth connection data (#1474), the cached live model catalogue, the UC Web
  image SSRF block (#1502), and the OpenWrt opkg package.
- Conflicts kept both sides: `.PHONY` in `Makefile` (`ci-local` and
  `ipk ipk-bins`), the keyword switch plus the catalogue `onMount` in
  `JailbreakView.svelte`, the keyword routes in both route tables, and the two
  appended test blocks in the jailbreak database and dashboard tests.
- `dashboard/usage.go` keeps the fork's `writeUsagePayload` cache and adopts the
  upstream `ctx` argument on `syncCodexPlanFromUsage`; `writeUsagePayload`
  already writes through `handlerutil.WriteJSON`.
- The upstream moved `context.Context` into `CreateProviderConnectionFull` and
  `ReplaceProviderConnectionPayload` (#1473). Four fork call sites still used
  the old signature and are updated: `usage_cache_test.go` and
  `maintain_cmd_test.go` (a fork-only file).
- Follow-up sync through `46a9b69d`: added the OpenCode free-model live feed,
  retryable compression analytics backfill, OpenWrt healthcheck/LAN firewall
  settings, bounds-checked Gemini Web registry flags, visible browser-cookie
  scan errors, deterministic combo-budget tests, and a 400 response for
  non-object Responses compact bodies. The merge simulation reported no
  conflicts; the upstream tree was merged without manual conflict edits.
- The upstream release's generated 712-line `CHANGELOG.md` section is not
  imported into this PR: the file remains at the fork base, the fork's existing
  `.changes` fragments are retained, and incoming upstream changes use their
  fragment files.
- Verified: `go build ./...` and `go vet ./...` exit 0; `go test -count=1 ./...`
  exit 0 (80 packages, no failure); `make test-integration` exit 0; `bun test
  src scripts` 1091 pass / 0 fail; `bun run ratchet:svelte` 0 errors;
  `bunx oxlint --deny-warnings` exit 0. After the follow-up sync, `go test
  -count=1 ./...` passed (85 packages), `go vet ./...` exited 0, and
  `make test-integration` passed. An independent merge review found no lost
  fork changes or blockers; it noted one upstream-only double parse in
  `live_catalog_opencode.go` for separate follow-up consideration.
