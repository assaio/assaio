package ingest

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/usage"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRunIngestsAllTools(t *testing.T) {
	home := t.TempDir()
	write(t, filepath.Join(home, ".claude", "projects", "-home-dev-app", "s1.jsonl"),
		`{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s1","message":{"model":"claude-opus-4-5","usage":{"input_tokens":10,"output_tokens":20,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}`+"\n")
	write(t, filepath.Join(home, ".codex", "sessions", "2026", "07", "01", "rollout-x-c1.jsonl"),
		`{"type":"session_meta","payload":{"id":"c1","cwd":"/x","timestamp":"2026-07-01T09:00:00Z"}}`+"\n"+
			`{"type":"turn_context","payload":{"model":"gpt-5.1"}}`+"\n"+
			`{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":0,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}`+"\n")
	write(t, filepath.Join(home, ".gemini", "tmp", "abc123", "chats", "session-g1.jsonl"),
		`{"sessionId":"g1","timestamp":"2026-07-01T11:00:00Z","model":"gemini-2.5-pro","tokens":{"input":100,"output":50,"cached":0,"thoughts":0,"tool":0,"total":150}}`+"\n")
	write(t, filepath.Join(home, ".cline", "data", "tasks", "task1", "ui_messages.json"),
		`[{"type":"say","say":"api_req_started","ts":1751360400000,"text":"{\"tokensIn\":100,\"tokensOut\":50,\"cacheWrites\":0,\"cacheReads\":0,\"cost\":0.01}"}]`)
	write(t, filepath.Join(home, ".cline", "data", "tasks", "task1", "task_metadata.json"),
		`{"model_usage":[{"ts":1751360400000,"model_id":"claude-sonnet-4-5"}]}`)

	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	results, err := Run(context.Background(), home, st, config.Sources{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	byTool := map[string]Result{}
	for _, r := range results {
		byTool[r.Tool] = r
	}
	if byTool["claude-code"].Inserted != 1 || byTool["codex"].Inserted != 1 || byTool["gemini-cli"].Inserted != 1 ||
		byTool["cline"].Inserted != 1 {
		t.Fatalf("results = %+v", results)
	}
	n, _ := st.Count(context.Background())
	if n != 4 {
		t.Fatalf("Count = %d want 4", n)
	}
}

func TestRunContinuesPastUnreadableFile(t *testing.T) {
	home := t.TempDir()
	badDir := filepath.Join(home, ".claude", "projects", "-home-dev-bad")
	goodDir := filepath.Join(home, ".claude", "projects", "-home-dev-good")
	if err := os.MkdirAll(badDir, 0o750); err != nil {
		t.Fatal(err)
	}
	// A directory named like a transcript file: os.Open will fail to read it as a file.
	if err := os.MkdirAll(filepath.Join(badDir, "s1.jsonl"), 0o750); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(goodDir, "s2.jsonl"),
		`{"type":"assistant","uuid":"a1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s2","message":{"model":"claude-opus-4-5","usage":{"input_tokens":10,"output_tokens":20}}}`+"\n")

	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	results, err := Run(context.Background(), home, st, config.Sources{}, nil)
	if err != nil {
		t.Fatalf("a single bad file must not abort the run: %v", err)
	}
	byTool := map[string]Result{}
	for _, r := range results {
		byTool[r.Tool] = r
	}
	claudeResult := byTool["claude-code"]
	if claudeResult.Files != 2 {
		t.Fatalf("Files = %d want 2", claudeResult.Files)
	}
	if claudeResult.Failed != 1 {
		t.Fatalf("Failed = %d want 1", claudeResult.Failed)
	}
	if claudeResult.Inserted != 1 {
		t.Fatalf("Inserted = %d want 1 (good file still processed)", claudeResult.Inserted)
	}
}

// TestIngestSourceInsertsRecordsRecoveredBeforeAParseError guards skip-and-count: a
// parser can hit a fatal condition partway through a file (e.g. a corrupt or oversized
// trailing line) and still return every record it recovered before that point alongside
// a non-nil error. That partial success must still be inserted -- the file counts as
// Failed, but its good records must not be silently discarded.
func TestIngestSourceInsertsRecordsRecoveredBeforeAParseError(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	parseErr := errors.New("scan claude transcript: corrupt trailing line")
	partial := func(io.Reader) ([]usage.Record, int, error) {
		return []usage.Record{
			{
				Tool: "claude-code", SessionID: "s1", Timestamp: time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
				Model: "claude-opus-4-5", InputTokens: 10, OutputTokens: 20, DedupeKey: "good-1",
			},
		}, 1, parseErr
	}
	path := filepath.Join(t.TempDir(), "session.jsonl")
	write(t, path, "irrelevant: parsing is stubbed for this test\n")

	s := source{tool: "claude-code", files: []string{path}, parse: partial}
	res, err := ingestSource(context.Background(), st, s, make(projectCache))
	if err != nil {
		t.Fatalf("ingestSource() err = %v, want nil (a per-file parse error must not abort the run)", err)
	}
	if res.Failed != 1 {
		t.Fatalf("Failed = %d, want 1", res.Failed)
	}
	if res.Records != 1 || res.Skipped != 1 {
		t.Fatalf("Records=%d Skipped=%d, want 1/1", res.Records, res.Skipped)
	}
	if res.Inserted != 1 {
		t.Fatalf("Inserted = %d, want 1 (the good record recovered before the parse error must still land)", res.Inserted)
	}

	n, err := st.Count(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("Count = %d, want 1 (the good record must actually be persisted, not just counted)", n)
	}
}

