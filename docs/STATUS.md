# Status

Updated 2026-11-27 · Fork `Natan-M64/9router-go` (upstream `luqman-v1/9router-go` / `decolua/9router`)
Current milestone: **upstream sync 21** (`sync/upstream-21`) — PR #4 was merged. Merge `f40905b` brings fork base `641e842d` through upstream `de05f24d`. Conflict resolutions retain the fork's `ci-local` target and Antigravity account-ban handling, and adopt upstream's race timeout, context-aware scoped error write, and contributor counts. Require draft PR review and CI, including PostgreSQL, before merging to `main`.

Fork sync automation is prepared locally in `.github/workflows/upstream-fork-sync.yml`. The merge source is `luqman-v1/9router-go:main`; `decolua/9router` remains the separate behavioral parity reference. Once on `main` with scheduled Actions enabled for this fork, it previews daily at 09:23 UTC and supports manual preview. Conflicts, changes to `CHANGELOG.md` or Actions configuration, and candidates without a changelog fragment stop the run and surface the reason in the Actions summary. Draft publication is opt-in (`publish` on manual runs or `UPSTREAM_SYNC_PUBLISH=true` on scheduled runs), with a repository-scoped `UPSTREAM_SYNC_TOKEN` that can write contents and pull requests. Existing open PRs from `sync/upstream-*` branches of this fork suppress publication. Start with preview, then verify one draft run and its CI before enabling scheduled publication. Do not enable automatic drafts if PR CI executes candidate code with repository secrets. No automatic merge or push to `main`.
Last proof: [journal/2026-11.md](journal/2026-11.md).
Optimization plan: [optimization-roadmap.md](optimization-roadmap.md) · RTK filters: [RTK_FILTERS.md](RTK_FILTERS.md).
Limits of this file: ≤120 lines, ≤8 KB, ≤2 cards in Now.

## Now

| ID | Expected result | State |
|---|---|---|
| N1 | **Billing and quota**: Venice prices checked against the operator's invoice; metered spend kept apart from list-price cost (`opencode-go`, `ollama`, `xiaomi-tokenplan` corrected); Venice balance from inference headers with an optional ADMIN probe; 70 s usage cache with frontend hydration; cache-break diagnosis with divergent index and cost | Merged in PR #2 (`ccb977dc`). 7 days after restart: metered `$0.2217`, list-price `$30.6896`, unclassified `$0`. |
| O2 | **MVP proof, remainder**: Claude, Codex, OpenRouter, Venice, DeepSeek and opencode-go answered 200; `ocg/deepseek-v4.1-flash-free` also completed a live Kilo text request with 200. Need stream/tool-call JSON probes; gemini/groq/kilocode availability and post-merge CI remain to be checked | Partial. |

## Next (in order)

| ID | Scope | Exit / dependency |
|---|---|---|
| O1 | **NVIDIA catalog**: extinct (410) and non-chat entries removed (`fbf5b96e`) | No NVIDIA traffic yet; confirm the next request lands on a live id. |
| O3 | **Block 2**: prefix-hash affinity (HRW; `session_affinity.go` already exists) together with the `x-9router-connection` header (3.4), because both decide the connection; prefix-safe compression; deduplicate `SemanticCache`/`PromptCache` | After O1. |
| O4 | **Rest of Block 3**: failover for a truncated stream or a stall, Progressive Tool Disclosure, Anti-Avalanche/`RefusalGuard` | After O3. |
| O5 | Headroom per-message guard (1.2b) and the cost of RTK's first pass (about 114 ms per 73 KB of new text) | Needs a design decision. |
| O6 | Issue #1 follow-up (runtime flags in `settings`) | Low priority. |

## Risks and open findings

