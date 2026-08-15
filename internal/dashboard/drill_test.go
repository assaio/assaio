package dashboard

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/store"
)

func TestTopProjectPicksHighestLines(t *testing.T) {
	usage := []store.UsageRow{
		{Project: "web", LinesAdded: 100},
		{Project: "api", LinesAdded: 300},
		{Project: "web", LinesAdded: 50},
	}
	if got := TopProject(usage); got != "api" {
		t.Fatalf("TopProject = %q, want %q", got, "api")
	}
}

// TestTopProjectIgnoresEmptyProjectName asserts unattributed usage (Project == "") can
// never win the drill-down, even when it dwarfs every named project.
func TestTopProjectIgnoresEmptyProjectName(t *testing.T) {
	usage := []store.UsageRow{
		{Project: "", LinesAdded: 10000},
		{Project: "web", LinesAdded: 1},
	}
	if got := TopProject(usage); got != "web" {
		t.Fatalf("TopProject = %q, want %q", got, "web")
	}
}

func TestTopProjectEmptyUsageReturnsEmpty(t *testing.T) {
	if got := TopProject(nil); got != "" {
		t.Fatalf("TopProject(nil) = %q, want \"\"", got)
	}
}

// TestTopProjectTieBreaksDeterministically asserts a tie resolves the same way on every
// call -- Go's randomized map iteration order must never leak into the result.
func TestTopProjectTieBreaksDeterministically(t *testing.T) {
	usage := []store.UsageRow{
		{Project: "zeta", LinesAdded: 100},
		{Project: "alpha", LinesAdded: 100},
	}
	got := TopProject(usage)
	if got != "alpha" {
		t.Fatalf("TopProject tie-break = %q, want the lexicographically smaller %q", got, "alpha")
	}
	for range 20 {
		if again := TopProject(usage); again != got {
			t.Fatalf("TopProject is non-deterministic on a tie: %q then %q", got, again)
		}
	}
}

// TestDrillCarriesTheStoresHorizon guards the horizon the drill dropped: buildDrill copied WindowStart, Ingested, ParsedBy and
// Trace onto the scoped Input and not HistoryStart, so every Trending validator inside the
// project panel claimed the store's history "could not be read" -- on the same page whose
// top-level ledger had just read it. The panel surfaces caveats only as a Prov. badge, so the
// wrong sentence never reached the page itself; the verdict carrying it did.
func TestDrillCarriesTheStoresHorizon(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	in := analyze.BuildInput([]store.UsageRow{
		{Day: "2026-08-14", Tool: "claude-code", Model: "claude-opus-4-5", Project: "web", In: 100, Out: 50, LinesAdded: 40},
		{Day: "2026-08-01", Tool: "claude-code", Model: "claude-opus-4-5", Project: "web", In: 100, Out: 50, LinesAdded: 30},
	}, nil, nil, now, 7*24*time.Hour, analyze.Delegation{})
	in.HistoryStart = now.AddDate(0, 0, -60)

	drill := buildDrill(in, nil, false)
	if drill == nil {
		t.Fatal("the window has a named project, so it has a drill")
	}
	trending := 0
	for _, v := range drill.Verdicts {
		for _, c := range v.Caveats {
			if strings.Contains(c, "history goes could not be read") {
				t.Errorf("%s in the drill claims the horizon is unknown while the ledger printed it", v.Name)
			}
			if strings.Contains(c, "this store's history begins") {
				trending++
			}
		}
	}
	if trending == 0 {
		t.Fatal("no Trending validator reached the drill, so the assertion above never ran")
	}
}
