package analyze

import (
	"testing"

	"github.com/assaio/assaio/internal/pricing"
)

func tierTestPrices() pricing.Table {
	return pricing.Table{
		"claude-opus-4-5":       {Input: 5e-6, Output: 25e-6},
		"claude-sonnet-4-5":     {Input: 3e-6, Output: 15e-6},
		"claude-mythos-preview": {Input: 0, Output: 0},
	}
}

func TestModelTierPremiumByPrice(t *testing.T) {
	if got := modelTier("claude-opus-4-5", tierTestPrices()); got != tierPremium {
		t.Fatalf("modelTier(opus) = %q, want %q", got, tierPremium)
	}
}

func TestModelTierCheaperByPrice(t *testing.T) {
	if got := modelTier("claude-sonnet-4-5", tierTestPrices()); got != tierCheaper {
		t.Fatalf("modelTier(sonnet) = %q, want %q", got, tierCheaper)
	}
}

func TestModelTierUnknownWhenUnpriced(t *testing.T) {
	if got := modelTier("claude-nonexistent-9", tierTestPrices()); got != tierUnknown {
		t.Fatalf("modelTier(unpriced) = %q, want %q", got, tierUnknown)
	}
}

// TestModelTierFreeModelIsCheaperNotPremium asserts a zero-priced (e.g. free preview)
// model that IS in the price table is "cheaper", not "unknown" -- unknown is reserved for
// models absent from the table, never for a legitimately zero price.
func TestModelTierFreeModelIsCheaperNotPremium(t *testing.T) {
	if got := modelTier("claude-mythos-preview", tierTestPrices()); got != tierCheaper {
		t.Fatalf("modelTier(free preview) = %q, want %q", got, tierCheaper)
	}
}

// TestModelTierBoundaryIsPremium asserts a model priced exactly at
// premiumOutputPriceFloor classifies premium (>=, not >), so the floor is inclusive.
func TestModelTierBoundaryIsPremium(t *testing.T) {
	prices := pricing.Table{"boundary-model": {Output: premiumOutputPriceFloor}}
	if got := modelTier("boundary-model", prices); got != tierPremium {
		t.Fatalf("modelTier(boundary) = %q, want %q", got, tierPremium)
	}
}

// TestModelTierNormalizesModelName asserts a log-style model name (date stamp + context
// tag) resolves through pricing.NormalizeModel to its priced base name, mirroring
// pricing.Table.CostTokens's own lookup fallback.
func TestModelTierNormalizesModelName(t *testing.T) {
	prices := pricing.Table{"claude-opus-4-5": {Output: 25e-6}}
	if got := modelTier("Claude-Opus-4-5-20251101[1m]", prices); got != tierPremium {
		t.Fatalf("modelTier(normalized name) = %q, want %q", got, tierPremium)
	}
}

func TestModelTierDeterministic(t *testing.T) {
	prices := tierTestPrices()
	first := modelTier("claude-opus-4-5", prices)
	for range 10 {
		if got := modelTier("claude-opus-4-5", prices); got != first {
			t.Fatalf("modelTier not deterministic: got %q, want %q", got, first)
		}
	}
}
