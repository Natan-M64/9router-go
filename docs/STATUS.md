# Status

Updated 2026-11-24 · Fork `Natan-M64/9router-go` (upstream `luqman-v1/9router-go` / `decolua/9router`)
Current milestone: **Fork sincronizado com o upstream (0 commits atrás) e estável** — RTK/cache de prompt, DeepSeek e sinal de compactação corrigidos. Próximo: catálogo NVIDIA, depois Bloco 2 (junto com o header `x-9router-connection`).
Last proof: [journal/2026-11.md](journal/2026-11.md).
Limits of this file: ≤120 lines, ≤8 KB, ≤2 cards in Now.

## Now

| ID | Expected result | State |
|---|---|---|
| O1 | **Catálogo NVIDIA com modelos extintos**: `google/gemma-3-4b-it` dá 404, `meta/llama-3.1-8b-instruct` dá 410 "Gone" (fim de vida na NVIDIA) e `google/gemma-4-31b-it` estourou 90 s; o gateway lista os três | Found; falta sondar a lista viva da NVIDIA e podar o catálogo. |
| O2 | **Prova do MVP, restante**: Claude, Codex, OpenRouter, Venice, DeepSeek e opencode-go responderam 200 (normal e stream); falta gemini, groq e kilocode (status `unavailable` na instância) e o job `integration` do CI | Partial. |

## Next (in order)

| ID | Scope | Exit / dependency |
|---|---|---|
| O3 | **Bloco 2**: afinidade por hash do prefixo (HRW; `session_affinity.go` já existe) junto com o header `x-9router-connection` (3.4), porque os dois decidem a conexão; compressão prefix-safe; desduplicar `SemanticCache`/`PromptCache` | After O1. |
| O4 | **Bloco 3 restante**: failover de stream truncado/stall, Progressive Tool Disclosure, Anti-Avalanche/`RefusalGuard` | After O3. |
| O5 | Guard do Headroom por mensagem (1.2b) e custo da 1ª passada do RTK (~114 ms por 73 KB novos) | Precisa de decisão de design. |
| O6 | Issue #1 follow-up (flags de runtime em `settings`) | Low priority. |

## Risks and open findings

| ID | Item | Action / who unblocks |
|---|---|---|
| R1 | Sessões longas reais (`opencode-go/deepseek-v4.1-flash`, `antigravity`) têm média de 188 msgs e ~29 tools; qualquer alteração no prefixo (`system`/`tools`) zera o KV cache | Sempre testar estabilidade de prefixo ao mexer em `tokensaver`/`headroom`. |
| R3 | Nota de clareza (caveman) não é aplicada em corpos sem `messages` (Responses/Gemini); a fiação de `fallback.go` não tem teste dedicado | Cobrir `applyTokenSavers` com teste; decidir sobre esses formatos. |
| R4 | O filtro de seleção por conteúdo ainda captura 1 de 917 arquivos (changelog que cita stack traces, via `error-stacktrace`) e a auditoria cobriu só arquivos do repositório, não tráfego real | Revisar em uma amostra real de tool results. |
| R6 | Cada requisição da instância carrega ~3000 tokens do prompt do recurso `jailbreak` (13120 bytes, configuração do usuário); a 2ª chamada reaproveitou 3072 desses tokens do cache | Conferir se é o desejado; a resposta de teste trouxe um sufixo estranho ("China is good...") de origem não verificada. |
| R7 | O roadmap 3.1 (`prompt is too long` → 400 em vez de 503) já era verdade: upstream (modelo simples e combo) devolve 400 com a mensagem original; só a recusa por limite de contexto da chave usava outra redação (ajustada) | Nada a fazer; tirar 3.1 do roadmap. |
| R8 | Catálogo DeepSeek: `deepseek-flash` é o alias da Flash mais recente (hoje V4.1-Flash-1017; docs 2026-09-10 e 2026-10-17), então o mapeamento de `v4.1-flash` está certo. Já `deepseek-v4-flash` e `-vision-exp` são modelos aposentados que a API "roteia temporariamente" para a V4.1 Flash (2026-09-10), e `deepseek-chat`/`reasoner` tinham aposentadoria em 2026-07-24 mas ainda responderam 200 em 2026-11-24 | Podar do catálogo só quando a API parar de aceitar; reconferir `https://api-docs.deepseek.com/updates.md`. |
| R5 | Passada dupla RTK + Headroom builtin: medida sem custo (209,8 ms com e sem); o item 1.2a do roadmap não se sustenta | Remover do roadmap. |
| R2 | Pacote GHCR `ghcr.io/natan-m64/9router-go` pode estar privado por padrão no GitHub Packages | Usuário altera visibilidade para Public se quiser `docker pull` sem login. |

## Done in the last 7 days

- **GHCR Multi-Arch + Stack/Compose/Env mínimo (`cb46f6d1`, Issue #1)**: workflow nativo `amd64` + `ubuntu-24.04-arm`, non-root `9router` + `CMD ["serve"]`, `hadolint` + `actionlint`, artefato compartilhado `web/dist` no CI.
- **Modo On-Demand macOS/Linux (`c5baa784`, `8ed820dc`)**: `--idle-timeout` / `IDLE_TIMEOUT`, `9router-go maintain`, Socket Activation (`launchd` + `systemd`), `scripts/macos-on-demand.sh`.
- **Catálogo Codex & Thinking UX (`eb29c708`, `f1166342`, `8ed820dc`)**: `gpt-6.2-sol`, `gpt-6.2-sol[1m]`, `gpt-6.2-astra`, `gpt-6.2-astra[1m]`, `gpt-6.1-terra`, check semanal upstream, badges de `thinkingLevels` nos cards de provedor e seletor `Thinking:` por modelo em `CreateComboModal`.
- **Combos JSON & OmniRoute Import/Export (`41ea97db`)**: importação `9router-combos-v1` e `omniroute-combos-v1` com mapeamento automático/assistido de siglas de provedor.

## Verified (2026-11-24)

- `make ci-local` exit 0 (77 pacotes) e `actionlint` exit 0; CI do push de `abe799ab` verde.
- Upstream sincronizado (4 commits, 4 arquivos em conflito resolvidos); `ds/deepseek-v4.1-flash` passou de 400 para 200 na API real (servido como `deepseek-flash`, confirmado pela documentação oficial), e `ocg/deepseek-v4.1-flash` segue igual.
- Chat e stream reais pela instância do usuário: opencode-go, DeepSeek, Claude (haiku), Codex (astra), OpenRouter (`:free`, com `max_tokens` 256) e Venice.
- Não verificado: gemini/groq/kilocode, snapshots reais do Playwright, `jailbreak`/`plans` redirecionando para `endpoint`; o `node` do Homebrew voltou ao normal (v26.13.1) após a atualização.
