package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

func stepStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "steps.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func step(key string, ordinal int64, kind string, at time.Time) usage.Step {
	return usage.Step{
		Tool: "claude-code", SessionID: "s1", DedupeKey: key,
		Timestamp: at, Ordinal: ordinal, Kind: kind,
	}
}

func TestInsertStepsIsIdempotent(t *testing.T) {
	ctx := context.Background()
	s := stepStore(t)
	at := time.Now().UTC()
	steps := []usage.Step{
		step("a", 1, usage.StepAssistant, at),
		step("b", 2, usage.StepRead, at),
	}
	n, _, err := s.InsertSteps(ctx, steps)
	if err != nil {
		t.Fatalf("InsertSteps: %v", err)
	}
	if n != 2 {
		t.Fatalf("first insert = %d, want 2", n)
	}
	n, _, err = s.InsertSteps(ctx, steps)
	if err != nil {
		t.Fatalf("re-insert: %v", err)
	}
	if n != 0 {
		t.Errorf("re-insert reported %d new rows: re-reading a transcript must not append a second copy", n)
	}
	h, err := s.Steps(ctx)
	if err != nil {
		t.Fatalf("Steps: %v", err)
	}
	if h.Steps != 2 {
		t.Errorf("stored steps = %d, want 2", h.Steps)
	}
}

// A longer prefix of the same append-only transcript completes a step: the outcome the
// answering line had not been written yet, the target the edit result names later, the token
// total that only reaches its true value on a response's last line.
func TestReReadCompletesAStepWithoutDegradingIt(t *testing.T) {
	ctx := context.Background()
	s := stepStore(t)
	at := time.Now().UTC()

	partial := step("a", 1, usage.StepEdit, at)
	partial.Tokens = 100
	if _, _, err := s.InsertSteps(ctx, []usage.Step{partial}); err != nil {
		t.Fatalf("InsertSteps: %v", err)
	}

	complete := partial
	complete.Outcome = usage.OutcomeOK
	complete.TargetRef = 7
	complete.Tokens = 250
	if _, _, err := s.InsertSteps(ctx, []usage.Step{complete}); err != nil {
		t.Fatalf("restate: %v", err)
	}
	var outcome string
	var target, tokens int64
	row := s.db.QueryRowContext(ctx,
		`SELECT outcome, target_ref, tokens FROM session_step WHERE dedupe_key = 'a'`)
	if err := row.Scan(&outcome, &target, &tokens); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if outcome != usage.OutcomeOK || target != 7 || tokens != 250 {
		t.Errorf("restated row = (%q,%d,%d), want (ok,7,250)", outcome, target, tokens)
	}

	// A shorter re-read must not undo any of it.
	if _, _, err := s.InsertSteps(ctx, []usage.Step{partial}); err != nil {
		t.Fatalf("shorter re-read: %v", err)
	}
	row = s.db.QueryRowContext(ctx,
		`SELECT outcome, target_ref, tokens FROM session_step WHERE dedupe_key = 'a'`)
	if err := row.Scan(&outcome, &target, &tokens); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if outcome != usage.OutcomeOK || target != 7 || tokens != 250 {
		t.Errorf("a shorter re-read degraded the row to (%q,%d,%d)", outcome, target, tokens)
	}
}

// The horizon is a size bound, so this is the test that has to be able to fail: widen the
// window and the old step survives, which is what a broken bound looks like.
func TestPruneStepsDropsOnlyWhatIsPastTheHorizon(t *testing.T) {
	ctx := context.Background()
	s := stepStore(t)
	now := time.Now().UTC()
	old := now.AddDate(0, 0, -40)
	if _, _, err := s.InsertSteps(ctx, []usage.Step{
		step("old", 1, usage.StepRead, old),
		step("new", 2, usage.StepRead, now),
	}); err != nil {
		t.Fatalf("InsertSteps: %v", err)
	}
	removed, err := s.PruneSteps(ctx, now.AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("PruneSteps: %v", err)
	}
	if removed != 1 {
		t.Errorf("pruned %d rows, want 1", removed)
	}
	h, err := s.Steps(ctx)
	if err != nil {
		t.Fatalf("Steps: %v", err)
	}
	if h.Steps != 1 {
		t.Fatalf("steps after prune = %d, want 1", h.Steps)
	}
	if h.Oldest.Before(now.Add(-time.Hour)) {
		t.Errorf("the surviving step is dated %s: the wrong row was kept", h.Oldest)
	}
}

