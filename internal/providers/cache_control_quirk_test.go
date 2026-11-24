package providers

import "testing"

// TestPreservesCacheControl pins the providers registered with upstream's
// preserveCacheControl quirk. The OpenAI filter strips the cache_control
// markers from the content blocks for every other provider, and these four
// gateways answer 400 when the markers are missing.
func TestPreservesCacheControl(t *testing.T) {
	registered := []string{"alicode", "alicode-intl", "alitp-intl", "alims-intl"}
	// openrouter is not in upstream's list: it keeps the markers on purpose,
	// because OpenRouter documents block-level cache_control for Anthropic
	// models and a stripped marker is a lost cache hit, not a 400.
	registered = append(registered, "openrouter")
	for _, provider := range registered {
		if !PreservesCacheControl(provider) {
			t.Errorf("%s must keep the cache_control markers", provider)
		}
	}
	for _, provider := range []string{"openai", "groq", "deepseek", "claude"} {
		if PreservesCacheControl(provider) {
			t.Errorf("%s must not keep the cache_control markers", provider)
		}
	}
}
