package providers

import "testing"

// TestPreservesCacheControl pins the providers registered with upstream's
// preserveCacheControl quirk. The OpenAI filter strips the cache_control
// markers from the content blocks for every other provider, and these four
// gateways answer 400 when the markers are missing.
func TestPreservesCacheControl(t *testing.T) {
	registered := []string{"alicode", "alicode-intl", "alitp-intl", "alims-intl"}
	// openrouter is not here: it keeps the markers only for anthropic/* models,
	// decided per model by translator.PreservesOpenRouterAnthropicCacheControl.
	for _, provider := range registered {
		if !PreservesCacheControl(provider) {
			t.Errorf("%s must keep the cache_control markers", provider)
		}
	}
	for _, provider := range []string{"openai", "groq", "deepseek", "claude", "openrouter"} {
		if PreservesCacheControl(provider) {
			t.Errorf("%s must not keep the cache_control markers", provider)
		}
	}
}
