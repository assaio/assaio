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

	if _, err := st.InsertLocal(ctx, []usage.Record{liveTurn(3, 0, 12)}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertLocal(ctx, []usage.Record{liveTurn(3, 2, 40)}); err != nil {
		t.Fatal(err)
	}

	reads, errors, lines := storedTurn(t, st)
	if reads != 3 || errors != 2 || lines != 40 {
		t.Fatalf("stored = (reads %d, errors %d, lines %d), want the finished turn's (3, 2, 40)", reads, errors, lines)
	}
}

// TestInsertLocalNeverDegradesAStoredRow keeps the honesty guarantee Insert already makes:
// a re-parse that extracts less than what is stored must not lower it.
func TestInsertLocalNeverDegradesAStoredRow(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	if _, err := st.InsertLocal(ctx, []usage.Record{liveTurn(3, 2, 40)}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertLocal(ctx, []usage.Record{liveTurn(1, 0, 0)}); err != nil {
		t.Fatal(err)
	}

	reads, errors, lines := storedTurn(t, st)
	if reads != 3 || errors != 2 || lines != 40 {
		t.Fatalf("stored = (reads %d, errors %d, lines %d), want the richer (3, 2, 40) intact", reads, errors, lines)
	}
}

// TestInsertLocalCountsOnlyNewRows keeps the reported figure meaning "records that did not
// exist before", so a restatement never inflates what backfill prints.
func TestInsertLocalCountsOnlyNewRows(t *testing.T) {
	ctx := context.Background()
	st := openTempStore(t)

	first, err := st.InsertLocal(ctx, []usage.Record{liveTurn(3, 0, 12)})
	if err != nil {
		t.Fatal(err)
	}
	again, err := st.InsertLocal(ctx, []usage.Record{liveTurn(3, 2, 40)})
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
