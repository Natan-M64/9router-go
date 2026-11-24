# Status

Updated 2026-11-24 · Fork `Natan-M64/9router-go` (upstream `luqman-v1/9router-go` / `decolua/9router`)
Current milestone: **Bloco 1 — Correções de Prompt Cache, Headroom e RTK** (após entrega de GHCR multi-arch, modo on-demand e import/export de combos).
Last proof: [journal/2026-11.md](journal/2026-11.md).
Limits of this file: ≤120 lines, ≤8 KB, ≤2 cards in Now.

## Now

| ID | Expected result | State |
|---|---|---|
| O1 | **Bloco 1 (Cache & Token Quick Wins)**: manter `CavemanAutoClarity` prefix-safe (cauda), corrigir `Headroom` `guard.go` por mensagem + remover duplo RTK, completar `AnchorClaudeCacheWithTTL` + `PreserveCacheControl` no OpenRouter e compactar snapshots ARIA/MCP no RTK | Ready to start. |

## Next (in order)

| ID | Scope | Exit / dependency |
|---|---|---|
| O2 | **Bloco 2 (OpenCode Go & DeepSeek Cache/Tokens)**: Prompt-Cache Affinity (Rendezvous Hashing), compressão prefix-safe + `CompactContextByBudget` em épocas, desduplicação `SemanticCache`/`PromptCache` + L2 lexical com `numericSignature` | After O1. |
| O3 | **Bloco 3 (Resiliência de Combos & Controles)**: Actionable Budget Exhaustion (`400` para auto-compact), Truncated-Stream & Agentic-Stall failover, Progressive Tool Disclosure (`BM25`/`zeroOverlap`), `x-9router-connection`, `count_tokens` local, Anti-Avalanche / `RefusalGuard` | After O2. |
| O4 | **Issue #1 (follow-up completo)**: mover flags de runtime restantes (`STRICT_SSL`, `ALLOW_PRIVATE_NETWORKING`, etc.) para tabela `settings` se necessário no Dashboard | Low priority; bootstrap mínimo já aplicado (`cb46f6d1`). |

## Risks and open findings

| ID | Item | Action / who unblocks |
|---|---|---|
| R1 | Sessões longas reais (`opencode-go/deepseek-v4.1-flash`, `antigravity`) têm média de 188 msgs e ~29 tools; qualquer alteração no prefixo (`system`/`tools`) zera o KV cache | Sempre testar estabilidade de prefixo ao mexer em `tokensaver`/`headroom`. |
| R2 | Pacote GHCR `ghcr.io/natan-m64/9router-go` pode estar privado por padrão no GitHub Packages | Usuário altera visibilidade para Public se quiser `docker pull` sem login. |

## Done in the last 7 days

- **GHCR Multi-Arch + Stack/Compose/Env mínimo (`cb46f6d1`, Issue #1)**: workflow nativo `amd64` + `ubuntu-24.04-arm`, non-root `9router` + `CMD ["serve"]`, `hadolint` + `actionlint`, artefato compartilhado `web/dist` no CI.
- **Modo On-Demand macOS/Linux (`c5baa784`, `8ed820dc`)**: `--idle-timeout` / `IDLE_TIMEOUT`, `9router-go maintain`, Socket Activation (`launchd` + `systemd`), `scripts/macos-on-demand.sh`.
- **Catálogo Codex & Thinking UX (`eb29c708`, `f1166342`, `8ed820dc`)**: `gpt-6.2-sol`, `gpt-6.2-sol[1m]`, `gpt-6.2-astra`, `gpt-6.2-astra[1m]`, `gpt-6.1-terra`, check semanal upstream, badges de `thinkingLevels` nos cards de provedor e seletor `Thinking:` por modelo em `CreateComboModal`.
- **Combos JSON & OmniRoute Import/Export (`41ea97db`)**: importação `9router-combos-v1` e `omniroute-combos-v1` com mapeamento automático/assistido de siglas de provedor.
