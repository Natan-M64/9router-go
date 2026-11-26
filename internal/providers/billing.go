package providers

import "strings"

// BillingMode says how a provider's usage is paid for, which is what lets the
// dashboard separate money actually invoiced per token from list-price cost on
// a subscription or free quota. The per-token `cost` column is a list-price
// estimate for every provider alike; adding it all up as "spend" overstates the
// bill by the price of every plan-backed and free request.
type BillingMode string

const (
	// BillingMetered providers bill per token. The estimated cost is money the
	// operator is charged, subject to list-price drift.
	BillingMetered BillingMode = "metered"
	// BillingIncluded providers are reached through a subscription, a plan
	// quota, free credits or a free web session. Tokens are not invoiced, so
	// the estimated cost is an opportunity cost, not a charge.
	BillingIncluded BillingMode = "included"
	// BillingUnknown providers cannot be classified: a user-defined compatible
	// endpoint or an id this build does not know. The cost is reported apart
	// rather than folded into either side.
	BillingUnknown BillingMode = "unknown"
)

// BillingModeForProvider classifies a provider id by how its credentials are
// obtained.
//
// AuthType is the signal rather than Category: an API key is a metered
// credential by definition, while an OAuth session, a web cookie session or a
// credential-less provider reaches an account with its own plan or free quota.
// Category disagrees with that in practice — `openrouter` and `kilo-gateway`
// are filed under Free Tier because they publish free models, but their paid
// usage is charged against an API key and must count as metered.
//
// A user-defined compatible endpoint stays unknown: it may be a local Ollama
// (free) or a paid upstream behind a custom base URL, and guessing either way
// would misstate the bill.
// planProviders are reached with an API key but paid for by a fixed-price
// plan, so a per-token list price is not an invoice. The registry Notice of
// each entry states the plan; it is quoted so the classification can be
// audited against the catalog.
//
// opencode-go is the one that matters in practice: it is the busiest provider
// on a typical install, and treating its list price as spend overstated the
// metered figure by an order of magnitude.
var planProviders = map[string]string{
	"opencode-go": "OpenCode Go subscription: $5/mo (then 10/mo)",
	"ollama":      "Free tier: light usage … Pro $20/mo · Max $100/mo",
	"xiaomi-mimo": "Xiaomi MiMo Token Plan subscription",
	"glm-cn":      "Z.AI Coding Plan",
}

func BillingModeForProvider(providerID string) BillingMode {
	meta, ok := GetProviderMeta(providerID)
	if !ok {
		return BillingUnknown
	}
	// A plan is not a per-token invoice, whatever credential it uses.
	if _, plan := planProviders[strings.ToLower(strings.TrimSpace(meta.ID))]; plan {
		return BillingIncluded
	}
	if meta.Category == CategoryCustom || meta.ID == "muse" {
		return BillingUnknown
	}
	if meta.NoAuth {
		return BillingIncluded
	}
	switch meta.AuthType {
	case "apikey":
		return BillingMetered
	case "oauth", "cookie", "none":
		return BillingIncluded
	}
	// An entry that declares no auth type falls back to its category.
	switch meta.Category {
	case CategoryAPIKey:
		return BillingMetered
	case CategoryOAuth, CategoryFreeTier, CategoryFree, CategoryWebCookie:
		return BillingIncluded
	}
	return BillingUnknown
}
