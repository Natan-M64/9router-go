### sync(upstream): merge `luqman-v1/9router-go` v1.9.17 (44 commits)
- Merge of `origin/main` (`67a4bc94`, `chore: bump to 1.9.17`) into
  `sync/upstream-20`, from base `ccd37917`. Upstream work carried in per-model
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
- Verified: `go build ./...` and `go vet ./...` exit 0; `go test -count=1 ./...`
  exit 0 (80 packages, no failure); `make test-integration` exit 0; `bun test
  src scripts` 1091 pass / 0 fail; `bun run ratchet:svelte` 0 errors;
  `bunx oxlint --deny-warnings` exit 0.
