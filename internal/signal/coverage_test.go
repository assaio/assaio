package signal

import (
	"math"
	"testing"
)

func supportFor(t *testing.T, byTool map[string]int64, id string) Support {
	t.Helper()
	support := Coverage(byTool)
	for i := range support {
		if support[i].Signal.ID == id {
			return support[i]
		}
	}
	t.Fatalf("%s missing from coverage", id)
	return Support{}
}

func TestCoverage(t *testing.T) {
	tests := []struct {
		name        string
		byTool      map[string]int64
		signal      string
		wantVerdict string
		wantShare   float64
	}{
		{
			"a deep source answers everything",
			map[string]int64{"claude-code": 100},
			"ai.lines.added", Full, 1,
		},
		{
			"a token-only source cannot answer activity at all",
			map[string]int64{"gemini-cli": 100},
			"ai.lines.added", None, 0,
		},
		{
			"a token-only source still answers tokens",
			map[string]int64{"gemini-cli": 100},
			"ai.tokens.total", Full, 1,
		},
		{
			"mixing sources makes activity partial, and names the share",
			map[string]int64{"claude-code": 75, "gemini-cli": 25},
			"ai.lines.added", Partial, 0.75,
		},
		{
			"only claude-code labels skills, so a mixed window is partial",
			map[string]int64{"claude-code": 50, "codex": 50},
			"ai.skill.tokens", Partial, 0.5,
		},
		{
			"codex carries no attribution at all",
			map[string]int64{"codex": 100},
			"ai.agent.tokens", None, 0,
		},
		{
			"an out-of-tree plugin counts for tokens",
			map[string]int64{"plugin:weekend": 100},
			"ai.tokens.total", Full, 1,
		},
		{
			"an out-of-tree plugin cannot carry activity: the protocol has no such field",
			map[string]int64{"plugin:weekend": 100},
			"ai.edits.count", None, 0,
		},
		{
			"copilot totals a session, so it cannot contribute a turn count",
			map[string]int64{"copilot-cli": 100},
			"ai.turns.count", None, 0,
		},
		{
			"copilot does record changed lines, though",
			map[string]int64{"copilot-cli": 100},
			"ai.lines.added", Full, 1,
		},
		{
			"copilot carries no edit count despite carrying lines",
			map[string]int64{"copilot-cli": 100},
			"ai.edits.count", None, 0,
		},
		{
			"codex marks failures only for file edits, so it answers no tool-error signal",
			map[string]int64{"codex": 100},
			"ai.tool_errors.count", None, 0,
		},
		{
			"an empty window supports nothing rather than everything",
			map[string]int64{},
			"ai.tokens.total", None, 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := supportFor(t, tc.byTool, tc.signal)
			if got.Verdict != tc.wantVerdict {
				t.Errorf("verdict = %s, want %s", got.Verdict, tc.wantVerdict)
			}
			if math.Abs(got.Share-tc.wantShare) > 1e-9 {
				t.Errorf("share = %v, want %v", got.Share, tc.wantShare)
			}
		})
	}
}

// A source that cannot answer a signal must not be listed as one of its sources, or the
// command would name a tool as evidence for a figure it never contributed to.
func TestCoverageNamesOnlyCapableSources(t *testing.T) {
	got := supportFor(t, map[string]int64{"claude-code": 50, "gemini-cli": 50}, "ai.lines.added")
	if len(got.Sources) != 1 || got.Sources[0] != "claude-code" {
		t.Fatalf("sources = %v, want only claude-code", got.Sources)
	}
}

func TestCoverageCoversEverySignal(t *testing.T) {
	if got, want := len(Coverage(map[string]int64{"claude-code": 1})), len(Catalog()); got != want {
		t.Fatalf("coverage reported %d signals, catalog declares %d", got, want)
	}
}
