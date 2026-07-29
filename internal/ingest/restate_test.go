package ingest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/store"
)

// storedToolErrors sums the failed tool calls the store holds, which is the column friction
// reads.
func storedToolErrors(t *testing.T, st *store.Store) int64 {
	t.Helper()
	rows, err := st.Usage(context.Background(), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for i := range rows {
		total += rows[i].ToolErrors
	}
	return total
}

// TestRunRestatesATurnIngestedMidSession is B68 end to end. A session ingested while it is
// still being written stores a turn whose tool calls are known but whose failures a later
// line has not attributed yet. Re-reading the finished file has to correct that turn: until
// it does, the row is frozen half-attributed and friction reports an error rate it can
// prove is wrong.
func TestRunRestatesATurnIngestedMidSession(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	path := filepath.Join(home, ".claude", "projects", "-home-dev-app", "s1.jsonl")
	const turn = `{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","content":[{"type":"tool_use","name":"Bash"},{"type":"tool_use","name":"Read"}],"usage":{"input_tokens":10,"output_tokens":20}}}` + "\n"
	const failure = `{"type":"user","uuid":"tr1","timestamp":"2026-07-01T10:00:01Z","sessionId":"s1","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","is_error":true}]}}` + "\n"

	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	write(t, path, turn)
	if _, err := Run(ctx, home, st, config.Sources{}, nil, Options{}); err != nil {
		t.Fatal(err)
	}
	if got := storedToolErrors(t, st); got != 0 {
		t.Fatalf("tool errors mid-session = %d, want 0: the log does not carry the failure yet", got)
	}

	write(t, path, turn+failure)
	if _, err := Run(ctx, home, st, config.Sources{}, nil, Options{}); err != nil {
		t.Fatal(err)
	}
	if got := storedToolErrors(t, st); got != 1 {
		t.Fatalf("tool errors after re-reading the finished session = %d, want 1", got)
	}
}
