package store

import (
	"context"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func TestStepWatchCountsOnlyTheColumnsThatCannotMoveWhileASessionIsWritten(t *testing.T) {
	at := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	base := usage.Step{
		Tool: "claude-code", SessionID: "s1", Timeline: "main", DedupeKey: "t1:0", Timestamp: at,
		Ordinal: 1, Kind: usage.StepRead, Outcome: usage.OutcomeOK, Model: "m", Tokens: 10, TargetRef: 3,
	}
	tests := []struct {
		name  string
		first func(*usage.Step)
		offer func(*usage.Step)
		want  int
	}{
		{"an identical re-read", nil, nil, 0},
		{"an earlier timestamp", nil, func(s *usage.Step) { s.Timestamp = at.Add(-time.Minute) }, 1},
		{"a later timestamp is not a move", nil, func(s *usage.Step) { s.Timestamp = at.Add(time.Minute) }, 0},
		{"a reclassified call", nil, func(s *usage.Step) { s.Kind = usage.StepEdit }, 1},
		{"a moved position", nil, func(s *usage.Step) { s.Ordinal = 2 }, 1},
		{"a different target", nil, func(s *usage.Step) { s.TargetRef = 4 }, 1},
		{"a corrected model", nil, func(s *usage.Step) { s.Model = "m2" }, 1},
		{"a grown token total is applied, not counted", nil, func(s *usage.Step) { s.Tokens = 20 }, 0},
		{"an outcome that arrived later is applied, not counted", func(s *usage.Step) { s.Outcome = "" }, nil, 0},
		{"a filled model is applied, not counted", func(s *usage.Step) { s.Model = "" }, nil, 0},
		{"a blank model keeps the stored one", nil, func(s *usage.Step) { s.Model = "" }, 0},
		{"a later time with grown tokens keeps the time and takes the tokens", nil, func(s *usage.Step) {
			s.Timestamp, s.Tokens = at.Add(time.Minute), 20
		}, 0},
		{"a reclassified call with grown tokens takes both", nil, func(s *usage.Step) { s.Kind, s.Tokens = usage.StepEdit, 20 }, 1},
		{"another source's later time is a correction", func(s *usage.Step) { s.Tool = "codex" }, func(s *usage.Step) {
			s.Tool, s.Timestamp = "codex", at.Add(time.Minute)
		}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			st := openTempStore(t)
			first, offer := base, base
			if tt.first != nil {
				tt.first(&first)
			}
			if tt.offer != nil {
				tt.offer(&offer)
			}
			if _, err := st.InsertSteps(ctx, []usage.Step{first}); err != nil {
				t.Fatal(err)
			}
			w, err := st.InsertSteps(ctx, []usage.Step{offer})
			if err != nil {
				t.Fatal(err)
			}
			if w.Changed != tt.want {
				t.Fatalf("Changed = %d, want %d", w.Changed, tt.want)
			}
			var got, want storedStep
			if err := st.db.QueryRowContext(ctx, `SELECT ts, kind, ordinal, target_ref, tokens, outcome, model FROM session_step`).Scan(
				&got.ts, &got.kind, &got.ordinal, &got.target, &got.tokens, &got.outcome, &got.model); err != nil {
				t.Fatal(err)
			}
			want = storedStep{
				ts: offer.Timestamp.UTC().Format(time.RFC3339), kind: offer.Kind, ordinal: offer.Ordinal,
				target: offer.TargetRef, tokens: offer.Tokens, outcome: offer.Outcome, model: offer.Model,
			}
			if offer.Tool == "claude-code" && first.Timestamp.Before(offer.Timestamp) {
				want.ts = first.Timestamp.UTC().Format(time.RFC3339)
			}
			if want.model == "" {
				want.model = first.Model
			}
			if got != want {
				t.Fatalf("stored %+v, want %+v", got, want)
			}
		})
	}
}

type storedStep struct {
	ts, kind, outcome, model string
	ordinal, target, tokens  int64
}
