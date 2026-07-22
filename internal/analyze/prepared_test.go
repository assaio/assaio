package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// preparedTestPrices seeds one premium-priced and one cheaper-priced model.
// "unknown-model" is deliberately absent from every test in this file, so BuildInput's
// tier classification and Priced=false path are both exercised against a real gap in the
// price table, not just the two known models.
func preparedTestPrices() pricing.Table {
	return pricing.Table{
		"premium-model": {Input: 1e-5, Output: 25e-6},
		"cheap-model":   {Input: 3e-6, Output: 15e-6},
	}
}

func TestBuildInputByModelSumsClassifiesAndShares(t *testing.T) {
	usage := []store.UsageRow{
		{Model: "premium-model", Project: "p", In: 1000, Out: 2000, CacheRead: 100, CacheWrite: 50, Reasoning: 10, LinesAdded: 30},
		{Model: "premium-model", Project: "p", In: 500, Out: 500, LinesAdded: 20},
		{Model: "cheap-model", Project: "p", In: 2000, Out: 4000, LinesAdded: 100},
		{Model: "unknown-model", Project: "p", In: 300, Out: 300, LinesAdded: 5},
	}
	in := BuildInput(usage, nil, preparedTestPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})

	if len(in.ByModel) != 3 {
		t.Fatalf("len(ByModel) = %d, want 3: %+v", len(in.ByModel), in.ByModel)
	}
	// Sorted by Tokens descending: cheap-model (6000) > premium-model (4150) > unknown-model (600).
	wantOrder := []string{"cheap-model", "premium-model", "unknown-model"}
	for i, want := range wantOrder {
		if got := in.ByModel[i].Model; got != want {
			t.Fatalf("ByModel[%d].Model = %q, want %q (sorted by Tokens desc): %+v", i, got, want, in.ByModel)
		}
	}

	premium := findModelStat(t, in.ByModel, "premium-model")
	if premium.Tier != tierPremium {
		t.Fatalf("premium-model Tier = %q, want %q", premium.Tier, tierPremium)
	}
	if premium.Input != 1500 || premium.Output != 2500 || premium.CacheRead != 100 || premium.CacheWrite != 50 {
		t.Fatalf("premium-model token breakdown = %+v, want In=1500 Out=2500 CacheRead=100 CacheWrite=50", premium)
	}
	if premium.Tokens != 4150 {
		t.Fatalf("premium-model Tokens = %d, want 4150 (In+Out+CacheRead+CacheWrite; reasoning is a subset of output)", premium.Tokens)
	}
	if premium.Lines != 50 {
		t.Fatalf("premium-model Lines = %d, want 50", premium.Lines)
	}
	if !premium.Priced || premium.Cost == nil || *premium.Cost <= 0 {
		t.Fatalf("premium-model Priced/Cost = %v/%v, want a priced positive cost", premium.Priced, premium.Cost)
	}

	cheap := findModelStat(t, in.ByModel, "cheap-model")
	if cheap.Tier != tierCheaper {
		t.Fatalf("cheap-model Tier = %q, want %q", cheap.Tier, tierCheaper)
	}
	if cheap.Tokens != 6000 || cheap.Lines != 100 {
		t.Fatalf("cheap-model Tokens/Lines = %d/%d, want 6000/100", cheap.Tokens, cheap.Lines)
	}

	unknown := findModelStat(t, in.ByModel, "unknown-model")
	if unknown.Tier != tierUnknown {
		t.Fatalf("unknown-model Tier = %q, want %q", unknown.Tier, tierUnknown)
	}
	if unknown.Priced || unknown.Cost != nil {
		t.Fatalf("unknown-model Priced/Cost = %v/%v, want false/nil", unknown.Priced, unknown.Cost)
	}

	wantShare := fracOf(premium.Tokens, in.Totals.Tokens)
	if premium.TokenShare != wantShare {
		t.Fatalf("premium-model TokenShare = %v, want %v", premium.TokenShare, wantShare)
	}
	var shareSum float64
	for _, m := range in.ByModel {
		shareSum += m.TokenShare
	}
	if diff := shareSum - 1; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("ByModel TokenShare sum = %v, want ~1", shareSum)
	}
}

