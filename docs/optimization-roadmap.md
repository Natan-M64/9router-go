# Optimization roadmap: prompt cache and tokens

This file is the source of the "Block 1/2/3" plan referred to in [STATUS.md](STATUS.md). The original plan was written outside the repository (an audit of 2026-11-24); this version replaces it and already records what was done, what measurement refuted and what is still open. Update it here, not in local copies.

**Legend.** `DONE` = implemented, with a test and a green `make ci-local`. `REFUTED` = the premise did not hold when measured, so there is nothing to fix. `OPEN` = not done yet. `DECISION` = waits on a design choice by the maintainer. Hashes are commits on `main`.

## Real traffic diagnosis (600 sampled calls)

| Observed metric | Value | Implication |
|---|---|---|
| Dominant providers | `antigravity` (448), `opencode-go` (124), `claude` (25) | Focus on Antigravity, OpenCode Go (`deepseek-v4.1-flash`: 121 calls) and Claude |
| Cached tokens | `antigravity` 44.9M, `opencode-go` 9.76M, `claude` 1.74M | Keeping the KV cache prefix intact is the biggest saving multiplier |
| Session size | average 187.9 messages per request (max 647) | Any mutation of the prefix destroys the cache |
| Calls with `tools[]` | 415 of 600 (average 28.9 tools, max 37) | Compress tool output without touching the `tools[]` schema |

**A rule that applies to everything below:** `system` and `tools` stay byte-for-byte identical between turns. Test prefix stability whenever you touch `tokensaver` or `headroom`.

## Block 1: cache and tokens (quick wins)

