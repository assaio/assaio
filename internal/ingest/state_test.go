package ingest

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

// claudeHome writes one Claude transcript under a fresh home and returns both paths.
func claudeHome(t *testing.T) (home, session string) {
	t.Helper()
	home = t.TempDir()
	session = filepath.Join(home, ".claude", "projects", "-home-dev-app", "s1.jsonl")
	write(t, session, claudeTurn)
	return home, session
}

const claudeTurn = `{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"

const claudeSecondTurn = `{"type":"assistant","uuid":"a2","timestamp":"2026-07-01T11:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"

func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// runClaude ingests home and returns the claude-code result.
func runClaude(t *testing.T, st *store.Store, home string, opts Options) Result {
	t.Helper()
	results, err := Run(context.Background(), home, st, config.Sources{}, nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Tool == "claude-code" {
			return r
		}
	}
	t.Fatal("no claude-code result")
	return Result{}
}

// touch rewrites a file with body and stamps it with mtime, so a test controls the exact
// signature the skip predicate reads.
func touch(t *testing.T, path, body string, mtime time.Time) {
	t.Helper()
	write(t, path, body)
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func TestSecondRunSkipsUnchangedInput(t *testing.T) {
	home, _ := claudeHome(t)
	st := openStore(t)

	first := runClaude(t, st, home, Options{})
	if first.Unchanged != 0 || first.Records != 1 {
		t.Fatalf("first run = %+v, want Unchanged=0 Records=1", first)
	}
	second := runClaude(t, st, home, Options{})
	if second.Unchanged != 1 || second.Records != 0 {
		t.Fatalf("second run = %+v, want Unchanged=1 Records=0", second)
	}
	if second.Files != 1 {
		t.Errorf("Files = %d, want the skipped input still counted as discovered", second.Files)
	}
}

func TestChangedInputIsReparsed(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, session string)
	}{
		{
			name: "size and content change",
			mutate: func(t *testing.T, session string) {
				write(t, session, claudeTurn+claudeSecondTurn)
			},
		},
		{
			name: "mtime change alone",
			mutate: func(t *testing.T, session string) {
				touch(t, session, claudeTurn, time.Now().Add(time.Hour))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home, session := claudeHome(t)
			st := openStore(t)
			runClaude(t, st, home, Options{})

			tt.mutate(t, session)

			again := runClaude(t, st, home, Options{})
			if again.Unchanged != 0 {
				t.Fatalf("Unchanged = %d, want 0 (a changed input must be re-parsed)", again.Unchanged)
			}
		})
	}
}

func TestFullReparsesEverything(t *testing.T) {
	home, _ := claudeHome(t)
	st := openStore(t)
	runClaude(t, st, home, Options{})

	full := runClaude(t, st, home, Options{Full: true})
	if full.Unchanged != 0 || full.Records != 1 {
		t.Fatalf("--full = %+v, want Unchanged=0 Records=1", full)
	}
	// The full run must leave state usable again, not clear it.
	after := runClaude(t, st, home, Options{})
	if after.Unchanged != 1 {
		t.Errorf("run after --full: Unchanged = %d, want 1", after.Unchanged)
	}
}

// TestStateIsScopedToBuild is the guard for store.Insert's restateSignals repair: state
// written by another build must never let this one skip an input.
func TestStateIsScopedToBuild(t *testing.T) {
	home, session := claudeHome(t)
	st := openStore(t)
	ctx := context.Background()

	fi, err := os.Stat(session)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.RecordIngest(ctx, "some-other-build", time.Now(), []store.IngestEntry{
		{Path: session, Tool: "claude-code", Size: fi.Size(), MtimeNS: fi.ModTime().UnixNano()},
	}); err != nil {
		t.Fatal(err)
	}

	res := runClaude(t, st, home, Options{})
	if res.Unchanged != 0 {
		t.Fatalf("Unchanged = %d, want 0 (another build's state must not be trusted)", res.Unchanged)
	}
}

// TestFailedInputIsNotRecorded keeps a broken file visible: recording it would drop it out
// of the Failed count on every later run, turning a persistent problem into a one-off.
func TestFailedInputIsNotRecorded(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	failing := func(io.Reader) ([]usage.Record, int, error) {
		return nil, 0, errors.New("scan transcript: corrupt trailing line")
	}
	path := filepath.Join(t.TempDir(), "session.jsonl")
	write(t, path, "irrelevant: parsing is stubbed for this test\n")

	sk, err := newSkipper(ctx, st, false)
	if err != nil {
		t.Fatal(err)
	}
	res, err := ingestSource(ctx, st, sk, source{tool: "claude-code", files: []string{path}, parse: failing}, make(projectCache))
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed != 1 {
		t.Fatalf("Failed = %d, want 1", res.Failed)
	}
	if err := sk.flush(ctx, st, time.Now()); err != nil {
		t.Fatal(err)
	}

	known, err := st.IngestState(ctx, buildIdentity())
	if err != nil {
		t.Fatal(err)
	}
	if _, recorded := known[path]; recorded {
		t.Error("a failed input must not be recorded, or it stops being retried and reported")
	}
}

func TestClineDirectorySignatureFollowsItsFiles(t *testing.T) {
	home := t.TempDir()
	taskDir := filepath.Join(home, ".cline", "data", "tasks", "task1")
	write(t, filepath.Join(taskDir, "ui_messages.json"),
		`[{"type":"say","say":"api_req_started","ts":1751360400000,"text":"{\"tokensIn\":100,\"tokensOut\":50,\"cacheWrites\":0,\"cacheReads\":0,\"cost\":0.01}"}]`)
	write(t, filepath.Join(taskDir, "task_metadata.json"),
		`{"model_usage":[{"ts":1751360400000,"model_id":"claude-sonnet-4-5"}]}`)
	st := openStore(t)

	clineResult := func(opts Options) Result {
		results, err := Run(context.Background(), home, st, config.Sources{}, nil, opts)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range results {
			if r.Tool == "cline" {
				return r
			}
		}
		t.Fatal("no cline result")
		return Result{}
	}

	clineResult(Options{})
	if got := clineResult(Options{}); got.Unchanged != 1 {
		t.Fatalf("Unchanged = %d, want 1", got.Unchanged)
	}

	write(t, filepath.Join(taskDir, "task_metadata.json"),
		`{"model_usage":[{"ts":1751360400000,"model_id":"claude-sonnet-4-5"},{"ts":1751360500000,"model_id":"claude-sonnet-4-5"}]}`)
	if got := clineResult(Options{}); got.Unchanged != 0 {
		t.Errorf("Unchanged = %d, want 0 (a change inside the task dir must invalidate it)", got.Unchanged)
	}
}

func TestBuildIdentityIsNeverEmpty(t *testing.T) {
	if buildIdentity() == "" {
		t.Fatal("buildIdentity() must never be empty; it is the skip predicate's key")
	}
}

const claudeZeroTokenTurn = `{"type":"assistant","uuid":"a9","timestamp":"2026-07-01T12:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":0,"output_tokens":0,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"
