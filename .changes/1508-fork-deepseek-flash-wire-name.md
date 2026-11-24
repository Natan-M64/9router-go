### fix(deepseek): send the wire id the DeepSeek API serves for v4.1-flash

- `deepseek-v4.1-flash` is valid on the opencode gateways but not on the DeepSeek API, which answers 400 and lists `deepseek-flash` and `deepseek-v4-pro` as the supported names. The deepseek provider now sends `deepseek-flash`, DeepSeek's alias for its latest Flash release (V4.1 Flash today). Only the model changes: the client's thinking and `reasoning_effort` are left as sent, and other providers keep the original id.
