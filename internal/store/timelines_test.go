package store

import (
	"context"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// Grouping rides on the query's own ORDER BY: the reader compares each row against the sequence it
// is building rather than keeping a map of every sequence seen, so an order that stopped matching
// the grouping key would split one sequence into several without failing anything else.
func TestTimelinesGroupStepsIntoSequences(t *testing.T) {
	ctx := context.Background()
	s := stepStore(t)
	at := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)

	steps := []usage.Step{
		{Tool: "claude-code", SessionID: "s1", DedupeKey: "a", Timestamp: at, Ordinal: 1, Kind: usage.StepAssistant, Tokens: 100},
		{Tool: "claude-code", SessionID: "s1", DedupeKey: "b", Timestamp: at.Add(time.Minute), Ordinal: 2, Kind: usage.StepEdit, Outcome: usage.OutcomeOK, TargetRef: 1},
		{Tool: "claude-code", SessionID: "s1", Timeline: "agent-7", DedupeKey: "c", Timestamp: at.Add(2 * time.Minute), Ordinal: 1, Kind: usage.StepRead, Outcome: usage.OutcomeError, TargetRef: 1},
		{Tool: "claude-code", SessionID: "s2", DedupeKey: "d", Timestamp: at.Add(3 * time.Minute), Ordinal: 1, Kind: usage.StepCommand, Outcome: usage.OutcomeDenied},
	}
	if _, rejected, err := s.InsertSteps(ctx, steps); err != nil || rejected != 0 {
		t.Fatalf("InsertSteps: %v, rejected %d", err, rejected)
	}

	got, err := s.Timelines(ctx, at.Add(-time.Hour))
	if err != nil {
		t.Fatalf("Timelines: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("sequences = %d, want 3: s1's main loop, s1's sub-agent, and s2", len(got))
	}
	main := got[0]
	if main.SessionID != "s1" || main.Timeline != "" || len(main.Steps) != 2 {
		t.Errorf("first sequence = %+v, want s1's main loop with 2 steps", main)
	}
	if main.Steps[0].Ordinal != 1 || main.Steps[1].Ordinal != 2 {
		t.Errorf("steps are not in sequence order: %+v", main.Steps)
	}
	if main.Steps[0].Tokens != 100 || main.Steps[1].TargetRef != 1 || main.Steps[1].Outcome != usage.OutcomeOK {
		t.Errorf("step fields did not survive the round trip: %+v", main.Steps)
	}
	sub := got[1]
	if !sub.SubAgent() || sub.Timeline != "agent-7" {
		t.Errorf("second sequence = %+v, want s1's sub-agent kept apart from its main loop", sub)
	}
}

// The window is a filter on steps, and a sequence straddling it comes back with the part inside.
// A reader has to allow for that rather than treating a sequence starting above ordinal 1 as loss.
func TestTimelinesReturnOnlyTheStepsInsideTheWindow(t *testing.T) {
	ctx := context.Background()
	s := stepStore(t)
	old := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	recent := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)

	steps := []usage.Step{
		{Tool: "claude-code", SessionID: "s1", DedupeKey: "old", Timestamp: old, Ordinal: 1, Kind: usage.StepAssistant},
		{Tool: "claude-code", SessionID: "s1", DedupeKey: "new", Timestamp: recent, Ordinal: 2, Kind: usage.StepEdit, TargetRef: 3},
	}
	if _, _, err := s.InsertSteps(ctx, steps); err != nil {
		t.Fatalf("InsertSteps: %v", err)
	}

	got, err := s.Timelines(ctx, recent.Add(-time.Hour))
	if err != nil {
		t.Fatalf("Timelines: %v", err)
	}
	if len(got) != 1 || len(got[0].Steps) != 1 {
		t.Fatalf("sequences = %+v, want one holding only the step inside the window", got)
	}
	if got[0].Steps[0].Ordinal != 2 {
		t.Errorf("kept step = %+v, want the ordinal-2 one: a window cuts the opening off", got[0].Steps[0])
	}
}

// Scope comes from the session's usage records. A sequence whose session has none in the window is
// left unattributed rather than given a scope nothing measured -- which trace.Scope then reports as
// "unstated" instead of quietly folding it in with a person's own sessions.
func TestTimelinesAttachTheSessionScopeAndLeaveItEmptyWhenUnknown(t *testing.T) {
	ctx := context.Background()
	s := stepStore(t)
	at := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)

	if _, err := s.InsertLocal(ctx, []usage.Record{{
		Tool: "claude-code", SessionID: "known", Timestamp: at, Model: "m", DedupeKey: "r1",
		Entrypoint: "cli", Project: "assaio", Granularity: "turn", InputTokens: 1,
	}}); err != nil {
		t.Fatalf("InsertLocal: %v", err)
	}
	steps := []usage.Step{
		{Tool: "claude-code", SessionID: "known", DedupeKey: "a", Timestamp: at, Ordinal: 1, Kind: usage.StepAssistant},
		{Tool: "claude-code", SessionID: "orphan", DedupeKey: "b", Timestamp: at, Ordinal: 1, Kind: usage.StepAssistant},
	}
	if _, _, err := s.InsertSteps(ctx, steps); err != nil {
		t.Fatalf("InsertSteps: %v", err)
	}

	got, err := s.Timelines(ctx, at.Add(-time.Hour))
	if err != nil {
		t.Fatalf("Timelines: %v", err)
	}
	byID := map[string]Timeline{}
	for _, tl := range got {
		byID[tl.SessionID] = tl
	}
	if k := byID["known"]; k.Entrypoint != "cli" || k.Project != "assaio" {
		t.Errorf("known session = %+v, want its own entrypoint and project", k)
	}
	if o := byID["orphan"]; o.Entrypoint != "" || o.Project != "" {
		t.Errorf("orphan session = %+v, want both left empty rather than borrowed", o)
	}
}
