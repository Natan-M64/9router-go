# Roadmap de otimização — cache de prompt e tokens

Este arquivo é a fonte do plano "Bloco 1/2/3" citado em [STATUS.md](STATUS.md). O plano original
nasceu fora do repositório (auditoria de 2026-11-24); esta versão o substitui e já traz o que foi
feito, o que a medição refutou e o que continua aberto. Atualize aqui, não em cópias locais.

**Legenda.** `FEITO` = implementado, com teste e `make ci-local` verde. `REFUTADO` = a premissa não
se sustentou ao medir; não há o que corrigir. `ABERTO` = ainda não feito. `DECISÃO` = depende de uma
escolha de design do mantenedor. Os hashes são commits de `main`.

## Diagnóstico do tráfego real (600 chamadas amostradas)

| Métrica observada | Valor | Implicação |
|---|---|---|
| Provedores dominantes | `antigravity` (448), `opencode-go` (124), `claude` (25) | Foco em Antigravity, OpenCode Go (`deepseek-v4.1-flash`: 121 chamadas) e Claude |
| Tokens em cache | `antigravity` 44,9M, `opencode-go` 9,76M, `claude` 1,74M | Preservar o prefixo do KV cache é o maior multiplicador de economia |
| Tamanho das sessões | média 187,9 mensagens por requisição (máx. 647) | Qualquer mutação no prefixo destrói o cache |
| Chamadas com `tools[]` | 415 de 600 (média 28,9 tools, máx. 37) | Comprimir saída de tools sem tocar no schema de `tools[]` |

**Regra que vale para tudo abaixo:** `system` e `tools` ficam byte a byte iguais entre turnos. Teste a
estabilidade do prefixo ao mexer em `tokensaver` ou `headroom`.

## Bloco 1 — cache e tokens (ganhos imediatos)

| ID | Item | Estado | Evidência e o que falta |
|---|---|---|---|
| 1.1 | Auto-Clarity do Caveman não pode quebrar o prefixo | FEITO | `2961edbb`, corrigido em `c9467d0c` (a nota só vai na cauda real: numa conversa de tools a última mensagem é `tool`, e a nota não vai para uma mensagem `user` do meio). Teste de fiação em `8e9c2520`. **Limites:** corpos sem `messages` (Responses, Gemini) não recebem a nota; e no turno seguinte o cliente reenvia o histórico sem ela, então o cache perde dessa mensagem em diante (ainda melhor que alterar o `system`). |
| 1.2a | Remover a passada dupla RTK + Headroom builtin | REFUTADO | Medido em `applyTokenSavers`: 209,8 ms com e 209,8 ms sem o builtin, saída idêntica (a segunda passada trabalha sobre texto já dobrado). Não há o que remover. |
| 1.2b | Guard do Headroom por mensagem (`guard.go` rejeita a requisição inteira se houver uma tool protegida) | DECISÃO | Não alterado. Com o cache de resultados do RTK (`6d054e0c`) o custo de repetir caiu muito; falta decidir o desenho antes de mexer. |
| 1.3.1 | `PreserveCacheControl` no OpenRouter | FEITO | `97270d01` (divergência deliberada do upstream: a documentação do OpenRouter suporta `cache_control` em blocos de conteúdo) e limite de 4 marcadores em `960981c4`, que também protege os 4 gateways Alibaba. |
| 1.3.2 | Âncora de cache também para chave de API Claude; `system` string vira bloco | FEITO | `2961edbb` e `c9467d0c`, **só para upstream Anthropic**. Provedores de terceiros no formato Messages ficam como estavam: não se sabe se aceitam `ttl` ou `system` em array. |
| 1.3.3 | Header beta `extended-cache-ttl` quando o TTL é `1h` | REFUTADO | O header já está na lista padrão de `Anthropic-Beta` (`providers.go`), e `ttl: "1h"` não precisa de header extra. |
| 1.3.4 | Breakpoint rolante no histórico `messages[]` | ABERTO | Não iniciado. Cuidado com a regra: entradas de TTL maior vêm antes das de TTL menor, e o máximo é 4 marcadores. |
| 1.4 | Compactar snapshots de acessibilidade (ARIA/Playwright/MCP) no RTK | FEITO | `c2c05f1f`, `89a5f392` (profundidade das linhas `/url`). Implementado a partir da descrição, sem a referência do MasterOfRoute. **Falta:** medir com snapshots reais de uma sessão. |

