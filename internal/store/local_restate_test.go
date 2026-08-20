package store

import (
	"context"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// liveTurn is one turn identified by dedupe key "k1", as a parser sees it at a given moment:
// a session still being written yields a turn whose later-attributed signals are not there yet.
func liveTurn(reads, errors, lines int64) usage.Record {
	return usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(), Model: "m",
		DedupeKey: "k1", Granularity: "turn", InputTokens: 10, OutputTokens: 5,
		ToolCalls: reads, ToolReads: reads, ToolErrors: errors, LinesAdded: lines,
	}
}

// storedTurn returns the activity the store holds for that turn.
func storedTurn(t *testing.T, st *Store) (reads, errors, lines int64) {
	t.Helper()
	row := st.db.QueryRowContext(context.Background(),
		`SELECT tool_reads, tool_errors, lines_added FROM usage_record WHERE dedupe_key = 'k1'`)
	if err := row.Scan(&reads, &errors, &lines); err != nil {
		t.Fatal(err)
	}
	return reads, errors, lines
}

// TestInsertLocalRestatesAHalfAttributedTurn is B68: a turn ingested while its session was
// still being written carries its tool calls but not the errors a later line attributes to
// it. Insert's repair refuses that row because it already has signals, so re-reading the
// finished file must be what corrects it -- otherwise friction reports a rate it knows is
// wrong.
func TestInsertLocalRestatesAHalfAttributedTurn(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	if _, _, err := st.InsertLocal(ctx, []usage.Record{liveTurn(3, 0, 12)}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.InsertLocal(ctx, []usage.Record{liveTurn(3, 2, 40)}); err != nil {
		t.Fatal(err)
	}

	reads, errors, lines := storedTurn(t, st)
	if reads != 3 || errors != 2 || lines != 40 {
		t.Fatalf("stored = (reads %d, errors %d, lines %d), want the finished turn's (3, 2, 40)", reads, errors, lines)
	}
}

// TestInsertLocalNeverDegradesAStoredTokenCount keeps MAX where the number is the vendor's
// own: a token count is read off the log, so a re-read that somehow sees less (a truncated or
// rotated file) must not lower a figure the vendor already stated.
func TestInsertLocalNeverDegradesAStoredTokenCount(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	rich := liveTurn(3, 2, 40)
	rich.InputTokens, rich.OutputTokens = 500, 900
	thin := liveTurn(1, 0, 0)
	thin.InputTokens, thin.OutputTokens = 10, 20

	if _, _, err := st.InsertLocal(ctx, []usage.Record{rich}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.InsertLocal(ctx, []usage.Record{thin}); err != nil {
		t.Fatal(err)
	}

	var in, out int64
	row := st.db.QueryRowContext(ctx, `SELECT input_tokens, output_tokens FROM usage_record WHERE dedupe_key = 'k1'`)
	if err := row.Scan(&in, &out); err != nil {
		t.Fatal(err)
	}
	if in != 500 || out != 900 {
		t.Fatalf("stored tokens = (%d, %d), want the vendor's stated (500, 900) intact", in, out)
	}
}

// TestInsertLocalLetsACorrectedRuleLowerActivity is the other half of that line, and the one
// v0.14 needed. An activity count is assaio's own reading -- which lines are edits, which turn
// a result belongs to -- so when a build corrects that rule downward the correction has to be
// able to reach a stored row. Under MAX it could not: the row already exists, so nothing new
// is inserted, and a maximum never accepts a smaller number. Every figure the duplicate-line
// fix lowers would have been pinned at its inflated value forever.
func TestInsertLocalLetsACorrectedRuleLowerActivity(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	if _, _, err := st.InsertLocal(ctx, []usage.Record{liveTurn(3, 2, 40)}); err != nil {
		t.Fatal(err)
	}
	// The same file re-read by a build that no longer counts a repeated line twice.
	if _, _, err := st.InsertLocal(ctx, []usage.Record{liveTurn(3, 1, 20)}); err != nil {
		t.Fatal(err)
	}

	reads, errors, lines := storedTurn(t, st)
	if reads != 3 || errors != 1 || lines != 20 {
		t.Fatalf("stored = (reads %d, errors %d, lines %d), want the corrected (3, 1, 20)", reads, errors, lines)
	}
}

// TestInsertLocalCountsOnlyNewRows keeps the reported figure meaning "records that did not
// exist before", so a restatement never inflates what backfill prints.
func TestInsertLocalCountsOnlyNewRows(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	first, _, err := st.InsertLocal(ctx, []usage.Record{liveTurn(3, 0, 12)})
	if err != nil {
		t.Fatal(err)
	}
	again, _, err := st.InsertLocal(ctx, []usage.Record{liveTurn(3, 2, 40)})
	if err != nil {
		t.Fatal(err)
	}
	if first != 1 || again != 0 {
		t.Fatalf("inserted = (%d, %d), want (1, 0)", first, again)
	}
}

// TestInsertLeavesAHalfAttributedTurnAlone pins the server's contract: pushed records are
// first-write-wins, so a second client cannot restate a row another one already wrote. Only
// the local file-reading path, which owns the file it re-reads, may correct a turn.
func TestInsertLeavesAHalfAttributedTurnAlone(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	if _, err := st.Insert(ctx, []usage.Record{liveTurn(3, 0, 12)}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Insert(ctx, []usage.Record{liveTurn(3, 2, 40)}); err != nil {
		t.Fatal(err)
	}

	reads, errors, lines := storedTurn(t, st)
	if reads != 3 || errors != 0 || lines != 12 {
		t.Fatalf("stored = (reads %d, errors %d, lines %d), want the first write (3, 0, 12) untouched", reads, errors, lines)
	}
}

// A build that learns a record summarizes a whole run rather than one request has to be able
// to say so on rows already stored -- the sub-agent aggregate, mislabelled as a turn until
// v0.10. MAX would keep 'turn' forever, since it sorts above 'session'.
func TestInsertLocalRelabelsGranularityFromTheCurrentParse(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	if _, _, err := st.InsertLocal(ctx, []usage.Record{liveTurn(1, 0, 5)}); err != nil {
		t.Fatal(err)
	}
	aggregate := liveTurn(1, 0, 5)
	aggregate.Granularity = "session"
	if _, _, err := st.InsertLocal(ctx, []usage.Record{aggregate}); err != nil {
		t.Fatal(err)
	}

	if got := storedGranularity(t, st); got != "session" {
		t.Fatalf("granularity = %q, want session after re-reading the file that owns it", got)
	}
}

// The sync path stays first-write-wins: a teammate's push must never relabel what another
// member's own parse recorded.
func TestInsertLeavesAStoredGranularityAlone(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	if _, _, err := st.InsertLocal(ctx, []usage.Record{liveTurn(1, 0, 5)}); err != nil {
		t.Fatal(err)
	}
	aggregate := liveTurn(1, 0, 5)
	aggregate.Granularity = "session"
	if _, err := st.Insert(ctx, []usage.Record{aggregate}); err != nil {
		t.Fatal(err)
	}

	if got := storedGranularity(t, st); got != "turn" {
		t.Fatalf("granularity = %q, want turn: a synced record may not restate a local one", got)
	}
}

func storedGranularity(t *testing.T, st *Store) string {
	t.Helper()
	var g string
	row := st.db.QueryRowContext(context.Background(),
		`SELECT granularity FROM usage_record WHERE dedupe_key = 'k1'`)
	if err := row.Scan(&g); err != nil {
		t.Fatal(err)
	}
	return g
}

// Migration 0006 is the half a re-parse cannot reach: once a sub-agent has its own
// transcript the parent's aggregate is suppressed, so the row already stored is never
// offered again and keeps claiming to be one turn.
func TestMigration0006RelabelsStoredSubAgentAggregates(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	rows := []usage.Record{
		{Tool: "claude-code", SessionID: "s", Timestamp: time.Now().UTC(), Model: "m", DedupeKey: "agent:a1", Granularity: "turn", OutputTokens: 9},
		{Tool: "claude-code", SessionID: "s", Timestamp: time.Now().UTC(), Model: "m", DedupeKey: "bob:agent:a2", Granularity: "turn", OutputTokens: 9},
		{Tool: "claude-code", SessionID: "s", Timestamp: time.Now().UTC(), Model: "m", DedupeKey: "plain-uuid", Granularity: "turn", OutputTokens: 9},
		{Tool: "cline", SessionID: "agent", Timestamp: time.Now().UTC(), Model: "m", DedupeKey: "agent:0", Granularity: "turn", OutputTokens: 9},
	}
	if _, err := st.Insert(ctx, rows); err != nil {
		t.Fatal(err)
	}
	// The migration ran at Open, before these rows existed; re-running it is what the test is.
	body, err := migrations.ReadFile("migrations/0006_subagent_session_grain.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(ctx, string(body)); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ key, want string }{
		{"agent:a1", "session"},
		{"bob:agent:a2", "session"},
		{"plain-uuid", "turn"},
		{"agent:0", "turn"}, // another tool's key scheme must not be matched by coincidence
	} {
		var got string
		row := st.db.QueryRowContext(ctx, `SELECT granularity FROM usage_record WHERE dedupe_key = ?`, tc.key)
		if err := row.Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("%s granularity = %q, want %q", tc.key, got, tc.want)
		}
	}
}
