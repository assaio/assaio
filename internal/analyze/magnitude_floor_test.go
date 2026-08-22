// Tests for the magnitude-floor honesty fix: adoption, throughput, and context must not
// assert a headline favorable read (STRONG/RAMPING/HEALTHY) from a trivially small sample
// -- 2 projects with 1 session each, a 1-to-2-line change, or a single session with zero
// compactions (see internal/analyze/{adoption,throughput,context}.go).
package analyze

import (
	"strings"
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
	// The verdict this floor used to guard is gone (B177), so asserting on its absence would
	// pass whatever the floor did. What the floor still decides is whether a direction is
	// stated at all, and that is what this asserts.
	if strings.Contains(got.Takeaway, "up 100%") {
		t.Fatalf("Takeaway = %q, want a 1-to-2-line change not stated as a direction", got.Takeaway)
	}
	if !strings.Contains(got.Takeaway, "too small") {
		t.Fatalf("Takeaway = %q, want the trivial comparison named", got.Takeaway)
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
	// Same as above: "HEALTHY" no longer exists, so the floor is asserted on what it still
	// controls -- whether a rate is read off one session at all.
	if got.Read != noDataRead {
		t.Fatalf("Read = %+v, want the withheld read for a single session", got.Read)
	}
	if !strings.Contains(got.Takeaway, "Too few sessions") {
		t.Fatalf("Takeaway = %q, want the session floor named", got.Takeaway)
	}
}
