package pricing

import (
	"math"
	"testing"
)

// Venice publishes per-token prices that do not match the DeepSeek family the
// bare model ids look like. Before providerPricing["venice"] existed, every
// Venice request fell through to the `deepseek-v*` pattern (0.14/0.28/0.0028)
// and the gateway recorded roughly 40% of the invoice. These rates come from a
// Venice billing export (2026-11-09 .. 2026-11-26, 600 requests, one API key):
// the per-SKU rate is amount/units on every row, so the fixture below is the
// invoice itself, not a model of it.
//
// Venice charges no separate cache-write or reasoning rate; those stay zero so
// CalculateCost falls back the way upstream does. venice-uncensored-1-2 showed
// no cache-input line at all in the export, so its cached rate stays unset
// rather than guessed.
func TestGetPricingForModel_VeniceInvoice(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  ModelPricing
	}{
		{
			name:  "uncensored 1017",
			model: "deepseek-v4-1-flash-1017-uncensored",
			want:  ModelPricing{InputPer1M: 0.33, OutputPer1M: 1.32, CachedPer1M: 0.00825},
		},
		{
			name:  "uncensored",
			model: "deepseek-v4-1-flash-uncensored",
			want:  ModelPricing{InputPer1M: 0.33, OutputPer1M: 1.32, CachedPer1M: 0.00825},
		},
		{
			name:  "venice uncensored 1.2",
			model: "venice-uncensored-1-2",
			want:  ModelPricing{InputPer1M: 0.2, OutputPer1M: 0.9},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := GetPricingForModel("venice", tt.model)
			if !found {
				t.Fatalf("GetPricingForModel(venice, %q) not found", tt.model)
			}
			if got != tt.want {
				t.Errorf("pricing = %+v, want %+v", got, tt.want)
			}
			if got.ReasoningPer1M != 0 || got.CacheCreationPer1M != 0 {
				t.Errorf("unpublished rates = %v/%v, want 0/0", got.ReasoningPer1M, got.CacheCreationPer1M)
			}
		})
	}
}

// The Venice override must stay scoped to the Venice gateway: the same model id
// served by a DeepSeek lane keeps its own, much cheaper, pattern price.
func TestGetPricingForModel_VeniceDoesNotLeak(t *testing.T) {
	tests := []struct {
		provider string
		model    string
		want     ModelPricing
	}{
		{
			provider: "deepseek",
			model:    "deepseek-v4-1-flash-1017-uncensored",
			want:     ModelPricing{InputPer1M: 0.14, OutputPer1M: 0.28, CachedPer1M: 0.0028, ReasoningPer1M: 0.28, CacheCreationPer1M: 0.14},
		},
		{
			provider: "opencode-go",
			model:    "deepseek-v4-1-flash-1017-uncensored",
			want:     ModelPricing{InputPer1M: 0.14, OutputPer1M: 0.28, CachedPer1M: 0.0028, ReasoningPer1M: 0.28, CacheCreationPer1M: 0.14},
		},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got, found := GetPricingForModel(tt.provider, tt.model)
			if !found {
				t.Fatalf("GetPricingForModel(%s, %q) not found", tt.provider, tt.model)
			}
			if got != tt.want {
				t.Errorf("pricing = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// One request straight out of the invoice: 42,796 uncached + 54,400 cached
// input against 883 output tokens cost $0.01573704. The recorded cost has to
// land on the invoice, not on the DeepSeek pattern that used to price it.
func TestCalculateCost_VeniceInvoiceRequest(t *testing.T) {
	got := EstimateCost("venice", "deepseek-v4-1-flash-uncensored", TokenCounts{
		PromptTokens:     97196,
		CachedTokens:     54400,
		CompletionTokens: 883,
	})
	const want = 0.01573704
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("EstimateCost = %.10f, want %.10f (invoice)", got, want)
	}

	// venice-uncensored-1-2: 1,399 input + 4 output = $0.0002834.
	got = EstimateCost("venice", "venice-uncensored-1-2", TokenCounts{
		PromptTokens:     1399,
		CompletionTokens: 4,
	})
	if math.Abs(got-0.0002834) > 1e-9 {
		t.Errorf("EstimateCost = %.10f, want %.10f (invoice)", got, 0.0002834)
	}
}
