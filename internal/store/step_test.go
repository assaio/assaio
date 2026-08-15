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
// answering line had not been written yet, and the token total that only reaches its true value
// on a response's last line. Both are monotone in the prefix read, so neither may go backwards.
func TestReReadCompletesAStepWithoutDegradingIt(t *testing.T) {
	ctx := context.Background()
	s := stepStore(t)
	at := time.Now().UTC()

	partial := step("a", 1, usage.StepEdit, at)
	partial.Tokens = 100
	// The target is the same in both reads: it is settled on the line that creates the step, so
	// a longer prefix never completes it. What a re-read may do to it is the next test.
	partial.TargetRef = 7
	if _, _, err := s.InsertSteps(ctx, []usage.Step{partial}); err != nil {
		t.Fatalf("InsertSteps: %v", err)
	}

	complete := partial
	complete.Outcome = usage.OutcomeOK
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

	// Every column here follows the current parse, outcome included: ingest re-reads whole
	// files, so a later read of an append-only transcript is a superset, and the alternative --
	// keeping the stored answer -- is what stops a corrected rule reaching a stored row. See
	// TestRestateLowersAStepTotalFromACorrectedRule.
	if _, _, err := s.InsertSteps(ctx, []usage.Step{partial}); err != nil {
		t.Fatalf("re-read: %v", err)
	}
	row = s.db.QueryRowContext(ctx,
		`SELECT outcome, target_ref FROM session_step WHERE dedupe_key = 'a'`)
	if err := row.Scan(&outcome, &target); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if outcome != "" || target != 7 {
		t.Errorf("re-read row = (%q,%d), want the current parse's own answer", outcome, target)
	}
}

// TestRestateLowersAStepOutcomeFromACorrectedRule is the same B116 argument one column over.
// `outcome` is not a vendor field: stopOutcome maps a stop reason and resolveResults attributes
// a result to a call, and steps.go records that the attribution rule has already been wrong once
// -- 42 of 497 real denials landed on the wrong call. A fill-only CASE kept the wrong answer.
func TestRestateLowersAStepOutcomeFromACorrectedRule(t *testing.T) {
	ctx := context.Background()
	s := stepStore(t)

	wrong := step("a", 1, usage.StepEdit, time.Now().UTC())
	wrong.Outcome = usage.OutcomeOK
	if _, _, err := s.InsertSteps(ctx, []usage.Step{wrong}); err != nil {
		t.Fatalf("InsertSteps: %v", err)
	}
	corrected := wrong
	corrected.Outcome = usage.OutcomeDenied
	if _, _, err := s.InsertSteps(ctx, []usage.Step{corrected}); err != nil {
		t.Fatalf("restate: %v", err)
	}
	var outcome string
	if err := s.db.QueryRowContext(ctx,
		`SELECT outcome FROM session_step WHERE dedupe_key = 'a'`).Scan(&outcome); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if outcome != usage.OutcomeDenied {
		t.Errorf("outcome = %q, want %q: a corrected attribution could not reach the stored row", outcome, usage.OutcomeDenied)
	}
}

// TestRestateLowersAStepTotalFromACorrectedRule is B116 in the newest table: `tokens` is not a
// vendor figure but assaio's own sum of four chosen fields, so a build that corrects which
// fields it sums has to be able to reach every stored row. Under MAX -- shipped in v0.20 -- an
// inflated 1,000,000 survived both a `--full` re-read and a restate offering 10, permanently
// under `trace.horizon_days: 0`, where nothing ages the row out.
func TestRestateLowersAStepTotalFromACorrectedRule(t *testing.T) {
	ctx := context.Background()
	s := stepStore(t)
	at := time.Now().UTC()

	inflated := step("a", 1, usage.StepAssistant, at)
	inflated.Tokens = 1_000_000
	if _, _, err := s.InsertSteps(ctx, []usage.Step{inflated}); err != nil {
		t.Fatalf("InsertSteps: %v", err)
	}
	corrected := inflated
	corrected.Tokens = 10
	if _, _, err := s.InsertSteps(ctx, []usage.Step{corrected}); err != nil {
		t.Fatalf("restate: %v", err)
	}
	var tokens int64
	if err := s.db.QueryRowContext(ctx,
		`SELECT tokens FROM session_step WHERE dedupe_key = 'a'`).Scan(&tokens); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if tokens != 10 {
		t.Errorf("tokens = %d, want 10: a corrected rule could not reach the stored row", tokens)
	}
}

// Position is a claim the current parse owns, not a value a later parse can only raise. Widening
// which calls register a target renumbers first-seen order, and keeping the higher of the two
// would leave one sequence holding a mix of two parsers' numbering: a 7 beside a 3 that stands
// for the same file, with nothing downstream able to tell they disagree.
func TestReReadRenumbersPositionRatherThanKeepingTheHigher(t *testing.T) {
	ctx := context.Background()
	s := stepStore(t)
	at := time.Now().UTC()

	before := step("a", 9, usage.StepEdit, at)
	before.TargetRef = 7
	if _, _, err := s.InsertSteps(ctx, []usage.Step{before}); err != nil {
		t.Fatalf("InsertSteps: %v", err)
	}
	after := before
	after.Ordinal = 4
	after.TargetRef = 3
	if _, _, err := s.InsertSteps(ctx, []usage.Step{after}); err != nil {
		t.Fatalf("re-read: %v", err)
	}
	var ordinal, target int64
	row := s.db.QueryRowContext(ctx,
		`SELECT ordinal, target_ref FROM session_step WHERE dedupe_key = 'a'`)
	if err := row.Scan(&ordinal, &target); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if ordinal != 4 || target != 3 {
		t.Errorf("re-read row = (ordinal %d, target %d), want (4, 3)", ordinal, target)
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
