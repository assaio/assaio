package report

import (
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// insightsNow is a fixed reference time so recent/prior splits are deterministic across
// test runs. recent=7d puts the cutoff at 2026-07-07.
var insightsNow = time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

const insightsRecentWindow = 7 * 24 * time.Hour

// insightsFixtureRows covers: a project active both prior and recent (web, must not be
// stale), a project active only recently (api, hot), a project active only prior with a
// priced model (infra, going-stale), a project active only prior with an unpriced model
// (ghost, going-stale + unpriced propagation), a tool used only prior (codex, dormant),
// and a tool used both prior and recent (claude-code, not dormant).
func insightsFixtureRows() []store.UsageRow {
	return []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-opus-4-5", Project: "web", In: 1000, Out: 500, LinesAdded: 100},
		{Day: "2026-07-11", Tool: "claude-code", Model: "claude-opus-4-5", Project: "api", In: 100, Out: 50, LinesAdded: 10},
		{Day: "2026-07-01", Tool: "claude-code", Model: "claude-opus-4-5", Project: "web", In: 200, Out: 100, LinesAdded: 20},
		{Day: "2026-07-02", Tool: "codex", Model: "claude-opus-4-5", Project: "infra", In: 300, Out: 150, LinesAdded: 30},
		{Day: "2026-07-03", Tool: "codex", Model: "some-unknown-model", Project: "ghost", In: 500, LinesAdded: 5},
	}
}

func TestBuildInsightsRecentPriorSplit(t *testing.T) {
	in := BuildInsights(insightsFixtureRows(), table(), insightsNow, insightsRecentWindow, 5)

	hotNames := groupNames(in.Hot)
	if !containsName(hotNames, "web") || !containsName(hotNames, "api") {
		t.Fatalf("Hot must include the recently active projects: %+v", in.Hot)
	}
	if containsName(hotNames, "infra") || containsName(hotNames, "ghost") {
		t.Fatalf("Hot must exclude projects with no recent usage: %+v", in.Hot)
	}
}

func TestBuildInsightsHotRankingAndTopNCap(t *testing.T) {
	in := BuildInsights(insightsFixtureRows(), table(), insightsNow, insightsRecentWindow, 5)
	if len(in.Hot) != 2 {
		t.Fatalf("len(Hot) = %d want 2: %+v", len(in.Hot), in.Hot)
	}
	if in.Hot[0].Name != "web" || in.Hot[1].Name != "api" {
		t.Fatalf("Hot must rank by cost desc (web > api): %+v", in.Hot)
	}

	capped := BuildInsights(insightsFixtureRows(), table(), insightsNow, insightsRecentWindow, 1)
	if len(capped.Hot) != 1 || capped.Hot[0].Name != "web" {
		t.Fatalf("topN=1 must cap Hot to the single hottest project: %+v", capped.Hot)
	}
}

func TestBuildInsightsGoingStaleDetectsActiveThenQuiet(t *testing.T) {
	in := BuildInsights(insightsFixtureRows(), table(), insightsNow, insightsRecentWindow, 5)

	staleNames := groupNames(in.GoingStale)
	if containsName(staleNames, "web") {
		t.Fatalf("web is still active in the recent window and must not be GoingStale: %+v", in.GoingStale)
	}
	if !containsName(staleNames, "infra") || !containsName(staleNames, "ghost") {
		t.Fatalf("infra and ghost went quiet and must be GoingStale: %+v", in.GoingStale)
	}

	byName := map[string]GroupStat{}
	for _, g := range in.GoingStale {
		byName[g.Name] = g
	}
	if byName["infra"].LastActive != "2026-07-02" {
		t.Fatalf("infra LastActive = %q want 2026-07-02", byName["infra"].LastActive)
	}
	if byName["ghost"].LastActive != "2026-07-03" {
		t.Fatalf("ghost LastActive = %q want 2026-07-03", byName["ghost"].LastActive)
	}
	// infra is priced, ghost is not: cost-desc ranking must put infra first.
	if in.GoingStale[0].Name != "infra" {
		t.Fatalf("GoingStale must rank priced infra above unpriced ghost: %+v", in.GoingStale)
	}
}

func TestBuildInsightsGoingStaleTopNCap(t *testing.T) {
	in := BuildInsights(insightsFixtureRows(), table(), insightsNow, insightsRecentWindow, 1)
	if len(in.GoingStale) != 1 {
		t.Fatalf("topN=1 must cap GoingStale to 1: %+v", in.GoingStale)
	}
}

func TestBuildInsightsDormantToolDetection(t *testing.T) {
	in := BuildInsights(insightsFixtureRows(), table(), insightsNow, insightsRecentWindow, 5)
	if len(in.DormantTools) != 1 || in.DormantTools[0].Name != "codex" {
		t.Fatalf("codex is the only tool absent from the recent window: %+v", in.DormantTools)
	}
	toolNames := groupNames(in.DormantTools)
	if containsName(toolNames, "claude-code") {
		t.Fatalf("claude-code is active in both windows and must not be dormant: %+v", in.DormantTools)
	}
}

func TestBuildInsightsUnpricedPropagation(t *testing.T) {
	in := BuildInsights(insightsFixtureRows(), table(), insightsNow, insightsRecentWindow, 5)

	if !in.Inventory.HasUnpriced {
		t.Fatal("Inventory.HasUnpriced must be true: ghost's model is unpriced")
	}
	if in.Inventory.TotalCost == nil {
		t.Fatal("Inventory.TotalCost must still be non-nil: other rows are priced")
	}

	var ghost GroupStat
	for _, g := range in.GoingStale {
		if g.Name == "ghost" {
			ghost = g
		}
	}
	if !ghost.HasUnpriced {
		t.Fatalf("ghost group must carry HasUnpriced: %+v", ghost)
	}
	if ghost.Cost != nil {
		t.Fatalf("ghost group must have nil Cost (only unpriced usage): %+v", ghost)
	}
	if ghost.CostPer100Lines != nil {
		t.Fatalf("ghost group must have nil CostPer100Lines when cost is unknown: %+v", ghost)
	}
}

func TestBuildInsightsEmptyInputZeroValue(t *testing.T) {
	for _, rows := range [][]store.UsageRow{nil, {}} {
		in := BuildInsights(rows, table(), insightsNow, insightsRecentWindow, 5)
		if len(in.Hot) != 0 || len(in.GoingStale) != 0 || len(in.DormantTools) != 0 {
			t.Fatalf("empty input must yield empty sections: %+v", in)
		}
		want := Inventory{}
		if in.Inventory != want {
			t.Fatalf("empty input must yield zero-value Inventory, got %+v", in.Inventory)
		}
	}
}

func TestBuildInsightsHotTieBreaksByName(t *testing.T) {
	rows := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-opus-4-5", Project: "beta", In: 100, Out: 50},
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-opus-4-5", Project: "alpha", In: 100, Out: 50},
	}
	for range 5 {
		in := BuildInsights(rows, table(), insightsNow, insightsRecentWindow, 5)
		if len(in.Hot) != 2 || in.Hot[0].Name != "alpha" || in.Hot[1].Name != "beta" {
			t.Fatalf("equal-cost groups must tie-break alphabetically: %+v", in.Hot)
		}
	}
}

func groupNames(stats []GroupStat) []string {
	names := make([]string, len(stats))
	for i, g := range stats {
		names[i] = g.Name
	}
	return names
}

func containsName(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}
