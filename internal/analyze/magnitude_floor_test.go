// Tests for the magnitude-floor honesty fix: adoption, throughput, and context must not
// assert a headline favorable read (STRONG/RAMPING/HEALTHY) from a trivially small sample
// -- 2 projects with 1 session each, a 1-to-2-line change, or a single session with zero
// compactions (see internal/analyze/{adoption,throughput,context}.go).
package analyze

import (
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// TestAdoptionTrivialBroadIsNotConfidentlyStrong reproduces the exact reviewer-cited bad
// case: 2 projects, 1 session each, no week-over-week trend data. Before the floor, "more
// than one project" alone was enough for a confident [STRONG].
func TestAdoptionTrivialBroadIsNotConfidentlyStrong(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 10, Out: 10, LinesAdded: 5},
		{Day: "2026-07-11", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "api", In: 10, Out: 10, LinesAdded: 5},
	}
	sessions := []store.SessionRow{
		{SessionID: "s1", Project: "web", Tool: "claude-code", Model: "claude-sonnet-4-5", FirstTs: validatorsTestNow, Turns: 1},
		{SessionID: "s2", Project: "api", Tool: "claude-code", Model: "claude-sonnet-4-5", FirstTs: validatorsTestNow, Turns: 1},
	}
	in := BuildInput(usage, sessions, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, ok := Get(adoptionName)
	if !ok {
		t.Fatal("adoption not registered")
	}
	got := v.Analyze(in)
	if got.Read.Label == "STRONG" {
		t.Fatalf("Read = %+v, want NOT STRONG for 2 projects with 1 session each (trivial sample)", got.Read)
	}
}

// TestThroughputTrivialLinesIsNotConfidentlyRamping reproduces the reviewer-cited bad
// case: a 1-to-2-line change is a 100%% week-over-week swing, but far too small a sample
// to call a ramp.
func TestThroughputTrivialLinesIsNotConfidentlyRamping(t *testing.T) {
	usage := []store.UsageRow{
		{Day: "2026-07-13", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 10, Out: 10, LinesAdded: 2},
		{Day: "2026-07-02", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", In: 10, Out: 10, LinesAdded: 1},
	}
	in := BuildInput(usage, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, ok := Get(throughputName)
	if !ok {
		t.Fatal("throughput not registered")
	}
	got := v.Analyze(in)
	if got.Read.Label == "RAMPING" {
		t.Fatalf("Read = %+v, want NOT RAMPING for a 1-to-2-line change (trivial sample)", got.Read)
	}
}

// TestContextTrivialHealthyIsNotConfident reproduces the parallel bad case: a single
// session with zero compactions read confidently HEALTHY before the session-count floor.
func TestContextTrivialHealthyIsNotConfident(t *testing.T) {
	sessions := []store.SessionRow{
		{SessionID: "s1", Project: "web", Tool: "claude-code", Model: "claude-sonnet-4-5", FirstTs: validatorsTestNow, Turns: 3},
	}
	in := BuildInput(nil, sessions, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	v, ok := Get(contextName)
	if !ok {
		t.Fatal("context not registered")
	}
	got := v.Analyze(in)
	if got.Read.Label == "HEALTHY" {
		t.Fatalf("Read = %+v, want NOT HEALTHY for a single session with zero compactions (trivial sample)", got.Read)
	}
}
