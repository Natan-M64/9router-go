### feat(usage): the Quota Tracker reads the OpenCode Go plan

- An OpenCode Go connection now shows its rolling (5 h), weekly and monthly usage, the same three windows the Zen plan shows. Before, the provider was listed as quota-capable but had no reader, so its row stayed empty.
- The reader is shared with OpenCode Zen (one request, the same rows). The usage route is `https://opencode.ai/zen/go/v1/usage`, the route the Go plan publishes, which is not derived from the chat base URL.