### Achados fora do plano original

| ID | Achado | Estado | Evidência |
|---|---|---|---|
| E1 | O RTK reescrevia texto comum como saída de um CLI quando o comando não era resolvido: de 463 arquivos comuns lidos como saída de tool, 445 eram capturados por um filtro e o pipeline mantinha ~17% do texto | FEITO | `f10a5b94` (dois sinais), `5e23108a` (padrões de seleção case-sensitive), `543622b6` e `0fa0b6bf` (`strongPatterns`), `54a22ad9` e `58c13834` (filtros `gh`, `git-branch`, `test-go`). Depois: 1 em 917 arquivos ainda é capturado (changelog que cita stack traces). Regras em [RTK_FILTERS.md](RTK_FILTERS.md). |
| E2 | Custo do RTK: ~208 ms por saída de tool de 73 KB, 97% em `MatchFilter`, repetido a cada turno | FEITO | `6d054e0c`, cache LRU de resultados: 0,41 ms ao repetir. |
| E3 | A primeira vez que um resultado grande aparece ainda custa ~114 ms por ~73 KB | ABERTO | O custo é estrutural (137 padrões contra o texto inteiro). Caminho possível: pré-filtro barato antes dos regex. |
| E4 | O catálogo DeepSeek listava `deepseek-v4.1-flash`, que a API recusa (400) | FEITO | `76f4074c`: o provedor `deepseek` envia `deepseek-flash`, o alias da Flash mais recente. |

## Bloco 2 — OpenCode Go e DeepSeek (prefixo e economia)

| ID | Item | Estado | Evidência e o que falta |
|---|---|---|---|
| 2.1 | Afinidade por prefixo (Rendezvous/HRW) para manter o mesmo `PrefixHash`/`prompt_cache_key` na mesma conexão | ABERTO | `session_affinity.go` já liga sessão a conexão, mas não há HRW pelo hash do prefixo. **Fazer junto com 3.4**, porque os dois decidem qual conexão usar. |
| 2.2 | Compressão prefix-safe e compactação por épocas (`CompactContextByBudget`) | ABERTO | `CompactContextByBudget` existe em `compact.go`; não se verificou se está ligado ao fluxo. O dobramento do RTK é por mensagem, o que ajuda: o resultado de uma mensagem antiga não muda. |
| 2.3 | Desduplicar `SemanticCache` e `PromptCache`; cache lexical L2 | ABERTO | Não iniciado. |

## Bloco 3 — resiliência de combos e controles

| ID | Item | Estado | Evidência e o que falta |
|---|---|---|---|
| 3.1 | `prompt is too long` como 400 em vez de 503 | REFUTADO / ajustado | Medido com upstream falso: para modelo simples e para combo o 400 do upstream já chega ao cliente com a mensagem original, e nenhum caminho sintetiza 503 por prompt grande. Só a recusa pelo limite de contexto da chave usava outra redação; ajustada em `c67b09a8`. |
| 3.2 | Failover de stream truncado e de estagnação do agente | ABERTO | Não iniciado. |
| 3.3 | Divulgação progressiva de tools (`zeroOverlap`/BM25, opt-in) | ABERTO | Não iniciado; deve ficar desligado por padrão para não mexer no prefixo. |
| 3.4 | Pinning por requisição (`x-9router-connection`) e `count_tokens` sem cota | PARCIAL | `count_tokens` já é local (`HandleCountTokens`). O header não existe; fazer com 2.1. |
| 3.5 | Anti-avalanche no failover e `RefusalGuard` | ABERTO | Não iniciado. |

## Ordem sugerida

1. Poda do catálogo NVIDIA (modelos extintos: 404, 410) — pequeno e independente.
2. Bloco 2.1 junto com 3.4 (um desenho só para escolher a conexão).
3. Bloco 2.2 e 2.3, depois o restante do Bloco 3.
4. 1.3.4 e 1.2b quando houver decisão de desenho; E3 se a latência da primeira passada incomodar.

## Como medir sem se enganar

- Meça antes de mudar: três itens deste plano (1.2a, 1.3.3, 3.1) estavam errados na premissa, e o ganho
  real veio de um achado que ninguém procurava (E1).
- Para o RTK, use texto comum como corpus negativo: se um filtro de CLI o captura, o filtro está solto.
- Para cache, compare `cached` no log de uso (`[usage] logged ... cached=N`) entre duas chamadas
  seguidas com o mesmo prefixo.