func TestRunUsesConfiguredClaudeRootInsteadOfDefault(t *testing.T) {
	home := t.TempDir()
	// Default location: must be ignored once sources.claude is configured.
	write(t, filepath.Join(home, ".claude", "projects", "-default", "default.jsonl"),
		`{"type":"assistant","uuid":"default-1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s-default","message":{"model":"claude-opus-4-5","usage":{"input_tokens":10,"output_tokens":20}}}`+"\n")
	// Configured root: must be the only place discovery looks.
	customRoot := filepath.Join(home, "custom-claude-logs")
	write(t, filepath.Join(customRoot, "-custom-project", "custom.jsonl"),
		`{"type":"assistant","uuid":"custom-1","timestamp":"2026-07-01T10:00:00Z","sessionId":"s-custom","message":{"model":"claude-opus-4-5","usage":{"input_tokens":10,"output_tokens":20}}}`+"\n")

	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	sources := config.Sources{Claude: []string{customRoot}}
	results, err := Run(context.Background(), home, st, sources, nil)
	if err != nil {
		t.Fatal(err)
	}
	byTool := map[string]Result{}
	for _, r := range results {
		byTool[r.Tool] = r
	}
	claudeResult := byTool["claude-code"]
	if claudeResult.Files != 1 || claudeResult.Inserted != 1 {
		t.Fatalf("claude-code result = %+v, want exactly the 1 file under the configured root (default root ignored)", claudeResult)
	}
}

func TestRunIngestsConfiguredPlugin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin test fixtures are POSIX shell scripts")
	}
	home := t.TempDir()
	pluginScript, err := filepath.Abs(filepath.Join("..", "plugin", "testdata", "good.sh"))
	if err != nil {
		t.Fatal(err)
	}

	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	plugins := []config.PluginConfig{{Name: "demo", Command: pluginScript, Timeout: "5s"}}
	results, err := Run(context.Background(), home, st, config.Sources{}, plugins)
	if err != nil {
		t.Fatal(err)
	}
	byTool := map[string]Result{}
	for _, r := range results {
		byTool[r.Tool] = r
	}
	demoResult, ok := byTool["plugin:demo"]
	if !ok {
		t.Fatalf("no plugin:demo result in %+v", results)
	}
	if demoResult.Records != 2 || demoResult.Inserted != 2 {
		t.Fatalf("demoResult = %+v, want Records=2 Inserted=2", demoResult)
	}
	n, _ := st.Count(context.Background())
	if n != 2 {
		t.Fatalf("Count = %d want 2", n)
	}
}

func TestRunPluginFailureCountsAsFailedAndContinues(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("plugin test fixtures are POSIX shell scripts")
	}
	home := t.TempDir()
	badScript, err := filepath.Abs(filepath.Join("..", "plugin", "testdata", "handshake_mismatch.sh"))
	if err != nil {
		t.Fatal(err)
	}
	goodScript, err := filepath.Abs(filepath.Join("..", "plugin", "testdata", "good_second.sh"))
	if err != nil {
		t.Fatal(err)
	}

	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	plugins := []config.PluginConfig{
		{Name: "demo", Command: badScript, Timeout: "5s"},
		{Name: "good", Command: goodScript, Timeout: "5s"},
	}
	results, err := Run(context.Background(), home, st, config.Sources{}, plugins)
	if err != nil {
		t.Fatalf("a failing plugin must not abort the run: %v", err)
	}
	byTool := map[string]Result{}
	for _, r := range results {
		byTool[r.Tool] = r
	}
	if byTool["plugin:demo"].Failed != 1 {
		t.Fatalf("plugin:demo Failed = %d, want 1", byTool["plugin:demo"].Failed)
	}
	if byTool["plugin:good"].Inserted != 2 {
		t.Fatalf("plugin:good Inserted = %d, want 2 (run continues past the failed plugin)", byTool["plugin:good"].Inserted)
	}
}
