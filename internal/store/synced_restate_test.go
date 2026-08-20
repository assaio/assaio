package store

import (
	"context"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// syncedTurn is one member's pushed turn, dedupe key already carrying the member prefix the
// sync endpoint composes.
func syncedTurn(out, lines int64, model string) usage.Record {
	return usage.Record{
		Tool: "claude-code", SessionID: "s1", Timestamp: time.Now().UTC(), Model: model,
		DedupeKey: "alice:k1", Member: "alice", Granularity: "turn",
		InputTokens: 10, OutputTokens: out, LinesAdded: lines,
	}
}

// TestInsertSyncedCorrectsAPartialPush is the team-server half of the live-session problem.
// A Claude response's output count only reaches its true total on the response's last line,
// so a sync that runs mid-stream pushes a partial figure. Routed through first-write-wins
// Insert, that undercount was permanent: the row exists, so nothing new is inserted and the
// signals-only repair never touches a token count. The member prefix gives each row exactly
// one possible writer, so restating it is that member correcting their own number.
func TestInsertSyncedCorrectsAPartialPush(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	if _, err := st.InsertSynced(ctx, []usage.Record{syncedTurn(10, 3, "m")}); err != nil {
		t.Fatal(err)
	}
	inserted, err := st.InsertSynced(ctx, []usage.Record{syncedTurn(20, 9, "m")})
	if err != nil {
		t.Fatal(err)
	}
	if inserted != 0 {
		t.Fatalf("inserted = %d, want 0 -- a re-push corrects a row, it does not add one", inserted)
	}

	var out, lines int64
	row := st.db.QueryRowContext(ctx,
		`SELECT output_tokens, lines_added FROM usage_record WHERE dedupe_key = 'alice:k1'`)
	if err := row.Scan(&out, &lines); err != nil {
		t.Fatal(err)
	}
	if out != 20 || lines != 9 {
		t.Fatalf("stored = (out %d, lines %d), want the completed response's (20, 9)", out, lines)
	}
}

// TestRestateFillsAMissingModelOnly: a source can emit a record before it knows the model
// (Cline reads it from a sidecar that may not exist yet), and those tokens are unpriceable
// until it does. Filling a blank is a repair; overwriting a name would break the rule that the
// first read is the authority on identity.
func TestRestateFillsAMissingModelOnly(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	if _, _, err := st.InsertLocal(ctx, []usage.Record{syncedTurn(10, 3, "")}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.InsertLocal(ctx, []usage.Record{syncedTurn(10, 3, "claude-opus-5")}); err != nil {
		t.Fatal(err)
	}
	if got := storedModel(t, st); got != "claude-opus-5" {
		t.Fatalf("model = %q, want the name the later read supplied", got)
	}

	if _, _, err := st.InsertLocal(ctx, []usage.Record{syncedTurn(10, 3, "something-else")}); err != nil {
		t.Fatal(err)
	}
	if got := storedModel(t, st); got != "claude-opus-5" {
		t.Fatalf("model = %q, want the first answer kept -- a stored name is never overwritten", got)
	}
}

func storedModel(t *testing.T, st *Store) string {
	t.Helper()
	var model string
	row := st.db.QueryRowContext(context.Background(),
		`SELECT model FROM usage_record WHERE dedupe_key = 'alice:k1'`)
	if err := row.Scan(&model); err != nil {
		t.Fatal(err)
	}
	return model
}
