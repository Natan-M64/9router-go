# Status

Updated 2026-11-26 · Fork `Natan-M64/9router-go` (upstream `luqman-v1/9router-go` / `decolua/9router`)
Current milestone: **Billing cost basis and Venice quota submitted in PR #2** (`feat/billing-venice`); local `main` remains 7 commits behind `fork/main` (`ccd37917` guardrail policies, `1706621d` session revocation on password change, `49b9f097` jailbreak per-key delete, wiki/images). Next: address review feedback and await CI/review.
Last proof: [journal/2026-11.md](journal/2026-11.md).
Optimization plan: [optimization-roadmap.md](optimization-roadmap.md) · RTK filters: [RTK_FILTERS.md](RTK_FILTERS.md).
Limits of this file: ≤120 lines, ≤8 KB, ≤2 cards in Now.

## Now

| ID | Expected result | State |
|---|---|---|
| N1 | **Billing and quota**: Venice prices checked against the operator's invoice; metered spend kept apart from list-price cost (`opencode-go`, `ollama`, `xiaomi-mimo` misclassified before); Venice balance from inference headers with an optional ADMIN probe; 70 s usage cache with frontend hydration; cache-break diagnosis with divergent index and cost | Submitted in PR #2 (`a907da3e`); review fixes pushed. 7 days after restart: metered `$0.2217`, list-price `$30.6896`, unclassified `$0`. |
| O2 | **MVP proof, remainder**: Claude, Codex, OpenRouter, Venice, DeepSeek and opencode-go answered 200 (plain and stream); gemini, groq and kilocode (status `unavailable` on the instance) and the CI `integration` job are still to be checked | Partial. |

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
| R12 | This session runs **through** the local gateway, so stopping or restarting that process cuts the session and the operator has to start it again by hand. The running binary predates the uncommitted tree, so what `:20130` serves is not what `main` now contains | Never stop/kill the gateway from tooling; commit, rebuild and let the operator restart. |
| R2 | The GHCR package `ghcr.io/natan-m64/9router-go` may be private by default on GitHub Packages | The user switches visibility to Public if `docker pull` without login is wanted. |

## Done in the last 7 days

- **GHCR multi-arch + minimal stack/compose/env (`cb46f6d1`, Issue #1)**: native `amd64` + `ubuntu-24.04-arm` workflow, non-root `9router` + `CMD ["serve"]`, `hadolint` + `actionlint`, a `web/dist` CI artifact.
- **On-demand mode, macOS/Linux (`c5baa784`, `8ed820dc`)**: `--idle-timeout` / `IDLE_TIMEOUT`, `9router-go maintain`, socket activation (`launchd` + `systemd`), `scripts/macos-on-demand.sh`.
- **Catalog, thinking UX and combos (`eb29c708`, `f1166342`, `41ea97db`)**: `gpt-6.2-sol`, `-sol[1m]`, `gpt-6.2-astra`, `-astra[1m]`, `gpt-6.1-terra`, a weekly upstream check, `thinkingLevels` badges, a per-model `Thinking:` selector, and `9router-combos-v1`/`omniroute-combos-v1` import-export with provider-abbreviation mapping.
- **Stabilisation (`d242f9ff` to `da41a01c`)**: daemon recognised by start time on macOS 27, `make ci-local`, images only on tags, RTK filter selection and result cache, snapshot compaction, prompt-cache fixes, OpenRouter `cache_control`, the DeepSeek wire name, the upstream sync and changelog fragments.

## Verified (2026-11-26)

- `go test ./... -count=1` exit 0 on the current tree: 11,327 tests in 84 packages (second run captured to a log, `EXIT=0`, no `FAIL`).
- `bun test src scripts`: 1057 pass / 0 fail, 2040 assertions, 82 files. `bun run ratchet:svelte`: 0 unresolved identifiers, 0 errors.
- Same tree, earlier today: `make test-integration`, `bun run build`, `oxlint` and both e2e suites (`quotaTrackerCache`, `usageCostBasis`) passed; the cache e2e failed on the base build without hydration, so it is a real regression test.
- Not re-run after the last edit: `make cross`, `go test -race`, Docker/GHCR.
- Token efficiency observed: prompt cache saved 412,247,811 tokens over 3,175 requests (about `$204.01`); compression saved 15,532,204 tokens.

## Verified (2026-11-24)

Moved to the journal: [journal/2026-11.md](journal/2026-11.md) — live DeepSeek records after restart, `make ci-local` (77 packages) and CI green, the upstream sync with `ds/deepseek-v4.1-flash` 400 → 200, and the live chat/stream pass. Not verified then: gemini/groq/kilocode, real Playwright snapshots, `jailbreak`/`plans` deep-link redirect.