| ID | Item | State | Evidence and what is left |
|---|---|---|---|
| 1.1 | Caveman Auto-Clarity must not break the prefix | DONE | `2961edbb`, corrected in `c9467d0c` (the note goes only on the true tail: in a tool loop the last message is a `tool` turn, and the note must not land on an earlier `user` message). Wiring test in `8e9c2520`. **Limits:** bodies with no `messages` array (Responses, Gemini) get no note; and on the next turn the client resends its history without it, so the cache is lost from that message on (still better than changing `system`). |
| 1.2a | Remove the RTK + built-in Headroom double pass | REFUTED | Measured through `applyTokenSavers`: 209.8 ms with and 209.8 ms without the built-in engine, identical output (the second pass works on text that is already folded). There is nothing to remove. |
| 1.2b | Headroom per-message guard (`guard.go` rejects the whole request if one tool is protected) | DECISION | Not changed. With the RTK result cache (`6d054e0c`) the cost of repeating dropped a lot; the design has to be decided before touching it. |
| 1.3.1 | `PreserveCacheControl` for OpenRouter | DONE | `97270d01` (a deliberate divergence from upstream: OpenRouter's documentation supports `cache_control` on content blocks) and the four-marker cap in `960981c4`, which also protects the four Alibaba gateways. |
| 1.3.2 | Cache anchor for Claude API-key requests too; a string `system` becomes a block | DONE | `2961edbb` and `c9467d0c`, **Anthropic upstream only**. Third-party providers that speak the Messages format stay as they were: it is not known whether they accept `ttl` or an array `system`. |
| 1.3.3 | `extended-cache-ttl` beta header when the TTL is `1h` | REFUTED | The header is already in the default `Anthropic-Beta` list (`providers.go`), and `ttl: "1h"` needs no extra header. |
| 1.3.4 | Rolling breakpoint in the `messages[]` history | OPEN | Not started. Mind the rule: longer-TTL entries come before shorter-TTL ones, and the limit is four markers. |
| 1.4 | Compact accessibility-tree snapshots (ARIA/Playwright/MCP) in RTK | DONE | `c2c05f1f`, `89a5f392` (depth of the `/url` lines). Implemented from the description, without the MasterOfRoute reference. **Left:** measure with real snapshots from a session. |

### Findings outside the original plan

| ID | Finding | State | Evidence |
|---|---|---|---|
| E1 | RTK rewrote ordinary text as the output of a CLI when the command was not resolved: of 463 ordinary files read as tool output, 445 were claimed by a filter and the pipeline kept about 17% of the text | DONE | `f10a5b94` (two signs), `5e23108a` (case-sensitive selection patterns), `543622b6` and `0fa0b6bf` (`strongPatterns`), `54a22ad9` and `58c13834` (the `gh`, `git-branch` and `test-go` filters). Afterwards 1 file in 917 is still claimed (a changelog that quotes stack traces). Rules in [RTK_FILTERS.md](RTK_FILTERS.md). |
| E2 | RTK cost: about 208 ms per 73 KB tool output, 97% of it in `MatchFilter`, repeated every turn | DONE | `6d054e0c`, an LRU result cache: 0.41 ms on a repeat. |
| E3 | The first time a large result appears it still costs about 114 ms per 73 KB | OPEN | The cost is structural (137 patterns against the whole text). A possible path: a cheap pre-filter ahead of the regexes. |
| E4 | The DeepSeek catalog listed `deepseek-v4.1-flash`, which the API refuses (400) | DONE | `76f4074c`: the `deepseek` provider sends `deepseek-flash`, the alias of the latest Flash release. |

## Block 2: OpenCode Go and DeepSeek (prefix and savings)

| ID | Item | State | Evidence and what is left |
|---|---|---|---|
| 2.1 | Prefix affinity (Rendezvous/HRW) to keep the same `PrefixHash`/`prompt_cache_key` on the same connection | OPEN | `session_affinity.go` already ties a session to a connection, but there is no HRW over the prefix hash. **Do it together with 3.4**, because both decide which connection to use. |
| 2.2 | Prefix-safe compression and epoch compaction (`CompactContextByBudget`) | OPEN | `CompactContextByBudget` exists in `compact.go`; whether it is wired into the flow was not checked. RTK's fold is per message, which helps: an old message's result does not change. |
| 2.3 | Deduplicate `SemanticCache` and `PromptCache`; an L2 lexical cache | OPEN | Not started. |

## Block 3: combo resilience and controls

| ID | Item | State | Evidence and what is left |
|---|---|---|---|
| 3.1 | `prompt is too long` as a 400 instead of a 503 | REFUTED / adjusted | Measured with a fake upstream: for a single model and for a combo the upstream's 400 already reaches the client with the original message, and no path synthesizes a 503 for an oversize prompt. Only the refusal from an API key's context limit used different wording; adjusted in `c67b09a8`. |
| 3.2 | Failover for a truncated stream and for agent stalls | OPEN | Not started. |
| 3.3 | Progressive tool disclosure (`zeroOverlap`/BM25, opt-in) | OPEN | Not started; it should stay off by default so it does not touch the prefix. |
| 3.4 | Per-request pinning (`x-9router-connection`) and quota-free `count_tokens` | PARTIAL | `count_tokens` is already local (`HandleCountTokens`). The header does not exist; do it with 2.1. |
| 3.5 | Anti-avalanche failover guard and `RefusalGuard` | OPEN | Not started. |

## Suggested order

1. Prune the NVIDIA catalog (extinct models: 404, 410): small and independent.
2. Block 2.1 together with 3.4 (one design for choosing the connection).
3. Block 2.2 and 2.3, then the rest of Block 3.
4. 1.3.4 and 1.2b once the design is decided; E3 if the first-pass latency becomes a problem.

## How to measure without fooling yourself

- Measure before changing: three items of this plan (1.2a, 1.3.3, 3.1) had a wrong premise, and the real gain came from a finding nobody was looking for (E1).
- For RTK, use ordinary text as a negative corpus: if a CLI filter claims it, the filter is too loose.
- For the cache, compare `cached` in the usage log (`[usage] logged ... cached=N`) between two consecutive calls with the same prefix.
