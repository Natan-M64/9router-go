### feat(usage): a request that loses its prompt cache says why

- A large request (20 thousand prompt tokens or more) whose cache hit is under 80% now carries `cacheBreak` in its record: the hit percentage and a cause.
- The cause compares the fixed prefix with the previous request of the same conversation (the session id, or the connection and model when the client sent none): `system-changed` or `tools-changed` names what moved; `history-rewritten` means the prefix is the same and the history was cut (a prune or a compaction); `first-seen` means there is no earlier request to compare with.
- The memory keeps 512 conversations for 30 minutes at most. Nothing about the prompt is stored, only the hash fingerprints.
