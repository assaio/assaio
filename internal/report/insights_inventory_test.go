package report

import "testing"

func TestBuildInsightsInventoryCounts(t *testing.T) {
	in := BuildInsights(insightsFixtureRows(), table(), insightsNow, insightsRecentWindow, 5)
	inv := in.Inventory
	if inv.Projects != 4 {
		t.Fatalf("Projects = %d want 4 (web, api, infra, ghost)", inv.Projects)
	}
	if inv.Models != 2 {
		t.Fatalf("Models = %d want 2", inv.Models)
	}
	if inv.Tools != 2 {
		t.Fatalf("Tools = %d want 2 (claude-code, codex)", inv.Tools)
	}
	if inv.Entrypoints != 1 {
		t.Fatalf("Entrypoints = %d want 1 (all unset)", inv.Entrypoints)
	}
	if inv.Days != 5 {
		t.Fatalf("Days = %d want 5", inv.Days)
	}
	if inv.TotalLinesAdded != 165 {
		t.Fatalf("TotalLinesAdded = %d want 165", inv.TotalLinesAdded)
	}
}

func TestBuildInventoryNoUsageAtAll(t *testing.T) {
	inv := BuildInventory(nil, table())
	if inv != (Inventory{}) {
		t.Fatalf("BuildInventory(nil) = %+v want zero value", inv)
	}
}
