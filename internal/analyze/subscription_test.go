package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// TestSubscriptionFitComparesPlanToApiEquivalent covers the paying-off path: with a plan
// cost configured, the validator projects the window's API-equivalent to a month and reports
// the value multiple -- the honest reframe of a raw estimate that is meaningless as spend.
func TestSubscriptionFitComparesPlanToApiEquivalent(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 10_000_000, Out: 100_000_000},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	in.PlanMonthlyCost = 195 // one active day of ~$1530 projects to ~$45.9K/mo vs a $195 plan

	got := mustGet(t, subscriptionName).Analyze(in)
	if got.Read.Key != "good" {
		t.Fatalf("Read = %+v, want a good (paying-off) read", got.Read)
	}
	joined := figureValues(got.Figures)
	if !strings.Contains(joined, "195") {
		t.Fatalf("Figures = %q, want the $195 plan cost", joined)
	}
	if !strings.Contains(joined, "x") {
		t.Fatalf("Figures = %q, want a value multiple like 7.8x", joined)
	}
	if !strings.Contains(got.Takeaway, "pay") {
		t.Fatalf("Takeaway = %q, want a paying-off message", got.Takeaway)
	}
}

// TestSubscriptionFitPromptsWhenUnconfigured covers the default state: with no plan cost
// configured, the validator still appears but prompts the user to set one.
func TestSubscriptionFitPromptsWhenUnconfigured(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 1000, Out: 1000},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, subscriptionName).Analyze(in)
	if !strings.Contains(got.Takeaway, "monthly_subscription_cost") {
		t.Fatalf("Takeaway = %q, want a prompt to configure the plan cost", got.Takeaway)
	}
}
