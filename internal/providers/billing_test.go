package providers

import "testing"

func TestBillingModeForProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     BillingMode
	}{
		{name: "api key provider is metered", provider: "venice", want: BillingMetered},
		{name: "api key provider without authType falls back to category", provider: "deepseek", want: BillingMetered},
		{name: "oauth subscription is included", provider: "codex", want: BillingIncluded},
		{name: "oauth quota provider is included", provider: "antigravity", want: BillingIncluded},
		{name: "free provider is included", provider: "gemini-cli", want: BillingIncluded},
		{name: "alias resolves to the canonical entry", provider: "cx", want: BillingIncluded},
		{
			name:     "free tier listing with an api key is metered",
			provider: "openrouter",
			want:     BillingMetered,
		},
		{
			name:     "a fixed-price plan is not a per-token invoice",
			provider: "opencode-go",
			want:     BillingIncluded,
		},
		{
			name:     "the plan override follows an alias",
			provider: "ocg",
			want:     BillingIncluded,
		},
		{name: "custom compatible endpoint is unknown", provider: "openai-compatible-local", want: BillingUnknown},
		{name: "unregistered provider is unknown", provider: "no-such-provider", want: BillingUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BillingModeForProvider(tt.provider); got != tt.want {
				t.Errorf("BillingModeForProvider(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}
