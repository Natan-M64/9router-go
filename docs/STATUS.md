# Status

Updated 2026-11-24 · Fork `Natan-M64/9router-go` (upstream `luqman-v1/9router-go` / `decolua/9router`)
Current milestone: **Estabilização do RTK e do cache de prompt** — `main` local `08e853d4` (Bloco 1, OpenRouter, snapshots MCP, filtro `gh`); branch `perf/rtk-filter-selection` revisada por um agente independente (sem bloqueadores; achados tratados), aguardando OK para a `main`. Nada disso está no fork ainda.
Last proof: [journal/2026-11.md](journal/2026-11.md).
Limits of this file: ≤120 lines, ≤8 KB, ≤2 cards in Now.

## Now

| ID | Expected result | State |
|---|---|---|
| O1 | **Seleção de filtros do RTK**: conteúdo exige 2 sinais (ou `strongPatterns`), padrões de seleção case-sensitive, cache de resultados (0,41 ms vs ~114 ms ao repetir). Medido: 445 de 463 arquivos comuns eram capturados por filtros de CLI, mantendo ~17% do texto | Implementado e revisado em `perf/rtk-filter-selection`; desvia do upstream (caixa e limiar), precisa de OK para ir à `main`. |
| O2 | **Prova do MVP**: smoke com upstream falso feito (chat 200, stream, 401 sem chave); falta provedor real e o job `integration` do CI | Partial. |

## Next (in order)

| ID | Scope | Exit / dependency |
|---|---|---|
| O3 | 3.1 `context_length_exceeded` 400 em vez de 503 (auto-compact do Claude Code); 3.4 header `x-9router-connection` (`count_tokens` já é local) | After O1. |
| O4 | **Bloco 2**: afinidade por hash do prefixo (HRW; `session_affinity.go` já existe), compressão prefix-safe, desduplicar `SemanticCache`/`PromptCache` | After O3. |
| O5 | **Bloco 3 restante**: failover de stream truncado/stall, Progressive Tool Disclosure, Anti-Avalanche/`RefusalGuard` | After O4. |
| O6 | Guard do Headroom por mensagem (1.2b) e custo da 1ª passada do RTK (~114 ms por 73 KB novos) | Precisa de decisão de design. |
| O7 | Issue #1 follow-up (flags de runtime em `settings`) | Low priority. |

## Risks and open findings

| ID | Item | Action / who unblocks |
|---|---|---|
| R1 | Sessões longas reais (`opencode-go/deepseek-v4.1-flash`, `antigravity`) têm média de 188 msgs e ~29 tools; qualquer alteração no prefixo (`system`/`tools`) zera o KV cache | Sempre testar estabilidade de prefixo ao mexer em `tokensaver`/`headroom`. |
| R3 | Nota de clareza (caveman) não é aplicada em corpos sem `messages` (Responses/Gemini); a fiação de `fallback.go` não tem teste dedicado | Cobrir `applyTokenSavers` com teste; decidir sobre esses formatos. |
| R4 | O filtro de seleção por conteúdo ainda captura 1 de 917 arquivos (changelog que cita stack traces, via `error-stacktrace`) e a auditoria cobriu só arquivos do repositório, não tráfego real | Revisar em uma amostra real de tool results. |
| R5 | Passada dupla RTK + Headroom builtin: medida sem custo (209,8 ms com e sem); o item 1.2a do roadmap não se sustenta | Remover do roadmap. |
| R2 | Pacote GHCR `ghcr.io/natan-m64/9router-go` pode estar privado por padrão no GitHub Packages | Usuário altera visibilidade para Public se quiser `docker pull` sem login. |

## Done in the last 7 days

- **GHCR Multi-Arch + Stack/Compose/Env mínimo (`cb46f6d1`, Issue #1)**: workflow nativo `amd64` + `ubuntu-24.04-arm`, non-root `9router` + `CMD ["serve"]`, `hadolint` + `actionlint`, artefato compartilhado `web/dist` no CI.
- **Modo On-Demand macOS/Linux (`c5baa784`, `8ed820dc`)**: `--idle-timeout` / `IDLE_TIMEOUT`, `9router-go maintain`, Socket Activation (`launchd` + `systemd`), `scripts/macos-on-demand.sh`.
- **Catálogo Codex & Thinking UX (`eb29c708`, `f1166342`, `8ed820dc`)**: `gpt-6.2-sol`, `gpt-6.2-sol[1m]`, `gpt-6.2-astra`, `gpt-6.2-astra[1m]`, `gpt-6.1-terra`, check semanal upstream, badges de `thinkingLevels` nos cards de provedor e seletor `Thinking:` por modelo em `CreateComboModal`.
- **Combos JSON & OmniRoute Import/Export (`41ea97db`)**: importação `9router-combos-v1` e `omniroute-combos-v1` com mapeamento automático/assistido de siglas de provedor.

## Verified (2026-11-24)

- `make ci-local` exit 0 (77 pacotes) na branch `perf/rtk-filter-selection`; smoke de instalação limpa; daemon start/status/stop no macOS 27; chat completion e stream por upstream falso; varredura Playwright de 34 rotas sem erro de console; negative controls dos testes de gh, cap de marcadores, `/url` e limiar de seleção.
- Não verificado: provedor real, actionlint, snapshots reais do Playwright, `jailbreak`/`plans` redirecionando para `endpoint` (intencional?).
