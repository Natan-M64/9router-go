### Billing and Venice quota
- Separate provider-metered spend from list-price estimates so usage analytics report what providers charged without conflating it with estimated model cost; classify Xiaomi Token Plan as included while ordinary Xiaomi MiMo API keys remain metered.
- Add invoice-aligned Venice model pricing, capture balance headers returned by inference, and optionally probe billing balance with an ADMIN key.
- Cache dashboard usage responses for 70 seconds and hydrate the view immediately; report prompt-cache breaks with their divergence index and estimated cost.
- Verified with `go test ./... -count=1`, `make test-integration`, frontend unit tests, browser e2e, Svelte ratchet, Oxlint, and production frontend build. The build still reports existing Svelte warnings.