func findModelStat(t *testing.T, models []ModelStat, name string) ModelStat {
	t.Helper()
	for _, m := range models {
		if m.Model == name {
			return m
		}
	}
	t.Fatalf("no ModelStat named %q in %+v", name, models)
	return ModelStat{}
}

func TestBuildInputByProjectSumsAndSorts(t *testing.T) {
	usage := []store.UsageRow{
		{Model: "cheap-model", Project: "web", In: 1000, Out: 1000, LinesAdded: 50},
		{Model: "unknown-model", Project: "web", In: 500, Out: 500, LinesAdded: 200},
		{Model: "cheap-model", Project: "api", In: 2000, Out: 2000, LinesAdded: 80},
	}
	in := BuildInput(usage, nil, preparedTestPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})

	if len(in.ByProject) != 2 {
		t.Fatalf("len(ByProject) = %d, want 2: %+v", len(in.ByProject), in.ByProject)
	}
	// Sorted by Lines descending: web (250) before api (80).
	if in.ByProject[0].Project != "web" || in.ByProject[1].Project != "api" {
		t.Fatalf("ByProject order = %+v, want web then api (Lines desc)", in.ByProject)
	}
	web := in.ByProject[0]
	if web.Lines != 250 {
		t.Fatalf("web Lines = %d, want 250", web.Lines)
	}
	if web.Priced {
		t.Fatalf("web Priced = true, want false (the unknown-model row excludes some cost)")
	}
	wantWebCost := 1000*3e-6 + 1000*15e-6
	if web.Cost == nil {
		t.Fatal("web Cost = nil, want the priced row's partial contribution despite Priced=false")
	}
	if diff := *web.Cost - wantWebCost; diff < -1e-12 || diff > 1e-12 {
		t.Fatalf("web Cost = %v, want %v (only the priced row's contribution)", *web.Cost, wantWebCost)
	}

	api := in.ByProject[1]
	if api.Lines != 80 || !api.Priced {
		t.Fatalf("api = %+v, want Lines=80 Priced=true", api)
	}
}

func TestBuildInputTotalsAndCacheEfficiency(t *testing.T) {
	usage := []store.UsageRow{
		{Model: "cheap-model", In: 100, Out: 50, CacheRead: 300, CacheWrite: 20, Reasoning: 5, LinesAdded: 10},
		{Model: "unknown-model", In: 200, Out: 50, LinesAdded: 4},
	}
	tot := BuildInput(usage, nil, preparedTestPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{}).Totals

	if tot.Input != 300 || tot.Output != 100 || tot.CacheRead != 300 || tot.CacheWrite != 20 {
		t.Fatalf("Totals token breakdown = %+v, want In=300 Out=100 CacheRead=300 CacheWrite=20", tot)
	}
	if tot.Tokens != 720 {
		t.Fatalf("Totals.Tokens = %d, want 720 (300+100+300+20; reasoning is a subset of output, not re-added)", tot.Tokens)
	}
	if tot.Lines != 14 {
		t.Fatalf("Totals.Lines = %d, want 14", tot.Lines)
	}
	if tot.Priced {
		t.Fatalf("Totals.Priced = true, want false (the unknown-model row is unpriced)")
	}
	wantCost := 100*3e-6 + 50*15e-6
	if tot.Cost == nil {
		t.Fatal("Totals.Cost = nil, want the priced row's partial contribution despite Priced=false")
	}
	if diff := *tot.Cost - wantCost; diff < -1e-12 || diff > 1e-12 {
		t.Fatalf("Totals.Cost = %v, want %v (only the priced row's contribution)", *tot.Cost, wantCost)
	}
	wantEff := 300.0 / 600.0
	if tot.CacheEfficiency != wantEff {
		t.Fatalf("CacheEfficiency = %v, want %v (CacheRead/(CacheRead+Input))", tot.CacheEfficiency, wantEff)
	}
}

