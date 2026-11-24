# Filtros do RTK — como são escolhidos e como escrever um

O RTK comprime a saída de ferramentas (`bash`, MCP, leitura de página…) antes de ela ir ao modelo.
Cada tipo de saída tem um filtro em `internal/tokensaver/filters/*.json` (embutido no binário). Um
filtro escolhido para o texto errado **apaga conteúdo**: ele mantém só as linhas que reconhece. Por
isso a escolha do filtro é a parte delicada, e é o assunto deste arquivo.

Código: `internal/tokensaver/filters.go` (`MatchFilter`, `LoadFilters`), `rtk.go`
(`compressTextDetailedWithCommand`), `rtk_cache.go`. Testes: `content_selection_test.go`,
`gh_filter_selection_test.go`.

## Como um filtro é escolhido

`MatchFilter(texto, comando, config)` segue esta ordem e para no primeiro acerto:

1. **Saída estruturada (JSON) nunca recebe filtro de prosa.** Se o texto é um documento JSON, retorna
   nenhum filtro: filtrar linha a linha deixaria um JSON inválido.
2. **Por comando.** Se o comando da ferramenta foi resolvido, o primeiro filtro habilitado (ordem de
   `priority` decrescente, depois `id`) cujo `match.commands` casa com ele é usado.
3. **Por conteúdo.** Só quando o comando não ajuda: filtros que declaram `commands` são pulados se o
   comando é conhecido; se o comando é vazio (resultado de MCP, página buscada, leitura via wrapper),
   todos os filtros habilitados entram. Um filtro é escolhido se **uma** de suas `strongPatterns`
   casa, **ou** se **pelo menos dois padrões distintos** de `match.patterns` casam.
4. **`generic-output`** é o último recurso.

A escolha por conteúdo é a única que pode errar sobre texto que não veio de um comando, e é a que
exige cuidado. O RTK escolhe o filtro **antes** de remover cores ANSI, então os padrões precisam
tolerar `\x1b[...m` e indentação quando a ferramenta imprime assim.

## Regras para escrever padrões de seleção

Elas existem porque o contrário já aconteceu: em 463 arquivos comuns lidos como saída de tool, 445
foram capturados por algum filtro e o pipeline mantinha ~17% do texto (0,0% no `CHANGELOG.md`).

- **Caixa exata.** Os padrões de `match.patterns` e `match.strongPatterns` são compilados só com
  `(?m)`. Uma CLI imprime `FAIL`, `ok`, `X` sempre na mesma caixa; casar sem diferenciar maiúsculas
  deixava a prosa ("fail the test…") passar por saída de `go test`. Se precisar mesmo de caixa
  indiferente, escreva `(?i)` no próprio padrão. As listas `includePatterns`, `dropPatterns`,
  `collapsePatterns` e `preserve.*` continuam com `(?im)`.
- **Um padrão solto não identifica um comando.** "failed", "INFO", "Already on", uma linha que parece
  caminho ou um bullet `* item` aparecem em texto comum. Padrões de seleção devem descrever a
  **forma** que o comando imprime: âncora no início da linha (`^`) e mais de uma palavra.
- **`strongPatterns` só para assinaturas inequívocas**, de várias palavras e ancoradas, que a prosa
  não imprime: `^diff --git \S+ \S+`, `^On branch \S+`, `^CONTAINER ID\s+IMAGE\s+COMMAND`. Não use
  uma frase que pode abrir uma linha de texto ("On branch protection rules…" já enganou
  `^On branch `, por isso o padrão exige o nome do branch sozinho na linha).
- **Filtro com um único padrão** nunca é escolhido só por conteúdo (precisaria de dois acertos); ele
  vale pelo comando, ou ganha uma `strongPattern`.
- **Fim de linha com CRLF:** use `\r?$`, não só `$`.
- **RE2 não tem lookahead.** `(?-i:X)` desliga a diferença de caixa só em um trecho.

## Como adicionar ou mudar um filtro

1. Edite o JSON do filtro. Campos: `id`, `label`, `description`, `category`, `priority`;
   `match` (`commands`, `patterns`, `strongPatterns`, `outputTypes`); `rules` (`stripAnsi`,
   `matchOutput`, `includePatterns`, `dropPatterns`, `collapsePatterns`, `deduplicate`, `maxLines`,
   `headLines`, `tailLines`, `onEmpty`); `preserve` (`errorPatterns`, `summaryPatterns`).
2. **Teste negativo.** Acrescente em `ordinaryTexts` (`content_selection_test.go`) um trecho de
   texto comum que contenha as palavras do seu padrão. O teste falha se o filtro o capturar.
3. **Teste positivo.** Acrescente a saída real do comando (inclusive com CRLF, se for plausível) em
   `TestContentSelectionStillFindsRealCommandOutput` ou em um teste próprio.
4. Rode `go test ./internal/tokensaver/` e, se mexeu no pipeline, `make ci-local`.
5. Faça o controle negativo: enfraqueça o padrão de propósito e confirme que o teste falha.

Sobre um corpus: antes de aceitar um padrão novo, rode `MatchFilter(texto, "", DefaultRTKConfig())` em
algumas centenas de arquivos do próprio repositório (Go, Markdown, Svelte, YAML). Qualquer filtro que
capture um arquivo comum está solto. O estado atual é 1 arquivo em 917 (um changelog que cita stack
traces, capturado por `error-stacktrace`).

## Cache de resultados

`CompressTextDetailedWithCommand` memoiza o resultado para textos de **4 KB ou mais** (LRU de até 1024
entradas e 64 MB de texto comprimido), com chave SHA-256 sobre `(comando, configuração, texto)`. Um
resultado de tool que volta a cada turno custa ~0,4 ms em vez de ~114 ms para 73 KB.

Isso só é correto porque a compressão é uma **função pura**. Não leia relógio, aleatoriedade, arquivo
nem variável de ambiente dentro do pipeline sem pôr o valor na chave. Todo campo novo de `RTKConfig`
já entra na chave por ser serializado; mantenha-o exportado e sem `json:"-"`.

## Snapshots de acessibilidade (`aria_snapshot.go`)

Texto com pelo menos cinco nós com `[ref=…]` é tratado como snapshot de página. Antes do truncamento
por linhas, o RTK remove folhas decorativas (`img`, `generic`, `presentation`, `none` **sem** nome,
texto nem atributo além do `ref`) até nenhum wrapper ficar vazio, e reduz a indentação pela metade
**só** se o menor degrau é exatamente 2, todas as colunas são pares e não há linha de continuação
indentada. A segunda condição torna a operação idempotente, o que importa porque o engine builtin do
Headroom reaplica a mesma dobra. Nós com `[cursor=pointer]`, nome, texto ou outro papel nunca saem.