| ID | Item | Action / who unblocks |
|---|---|---|
| R1 | Long sessions (`opencode-go/deepseek-v4.1-flash`, `antigravity`) average 188 messages and ~29 tools; any change to the prefix (`system`/`tools`) zeroes the KV cache | Always test prefix stability when touching `tokensaver`/`headroom`. |
| R3 | The caveman clarity note is not applied to bodies with no `messages` array (Responses/Gemini) | Decide what those formats should get. |
| R4 | Content-based filter selection still claims 1 of 917 files (a changelog that quotes stack traces, via `error-stacktrace`), and the audit covered only repository files, not real traffic | Review on a real sample of tool results. |
| R6 | The extra ~3000 tokens and the odd suffix came from 54 auto-synced third-party presets whose keyword rules armed requests on their own. Keyword rules and the sync are off by default; a combo opts in from its edit dialog, existing instances keep the presets and any enabled sync source, and the hourly refresh only runs for a source an enabled combo uses | Remove or disable the sync source and unused presets if wanted; open: a combo bound off still lets a leg's keyword fire when the global switch is on. |
| R7 | Roadmap 3.1 (`prompt is too long` as a 400 instead of a 503) already held upstream; only a key's context-limit refusal used different wording (adjusted) | REFUTED in the roadmap. |
| R8 | `deepseek-flash` is the alias of the latest Flash (V4.1-Flash-1017), so the `v4.1-flash` mapping is right. `deepseek-v4-flash`/`-vision-exp` are retired and "temporarily routed" to V4.1 Flash; `deepseek-chat`/`reasoner` were due to retire on 2026-07-24 but still answered 200 on 2026-11-24 | Prune only when the API stops accepting them; recheck `https://api-docs.deepseek.com/updates.md`. |
| R9 | Fork redundancy review: `capPreservedCacheMarkers` now reuses `cacheControlMarkerBudget`, and `make ci-local` calls the existing `vet` target. Open: `providers.ResolveModelAlias` has no callers and duplicates the executor's `deepSeekV4ProAlias` (DeepSeek ids in three tables); the Headroom `MemoryStore` partly overlaps the RTK result cache | Delete or wire the dead alias table; revisit `MemoryStore` once sessions show whether it pays. |
| R10 | A bare-429 retry cost 1–2.5 s on 4 of 4 large requests in one conversation. A learned-shape memo now skips it from the second turn (per conversation, 15 min, falls back to the regular shape if refused); the trigger did not reproduce (176 KB prompt, 30 tools), so real benefit is unverified | Confirm in the log that "retrying with systemInstruction" only appears on the first turn. |
| R5 | RTK + built-in Headroom double pass measured at no cost (209.8 ms with and without), so roadmap 1.2a does not hold | REFUTED in the roadmap. |
| R11 | A reply hit `finish_reason: length` at 1133 tokens (limit 65536) on `main-fast` → `antigravity/gemini-4-hydrogen-low`. The gateway relays Gemini's `MAX_TOKENS` unchanged; the sent `maxOutputTokens` and `thoughtsTokenCount` are not recorded | Records now carry `finishReason` and the client's `maxTokens`; check both on the next `length`. |
| R12 | This session runs **through** the local gateway, so stopping or restarting that process cuts the session and the operator has to start it again by hand. The running binary predates the current source tree | Never stop/kill the gateway from tooling; rebuild and let the operator restart. |
| R13 | Kilo `JSON parsing failed` on `ocg/deepseek-v4.1-flash-free`: the provider packs the tool-call and terminal `tool_calls` chunks with a single `\n`, so a strict client reads one event with two payloads. `SSECopy.writeEvents` splits consecutive `data:` lines with a blank separator (`internal/proxy/sse_copy.go`, `sse_packed_data_events_test.go`), pinned red→green including byte-at-a-time reads | **Recurrence explained 2026-11-26**: `go version -m` shows the process on `:20130` (PID 88437) runs revision `ccb977dc`, built 11:41, before the fix commits (12:10, 12:27) — the fix was never deployed. Binary rebuilt to `ad42a401` at 14:34; the operator must restart the gateway, since this session runs through it. |
| R2 | The GHCR package `ghcr.io/natan-m64/9router-go` may be private by default on GitHub Packages | The user switches visibility to Public if `docker pull` without login is wanted. |

## Done in the last 7 days

- **GHCR multi-arch + minimal stack/compose/env (`cb46f6d1`, Issue #1)**: native `amd64` + `ubuntu-24.04-arm` workflow, non-root `9router` + `CMD ["serve"]`, `hadolint` + `actionlint`, a `web/dist` CI artifact.
- **On-demand mode, macOS/Linux (`c5baa784`, `8ed820dc`)**: `--idle-timeout` / `IDLE_TIMEOUT`, `9router-go maintain`, socket activation (`launchd` + `systemd`), `scripts/macos-on-demand.sh`.
- **Catalog, thinking UX and combos (`eb29c708`, `f1166342`, `41ea97db`)**: `gpt-6.2-sol`, `-sol[1m]`, `gpt-6.2-astra`, `-astra[1m]`, `gpt-6.1-terra`, a weekly upstream check, `thinkingLevels` badges, a per-model `Thinking:` selector, and `9router-combos-v1`/`omniroute-combos-v1` import-export with provider-abbreviation mapping.
- **Stabilisation (`d242f9ff` to `da41a01c`)**: daemon recognised by start time on macOS 27, `make ci-local`, images only on tags, RTK filter selection and result cache, snapshot compaction, prompt-cache fixes, OpenRouter `cache_control`, the DeepSeek wire name, the upstream sync and changelog fragments.

## Verified (2026-11-27)

- `sync/upstream-20`, after merging `origin/main` through `46a9b69d`: `go test -count=1 ./...` passed (85 packages), `go vet ./...` exited 0, and `make test-integration` passed. Independent review found no lost fork changes or blockers.
- Not run for this sync: `make cross`, `go test -race`, the e2e dashboard suites, the PostgreSQL smoke, and OpenWrt shell validation.
- Review-fix validation: full `go test -count=1 ./...`, `go vet ./...`, and `make test-integration` pass; Playground, SSE peeker and IP-limiter regression tests pass under focused race runs; `gofmt -d` and `git diff --check` are clean. PostgreSQL smoke skips locally because `TEST_PG_DSN` is unavailable; PR CI must verify it.
