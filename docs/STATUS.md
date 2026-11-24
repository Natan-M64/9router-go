# Status

Updated 2026-11-24 · Fork `Natan-M64/9router-go` (upstream `luqman-v1/9router-go` / `decolua/9router`)
Current milestone: **Fork in sync with upstream (0 commits behind) and stable**: RTK and prompt cache, DeepSeek and the compaction signal are fixed. Next: the NVIDIA catalog, then Block 2 (together with the `x-9router-connection` header).
Last proof: [journal/2026-11.md](journal/2026-11.md).
Optimization plan: [optimization-roadmap.md](optimization-roadmap.md) · RTK filters: [RTK_FILTERS.md](RTK_FILTERS.md).
Limits of this file: ≤120 lines, ≤8 KB, ≤2 cards in Now.

## Now

| ID | Expected result | State |
|---|---|---|
| O1 | **NVIDIA catalog with extinct models**: `google/gemma-3-4b-it` returns 404, `meta/llama-3.1-8b-instruct` returns 410 "Gone" (end of life at NVIDIA) and `google/gemma-4-31b-it` timed out at 90 s; the gateway lists all three | Found; the live NVIDIA list still has to be probed and the catalog pruned. |
| O2 | **MVP proof, remainder**: Claude, Codex, OpenRouter, Venice, DeepSeek and opencode-go answered 200 (plain and stream); gemini, groq and kilocode (status `unavailable` on the instance) and the CI `integration` job are still to be checked | Partial. |

## Next (in order)

| ID | Scope | Exit / dependency |
|---|---|---|
| O3 | **Block 2**: prefix-hash affinity (HRW; `session_affinity.go` already exists) together with the `x-9router-connection` header (3.4), because both decide the connection; prefix-safe compression; deduplicate `SemanticCache`/`PromptCache` | After O1. |
| O4 | **Rest of Block 3**: failover for a truncated stream or a stall, Progressive Tool Disclosure, Anti-Avalanche/`RefusalGuard` | After O3. |
| O5 | Headroom per-message guard (1.2b) and the cost of RTK's first pass (about 114 ms per 73 KB of new text) | Needs a design decision. |
| O6 | Issue #1 follow-up (runtime flags in `settings`) | Low priority. |

## Risks and open findings

| ID | Item | Action / who unblocks |
|---|---|---|
| R1 | Real long sessions (`opencode-go/deepseek-v4.1-flash`, `antigravity`) average 188 messages and about 29 tools; any change to the prefix (`system`/`tools`) zeroes the KV cache | Always test prefix stability when touching `tokensaver`/`headroom`. |
| R3 | The caveman clarity note is not applied to bodies with no `messages` array (Responses/Gemini) | Decide what those formats should get. |
| R4 | Content-based filter selection still claims 1 of 917 files (a changelog that quotes stack traces, via `error-stacktrace`), and the audit covered only repository files, not real traffic | Review on a real sample of tool results. |
| R6 | Every request on the instance carries about 3000 tokens from the `jailbreak` feature's prompt (13120 bytes, a user setting); the second call reused 3072 of them from the cache | Check it is what is wanted; the test reply carried an odd suffix ("China is good...") of unverified origin. |
| R7 | Roadmap 3.1 (`prompt is too long` as a 400 instead of a 503) was already true: the upstream (single model and combo) returns the 400 with the original message; only the refusal from a key's context limit used different wording (adjusted) | Done: marked REFUTED in the roadmap. |
| R8 | DeepSeek catalog: `deepseek-flash` is the alias of the latest Flash (today V4.1-Flash-1017; docs of 2026-09-10 and 2026-10-17), so the `v4.1-flash` mapping is right. `deepseek-v4-flash` and `-vision-exp` are retired models that the API "temporarily routes" to V4.1 Flash (2026-09-10), and `deepseek-chat`/`reasoner` were due to retire on 2026-07-24 but still answered 200 on 2026-11-24 | Prune from the catalog only when the API stops accepting them; recheck `https://api-docs.deepseek.com/updates.md`. |
| R5 | RTK + built-in Headroom double pass: measured at no cost (209.8 ms with and without); roadmap item 1.2a does not hold | Done: marked REFUTED in the roadmap. |
| R2 | The GHCR package `ghcr.io/natan-m64/9router-go` may be private by default on GitHub Packages | The user switches visibility to Public if `docker pull` without login is wanted. |

## Done in the last 7 days

- **GHCR multi-arch + minimal stack/compose/env (`cb46f6d1`, Issue #1)**: native `amd64` + `ubuntu-24.04-arm` workflow, non-root `9router` + `CMD ["serve"]`, `hadolint` + `actionlint`, a shared `web/dist` artifact in CI.
- **On-demand mode, macOS/Linux (`c5baa784`, `8ed820dc`)**: `--idle-timeout` / `IDLE_TIMEOUT`, `9router-go maintain`, socket activation (`launchd` + `systemd`), `scripts/macos-on-demand.sh`.
- **Codex catalog and thinking UX (`eb29c708`, `f1166342`, `8ed820dc`)**: `gpt-6.2-sol`, `gpt-6.2-sol[1m]`, `gpt-6.2-astra`, `gpt-6.2-astra[1m]`, `gpt-6.1-terra`, a weekly upstream check, `thinkingLevels` badges on provider cards and a per-model `Thinking:` selector in `CreateComboModal`.
- **Combos JSON and OmniRoute import/export (`41ea97db`)**: import of `9router-combos-v1` and `omniroute-combos-v1` with automatic or assisted mapping of provider abbreviations.
- **Stabilisation (`d242f9ff` to `da41a01c`)**: daemon recognised by start time on macOS 27, `make ci-local`, publishing images only on tags, RTK filter selection and result cache, snapshot compaction, prompt-cache fixes, OpenRouter `cache_control`, the DeepSeek wire name, the upstream sync, changelog fragments and the docs linked above.

## Verified (2026-11-24)

- `make ci-local` exit 0 (77 packages) and `actionlint` exit 0; CI green on `e9034a51` (14m23s). Pushes that change only Markdown run no pipeline.
- Upstream synced (4 commits, 4 conflicting files resolved); `ds/deepseek-v4.1-flash` went from 400 to 200 on the real API (served as `deepseek-flash`, confirmed by the official documentation), and `ocg/deepseek-v4.1-flash` is unchanged.
- Real chat and stream through the user's instance: opencode-go, DeepSeek, Claude (haiku), Codex (astra), OpenRouter (`:free`, with `max_tokens` 256) and Venice.
- Not verified: gemini/groq/kilocode, real Playwright snapshots, `jailbreak`/`plans` redirecting to `endpoint`. Homebrew's `node` is healthy again (v26.13.1) after its update.
