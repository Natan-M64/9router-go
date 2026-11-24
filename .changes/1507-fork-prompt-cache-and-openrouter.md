### fix(cache): keep the prompt prefix stable and cache for more Claude requests

- The Caveman auto-clarity bypass no longer changes the system prompt. It appends a short note to the last user message instead, and only when that message is the true tail of the request, so the cached prefix stays byte-identical. Bodies with no `messages` array get no note.
- The Claude cache breakpoints are anchored for API-key requests to Anthropic too, not only OAuth ones, and a string `system` is promoted to a text block so it can carry `cache_control`. Other providers that speak the Messages format are left as they were.
- OpenRouter keeps the `cache_control` markers a client sends, which it documents for Anthropic models. At most the last four markers are forwarded, since a fifth is a 400; the same cap now protects the four Alibaba gateways that already kept markers.
- A request refused by an API key's context limit still returns 400, and the message now opens with `prompt is too long: N tokens > M maximum`, the phrase Claude Code matches to compact a conversation.
