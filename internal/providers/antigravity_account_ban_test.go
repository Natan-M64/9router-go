package providers

import (
	"net/http"
	"strings"
	"testing"
)

const tosBody = `{"error":{"code":403,"message":"This service has been disabled in this account for violation of Terms of Service. Please submit an appeal to continue using this product.","status":"PERMISSION_DENIED","details":[{"reason":"TOS_VIOLATION"}]}}`

// TestIsAntigravityAccountBan covers the rule that switches an account off: a
// Google ToS refusal of the account itself, on the Antigravity providers only.
func TestIsAntigravityAccountBan(t *testing.T) {
	cases := []struct {
		name      string
		provider  string
		status    int
		body      string
		errorText string
		want      bool
	}{
		{"antigravity ToS 403 bans the account", "antigravity", http.StatusForbidden, tosBody, "", true},
		{"agy alias is covered too", "agy", http.StatusForbidden, tosBody, "", true},
		{"same text on another provider is ignored", "openrouter", http.StatusForbidden, tosBody, "", false},
		{"ToS text on a non-403 is not a ban", "antigravity", http.StatusTooManyRequests, tosBody, "", false},
		{"a model quota 403 is not an account ban", "antigravity", http.StatusForbidden,
			`{"error":{"message":"Model quota reached for gemini-4-hydrogen-low"}}`, "Model quota reached for gemini-4-hydrogen-low", false},
		{"the error text alone counts when it carries the Cloud Code marker", "antigravity", http.StatusForbidden, "",
			"tos_violation from cloudcode-pa.googleapis.com", true},
		{"a bare tos_violation with no Cloud Code marker does not count", "antigravity", http.StatusForbidden, "",
			"tos_violation from some other service", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsAntigravityAccountBan(c.provider, c.status, []byte(c.body), c.errorText); got != c.want {
				t.Errorf("IsAntigravityAccountBan(%q, %d) = %v, want %v", c.provider, c.status, got, c.want)
			}
		})
	}
}

// The stored reason is our fixed sentence. It must never carry an appeal link
// or any token from the upstream body.
func TestAntigravityAccountDisabledMessageHasNoLink(t *testing.T) {
	for _, bad := range []string{"http", "https", "accounts.google", "forms.gle", "appeal_url", "token"} {
		if strings.Contains(strings.ToLower(AntigravityAccountDisabledMessage), bad) {
			t.Errorf("the disabled reason contains %q; it must stay a fixed sentence", bad)
		}
	}
}
