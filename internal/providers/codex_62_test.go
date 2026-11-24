package providers

import (
	"slices"
	"testing"
)

func TestCodex62ModelsParity(t *testing.T) {
	models := GetProviderModels("codex")
	for _, want := range []string{"gpt-6.2-sol", "gpt-6.2-sol[1m]", "gpt-6.2-astra", "gpt-6.2-astra[1m]", "gpt-6.1-terra"} {
		if !slices.Contains(models, want) {
			t.Errorf("codex missing upstream model %q", want)
		}
	}

	cxModels := GetProviderModels("cx")
	for _, want := range []string{"gpt-6.2-sol", "gpt-6.2-sol[1m]", "gpt-6.2-astra", "gpt-6.2-astra[1m]", "gpt-6.1-terra"} {
		if !slices.Contains(cxModels, want) {
			t.Errorf("cx alias missing upstream model %q", want)
		}
	}

	cases := []struct {
		model  string
		wantCW int
	}{
		{"gpt-6.2-sol", 272000},
		{"gpt-6.2-astra", 272000},
		{"gpt-6.2-sol[1m]", codexExtendedContextWindow},
		{"gpt-6.2-astra[1m]", codexExtendedContextWindow},
		{"gpt-6.1-terra", 500000},
	}
	for _, tc := range cases {
		caps := GetCapabilitiesDetailForModel("codex", tc.model)
		if caps.ContextWindow != tc.wantCW {
			t.Errorf("codex/%s ContextWindow = %d, want %d", tc.model, caps.ContextWindow, tc.wantCW)
		}
	}

	if got := CodexUpstreamModelID("gpt-6.2-sol[1m]"); got != "gpt-6.2-sol" {
		t.Errorf("CodexUpstreamModelID(gpt-6.2-sol[1m]) = %q, want gpt-6.2-sol", got)
	}
}
