package dashboard

import (
	"testing"

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
