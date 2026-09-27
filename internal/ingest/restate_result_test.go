package ingest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

// TestReReadsReachTheResult: the store counts what a re-read changed, and nothing else carries
// those counts to the backfill line -- a dropped `+=` here passes every store test.
func TestReReadsReachTheResult(t *testing.T) {
	at := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	rec := usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: at, Model: "m", DedupeKey: "k1",
		Granularity: "turn", InputTokens: 10, OutputTokens: 5, LinesAdded: 30,
	}
	step := usage.Step{
		Tool: "claude-code", SessionID: "s1", Timeline: "main", DedupeKey: "k1:0", Timestamp: at,
		Ordinal: 1, Kind: usage.StepRead,
	}
	tests := []struct {
		name  string
		rec   func(*usage.Record)
		step  func(*usage.Step)
		check func(Result) bool
	}{
		{"an identical re-read", nil, nil, func(r Result) bool {
			return r.Lowered == 0 && r.Identity == (store.IdentityChanges{}) && r.StepsChanged == 0
		}},
		{"a lowered figure", func(r *usage.Record) { r.LinesAdded = 10 }, nil, func(r Result) bool { return r.Lowered == 1 }},
		{"a replaced model", func(r *usage.Record) { r.Model = "m2" }, nil, func(r Result) bool {
			return r.Identity.Rows == 1 && r.Identity.Model == 1
		}},
		{"a reclassified step", nil, func(s *usage.Step) { s.Kind = usage.StepEdit }, func(r Result) bool { return r.StepsChanged == 1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = st.Close() }()
			var first Result
			if err := ingestParsed(ctx, st, make(projectCache), &first, []usage.Record{rec}, 0, nil); err != nil {
				t.Fatal(err)
			}
			if err := ingestSteps(ctx, st, &first, []usage.Step{step}); err != nil {
				t.Fatal(err)
			}
			again, againStep := rec, step
			if tt.rec != nil {
				tt.rec(&again)
			}
			if tt.step != nil {
				tt.step(&againStep)
			}
			var res Result
			if err := ingestParsed(ctx, st, make(projectCache), &res, []usage.Record{again}, 0, nil); err != nil {
				t.Fatal(err)
			}
			if err := ingestSteps(ctx, st, &res, []usage.Step{againStep}); err != nil {
				t.Fatal(err)
			}
			if !tt.check(res) {
				t.Fatalf("Result = %+v", res)
			}
		})
	}
}