func TestBuildInputCacheEfficiencyZeroDenominatorIsZero(t *testing.T) {
	usage := []store.UsageRow{{Model: "cheap-model", Out: 100, LinesAdded: 1}}
	tot := BuildInput(usage, nil, preparedTestPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{}).Totals
	if tot.CacheEfficiency != 0 {
		t.Fatalf("CacheEfficiency = %v, want 0 when CacheRead and Input are both zero", tot.CacheEfficiency)
	}
}

// TestBuildInputEmptyUsageIsSafeAndZero asserts BuildInput on no usage never panics and
// produces empty (not nil-panicking) prepared views and an honestly zero-value Totals,
// including Priced=false -- a vacuous "everything priced" would be dishonest when there
// is nothing to price.
func TestBuildInputEmptyUsageIsSafeAndZero(t *testing.T) {
	in := BuildInput(nil, nil, preparedTestPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	if len(in.ByModel) != 0 {
		t.Fatalf("ByModel = %+v, want empty", in.ByModel)
	}
	if len(in.ByProject) != 0 {
		t.Fatalf("ByProject = %+v, want empty", in.ByProject)
	}
	if in.Totals != (Totals{}) {
		t.Fatalf("Totals = %+v, want the zero value", in.Totals)
	}
}

// TestBuildInputNilPricesIsSafe asserts a nil pricing.Table never panics -- every model
// reads as unpriced/unknown, not a crash.
func TestBuildInputNilPricesIsSafe(t *testing.T) {
	usage := []store.UsageRow{{Model: "m", In: 10, Out: 10, LinesAdded: 1}}
	in := BuildInput(usage, nil, nil, validatorsTestNow, 7*24*time.Hour, Delegation{})
	if len(in.ByModel) != 1 || in.ByModel[0].Priced {
		t.Fatalf("ByModel = %+v, want one unpriced entry", in.ByModel)
	}
	if in.Totals.Priced {
		t.Fatalf("Totals.Priced = true, want false with a nil price table")
	}
}

// TestModelFitReadsByModelDirectly locks in that model-fit's Figures/Bars come straight
// from in.ByModel with no re-derivation: a hand-computed premium/cheaper split over the
// same usage BuildInput classifies must match Analyze's rendered shares exactly.
func TestModelFitReadsByModelDirectly(t *testing.T) {
	usage := []store.UsageRow{
		{Model: "premium-model", Project: "p", In: 1000, Out: 3000, LinesAdded: 40},
		{Model: "cheap-model", Project: "p", In: 1000, Out: 1000, LinesAdded: 60},
	}
	in := BuildInput(usage, nil, preparedTestPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{Sub: 10, Total: 100})
	v, ok := Get(modelFitName)
	if !ok {
		t.Fatal("model-fit not registered")
	}
	got := v.Analyze(in)

	premiumTokens, cheaperTokens := int64(4000), int64(2000) // 1000+3000, 1000+1000
	total := premiumTokens + cheaperTokens
	wantPremiumShare := shareOrDash(premiumTokens, total, 1)
	wantCheaperShare := shareOrDash(cheaperTokens, total, 1)

	joined := figureValues(got.Figures)
	if !strings.Contains(joined, wantPremiumShare) {
		t.Fatalf("Figures = %q, want the hand-computed premium share %q", joined, wantPremiumShare)
	}
	if !strings.Contains(joined, wantCheaperShare) {
		t.Fatalf("Figures = %q, want the hand-computed cheaper share %q", joined, wantCheaperShare)
	}
	if len(got.Bars) != 2 || got.Bars[0].Label != "premium-model" {
		t.Fatalf("Bars = %+v, want premium-model first (more tokens)", got.Bars)
	}
}
