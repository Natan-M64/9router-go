### feat(providers): Codex 6.2 catalog sync and per-model thinking selector

- Codex catalog gains `gpt-6.2-sol`, `gpt-6.2-sol[1m]`, `gpt-6.2-astra`, `gpt-6.2-astra[1m]` and `gpt-6.1-terra`, with a drift check against upstream. The fork's lists are the upstream's plus `gpt-6.1-terra`.
- Provider model cards show each model's `thinkingLevels` as badges, and the combo editor has a per-model `Thinking:` selector.
