package claude

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/usage"
)

func TestSubagentID(t *testing.T) {
	cases := map[string]string{
		"/p/session/subagents/agent-a16d8eac854b22bd6.jsonl": "a16d8eac854b22bd6",
		"agent-abc.jsonl": "abc",
		"/p/session/subagents/workflows/wf_1/agent-a0084420a4f553fc1.jsonl": "a0084420a4f553fc1",
	}
	for path, want := range cases {
		if got := SubagentID(path); got != want {
			t.Errorf("SubagentID(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestSuppressCovered(t *testing.T) {
	recs := []usage.Record{
		{DedupeKey: "uuid-1"},                 // an ordinary assistant turn: always kept
		{DedupeKey: agentDedupePrefix + "aa"}, // covered aggregate: dropped (file exists)
		{DedupeKey: agentDedupePrefix + "bb"}, // uncovered aggregate: kept as fallback
		{DedupeKey: "uuid-2"},
	}
	covered := map[string]struct{}{"aa": {}}
	got := SuppressCovered(recs, covered)

	var keys []string
	for _, r := range got {
		keys = append(keys, r.DedupeKey)
	}
	sort.Strings(keys)
	want := []string{"agent:bb", "uuid-1", "uuid-2"}
	if strings.Join(keys, ",") != strings.Join(want, ",") {
		t.Fatalf("SuppressCovered kept %v, want %v", keys, want)
	}
}

func TestSuppressCoveredEmptySetIsNoOp(t *testing.T) {
	recs := []usage.Record{{DedupeKey: agentDedupePrefix + "aa"}}
	if got := SuppressCovered(recs, nil); len(got) != 1 {
		t.Fatalf("empty covered set dropped records: got %d, want 1", len(got))
	}
}

func TestDiscoverSubagents(t *testing.T) {
	root := t.TempDir()
	sess := filepath.Join(root, "-enc-cwd", "session-1")
	mustWrite(t, filepath.Join(root, "-enc-cwd", "session-1.jsonl")) // parent transcript: not a sub-agent
	mustWrite(t, filepath.Join(sess, "subagents", "agent-aa.jsonl"))
	mustWrite(t, filepath.Join(sess, "subagents", "agent-aa.meta.json")) // sidecar: excluded
	mustWrite(t, filepath.Join(sess, "subagents", "workflows", "wf_1", "agent-bb.jsonl"))
	mustWrite(t, filepath.Join(sess, "tool-results", "agent-cc.jsonl")) // not under subagents/: excluded

	found, err := DiscoverSubagents(root)
	if err != nil {
		t.Fatal(err)
	}
	ids := CoveredAgents(found)
	if len(ids) != 2 {
		t.Fatalf("discovered %d sub-agent files %v, want 2 (aa, bb)", len(ids), found)
	}
	if _, ok := ids["aa"]; !ok {
		t.Error("missing sub-agent aa")
	}
	if _, ok := ids["bb"]; !ok {
		t.Error("missing nested workflow sub-agent bb")
	}
}

// TestParseClampsNegativeTokens covers the parser contract's NonNeg requirement: a
// syntactically valid line carrying a negative token count must never reach a record.
func TestParseClampsNegativeTokens(t *testing.T) {
	line := `{"type":"assistant","uuid":"u1","message":{"model":"claude-x","usage":{"input_tokens":-500,"output_tokens":10,"cache_read_input_tokens":-1}}}`
	recs, _, err := Parse(strings.NewReader(line))
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	if recs[0].InputTokens != 0 || recs[0].CacheReadTokens != 0 || recs[0].OutputTokens != 10 {
		t.Fatalf("negative tokens not clamped: %+v", recs[0])
	}
}

func mustWrite(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
}
