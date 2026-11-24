### feat(usage): a request record carries the stop reason and the output limit the client asked for

- The stored record of a successful request now has `finishReason` (the last `finish_reason` the client received, for any provider) and `maxTokens` (the client's `max_tokens` or `max_completion_tokens`, absent when it set none). A reply that ends with `length` can now be told apart from a normal stop without reading the console log.
- The stop reason is read from the captured response, which keeps the first 100 KB: for a longer reply it is left out rather than guessed. The limit the gateway raises for a thinking model is not recorded, only what the client asked for.
