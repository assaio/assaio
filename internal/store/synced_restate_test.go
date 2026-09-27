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

// TestSyncedRestateNamesTheModelTheMemberStates: a synced row has exactly one writer, the
// member whose own re-read produced it, so the naming rule applies there as it does locally --
// a stated model replaces the stored one, and a blank keeps it.
func TestSyncedRestateNamesTheModelTheMemberStates(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	for _, step := range []struct {
		offered, want string
	}{
		{"", ""},
		{"claude-opus-5", "claude-opus-5"},
		{"", "claude-opus-5"},
		{"claude-opus-5-5", "claude-opus-5-5"},
	} {
		if _, err := st.InsertSynced(ctx, []usage.Record{syncedTurn(10, 3, step.offered)}); err != nil {
			t.Fatal(err)
		}
		if got := storedModel(t, st); got != step.want {
			t.Fatalf("after offering %q: model = %q, want %q", step.offered, got, step.want)
		}
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

// TestSyncedRestateKeepsTheEarliestTime: a member's store already keeps the earliest carrier's
// time, so a later time pushed from an older build or a rebuilt store is not a correction.
func TestSyncedRestateKeepsTheEarliestTime(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)
	early := syncedTurn(10, 3, "m")
	late := early
	late.Timestamp = early.Timestamp.Add(time.Minute)
	for _, r := range []usage.Record{early, late} {
		if _, err := st.InsertSynced(ctx, []usage.Record{r}); err != nil {
			t.Fatal(err)
		}
	}
	var ts string
	if err := st.db.QueryRowContext(ctx, `SELECT ts FROM usage_record`).Scan(&ts); err != nil {
		t.Fatal(err)
	}
	if want := early.Timestamp.UTC().Format(time.RFC3339); ts != want {
		t.Fatalf("ts = %q, want the earlier %q", ts, want)
	}
}
