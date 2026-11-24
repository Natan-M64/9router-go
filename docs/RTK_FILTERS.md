# RTK filters: how they are chosen and how to write one

RTK compresses tool output (`bash`, MCP, fetched pages, ...) before it goes to the model. Each kind of output has a filter in `internal/tokensaver/filters/*.json`, embedded in the binary. A filter chosen for the wrong text **deletes content**: it keeps only the lines it recognises. That is why choosing the filter is the delicate part, and it is what this file is about.

Code: `internal/tokensaver/filters.go` (`MatchFilter`, `LoadFilters`), `rtk.go` (`compressTextDetailedWithCommand`), `rtk_cache.go`. Tests: `content_selection_test.go`, `gh_filter_selection_test.go`.

## How a filter is chosen

`MatchFilter(text, command, config)` follows this order and stops at the first hit:

1. **Structured output (JSON) never gets a prose filter.** If the text is a JSON document, no filter is returned: filtering line by line would leave invalid JSON behind.
2. **By command.** If the tool's command was resolved, the first enabled filter whose `match.commands` matches it is used (ordered by `priority`, highest first, then by `id`).
3. **By content.** Only when the command does not settle it. Filters that declare `commands` are skipped when the command is known. When the command is empty (an MCP result, a fetched page, a read through a wrapper), every enabled filter is a candidate. A filter is chosen if **one** of its `strongPatterns` matches, **or** if **at least two distinct patterns** of `match.patterns` match.
4. **`generic-output`** is the last resort.

The published wiki (`wiki/token-saver.md`, "What it protects") lists "the filter is chosen by the **command** that produced the output, not by its content" as an invariant. Selection by content exists as a fallback for results whose command cannot be resolved (MCP results, fetched pages), and the rules below are what keep it from breaking that invariant for ordinary text.

Selection by content is the only path that can go wrong on text that never came from a command, and it is the one that needs care. RTK picks the filter **before** it strips ANSI colours, so patterns have to tolerate `\x1b[...m` and indentation when the tool prints them.

## Rules for writing selection patterns

They exist because the opposite already happened: on 463 ordinary files read as tool output, 445 were claimed by some filter and the pipeline kept about 17% of the text (0.0% for `CHANGELOG.md`).

- **Exact case.** `match.patterns` and `match.strongPatterns` are compiled with `(?m)` only. A CLI prints `FAIL`, `ok` and `X` in one fixed case, and matching case-insensitively let prose ("fail the test...") pass for `go test` output. If you really need case folding, write `(?i)` in the pattern itself. The `includePatterns`, `dropPatterns`, `collapsePatterns` and `preserve.*` lists keep the shared `(?im)`.
- **A loose pattern does not identify a command.** "failed", "INFO", "Already on", a line that looks like a path or a `* item` bullet all occur in ordinary text. Selection patterns should describe the **shape** the command prints: anchored at the start of the line, and more than one word.
- **`strongPatterns` are for unmistakable signatures only**: multi-word, anchored lines that prose does not print, such as `^diff --git \S+ \S+`, `^On branch \S+` or `^CONTAINER ID\s+IMAGE\s+COMMAND`. Do not use a phrase that can open a line of text ("On branch protection rules..." fooled `^On branch `, which is why the pattern now requires the branch name alone on the line).
- **A filter with a single pattern** is never chosen by content alone (it would need two hits). It works through its command, or it gets a strong pattern.
- **CRLF line endings:** end anchors are `\r?$`, not just `$`.
- **RE2 has no lookahead.** `(?-i:X)` turns off case folding for one part of a pattern only.

## Adding or changing a filter

1. Edit the filter's JSON. Fields: `id`, `label`, `description`, `category`, `priority`; `match` (`commands`, `patterns`, `strongPatterns`, `outputTypes`); `rules` (`stripAnsi`, `matchOutput`, `includePatterns`, `dropPatterns`, `collapsePatterns`, `deduplicate`, `maxLines`, `headLines`, `tailLines`, `onEmpty`); `preserve` (`errorPatterns`, `summaryPatterns`).
2. **Negative test.** Add to `ordinaryTexts` (`content_selection_test.go`) a piece of ordinary text that contains the words of your pattern. The test fails if the filter claims it.
3. **Positive test.** Add the command's real output (with CRLF too, if plausible) to `TestContentSelectionStillFindsRealCommandOutput` or to a test of its own.
4. Run `go test ./internal/tokensaver/`, and `make ci-local` if you touched the pipeline.
5. Run the negative control: weaken the pattern on purpose and confirm the test fails.

About a corpus: before accepting a new pattern, run `MatchFilter(text, "", DefaultRTKConfig())` over a few hundred files of this repository (Go, Markdown, Svelte, YAML). Any filter that claims an ordinary file is too loose. The current state is 1 file in 917 (a changelog that quotes stack traces, claimed by `error-stacktrace`).

## Result cache

`CompressTextDetailedWithCommand` memoizes the result for texts of **4 KB or more** (an LRU of up to 1024 entries and 64 MB of compressed text), keyed by a SHA-256 over `(command, configuration, text)`. A tool result that comes back every turn costs about 0.4 ms instead of about 114 ms for 73 KB.

This is only correct because compression is a **pure function**. Do not read a clock, a random source, a file or an environment variable inside the pipeline without putting the value in the key. Every new `RTKConfig` field already enters the key because it is serialised; keep it exported and without `json:"-"`.

## Accessibility snapshots (`aria_snapshot.go`)

A text with at least five nodes carrying `[ref=...]` is treated as a page snapshot. Before the line-based truncation, RTK removes decorative leaves (`img`, `generic`, `presentation`, `none` with **no** name, text or attribute other than the `ref`) until no wrapper is left empty, and it halves the indentation **only** if the smallest step is exactly 2, every column is even and there is no indented continuation line. The last condition makes the operation idempotent, which matters because the built-in Headroom engine reruns the same fold. Nodes with `[cursor=pointer]`, a name, text or another role are never removed.
