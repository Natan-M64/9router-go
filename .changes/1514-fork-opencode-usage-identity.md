### fix(usage): the OpenCode quota read identifies as the CLI and keeps one session per key

- The OpenCode Zen and Go usage reads send the OpenCode CLI's User-Agent and a session that stays the same for a given key, instead of the gateway's own identity. OpenCode flags background calls that look like a bare runtime.