// A kind or outcome outside the closed vocabulary is rejected at the boundary. Storing it
// would render a category no validator can interpret and nobody defined.
func TestStepsOutsideTheVocabularyAreRejected(t *testing.T) {
	ctx := context.Background()
	s := stepStore(t)
	at := time.Now().UTC()
	bad := []usage.Step{
		step("a", 1, "vibes", at),
		func() usage.Step {
			st := step("b", 2, usage.StepRead, at)
			st.Outcome = "probably-fine"
			return st
		}(),
	}
	n, rejected, err := s.InsertSteps(ctx, bad)
	if err != nil {
		t.Fatalf("InsertSteps: %v", err)
	}
	if n != 0 {
		t.Errorf("stored %d rows outside the vocabulary, want 0", n)
	}
	if rejected != len(bad) {
		t.Errorf("reported %d rejections, want %d: a silent skip is the loss skip-and-count exists to prevent", rejected, len(bad))
	}
}

// Erasing history has to erase the sequence too. Without this, `clear` left every step behind
// and the next backfill's INSERT OR IGNORE found them present and printed steps=0, telling the
// person who asked to forget their history that there was nothing there.
func TestClearErasesTheTimelineUnderTheSameScope(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	for _, tc := range []struct {
		name   string
		before time.Time
		tool   string
		want   int64
	}{
		{"everything", time.Time{}, "", 0},
		{"one tool", time.Time{}, "claude-code", 0},
		{"another tool is untouched", time.Time{}, "codex", 2},
		{"by date", now.AddDate(0, 0, -10), "", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := stepStore(t)
			if _, _, err := s.InsertSteps(ctx, []usage.Step{
				step("old", 1, usage.StepRead, now.AddDate(0, 0, -20)),
				step("new", 2, usage.StepRead, now),
			}); err != nil {
				t.Fatalf("InsertSteps: %v", err)
			}
			if _, err := s.Clear(ctx, tc.before, tc.tool); err != nil {
				t.Fatalf("Clear: %v", err)
			}
			h, err := s.Steps(ctx)
			if err != nil {
				t.Fatalf("Steps: %v", err)
			}
			if h.Steps != tc.want {
				t.Errorf("steps left after clear = %d, want %d", h.Steps, tc.want)
			}
		})
	}
}

// A forked sub-agent replays its origin's whole prefix under a new agent id, so the same
// message.id and tool_use.id legitimately belong to more than one sequence — 141 such ids in a
// 400-file sample of the maintainer's corpus. Keyed without the timeline, INSERT OR IGNORE kept
// whichever arrived first and 13,897 steps vanished across 43 sequences.
func TestTwoTimelinesMayShareADedupeKey(t *testing.T) {
	ctx := context.Background()
	s := stepStore(t)
	at := time.Now().UTC()

	origin := step("msg_shared", 1, usage.StepAssistant, at)
	fork := origin
	fork.Timeline = "fork1"

	if _, _, err := s.InsertSteps(ctx, []usage.Step{origin, fork}); err != nil {
		t.Fatalf("InsertSteps: %v", err)
	}
	h, err := s.Steps(ctx)
	if err != nil {
		t.Fatalf("Steps: %v", err)
	}
	if h.Steps != 2 {
		t.Fatalf("stored %d steps, want 2: the fork's copy was swallowed by the origin's key", h.Steps)
	}
	var forkOrdinal int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT ordinal FROM session_step WHERE timeline = 'fork1'`).Scan(&forkOrdinal); err != nil {
		t.Fatalf("the fork's row is missing entirely: %v", err)
	}
}
