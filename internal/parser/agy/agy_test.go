package agy

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/usage"
)

var update = flag.Bool("update", false, "update golden files")

func parseFixture(t *testing.T, name, conversationID string) []usage.Record {
	t.Helper()
	f, err := os.Open("testdata/" + name + ".jsonl") //nolint:gosec // a fixture named by this test file
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	recs, _, err := ParseTranscript(f, conversationID)
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

func TestParseGolden(t *testing.T) {
	for _, fixture := range []string{"session", "interrupted"} {
		t.Run(fixture, func(t *testing.T) {
			got, err := json.MarshalIndent(parseFixture(t, fixture, "conv-1"), "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, '\n')
			golden := "testdata/" + fixture + ".golden"
			if *update {
				if err := os.WriteFile(golden, got, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden) //nolint:gosec // a golden path derived from this test file's own fixture names
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("golden mismatch:\n got=%s\nwant=%s", got, want)
			}
		})
	}
}

// TestEveryTokenFieldIsZero is the claim this source is built around: Antigravity CLI publishes
// no input, output, cache or reasoning counter, so a record that carried one would have been
// invented. The depth row declares those signals unanswered, which is the other half of the
// statement -- a zero nobody declared is the fabricated figure, not the zero itself.
func TestEveryTokenFieldIsZero(t *testing.T) {
	recs := parseFixture(t, "session", "conv-1")
	for i := range recs {
		r := &recs[i]
		if r.InputTokens|r.OutputTokens|r.CacheReadTokens|r.CacheWriteTokens|
			r.CacheWrite1hTokens|r.ReasoningTokens != 0 {
			t.Fatalf("record %d carries tokens this source does not publish: %+v", i, r)
		}
	}
}

// TestOnlyModelTurnsBecomeRecords: a person's prompt and a platform error message are entries in
// the same file and are not model turns. Emitting them would report a conversation the model
// never answered as AI usage -- 252 of the 500 captured conversations are exactly one prompt and
// nothing else.
func TestOnlyModelTurnsBecomeRecords(t *testing.T) {
	// session.jsonl holds 8 entries: one prompt and 7 MODEL lines, 6 of which are three
	// split responses -- a GENERIC and a PLANNER_RESPONSE stamped with the same second.
	if got := len(parseFixture(t, "session", "conv-1")); got != 4 {
		t.Errorf("session records = %d, want the 4 model turns its 7 MODEL lines are written as", got)
	}
	// interrupted.jsonl holds a prompt, two model turns, a SYSTEM error message between them,
	// and a closing model turn. No two of its MODEL lines share a second.
	if got := len(parseFixture(t, "interrupted", "conv-2")); got != 3 {
		t.Errorf("interrupted records = %d, want 3 model turns", got)
	}
}

// TestEntriesSharingASecondAreOneTurn holds both directions of the fold, including the case it
// is knowingly wrong about. The vendor writes no response id, so one created_at is the only
// evidence that two entries are one response -- which means a genuine two-response second is
// indistinguishable from a split one and folds too. That is stated here rather than discovered
// later: on the 500-conversation capture the fold moves 653 MODEL lines to 645 turns, all 8 of
// them the GENERIC + PLANNER_RESPONSE split this exists for.
func TestEntriesSharingASecondAreOneTurn(t *testing.T) {
	const (
		content = `{"step_index":%d,"source":"MODEL","created_at":%q}`
		acting  = `{"step_index":%d,"source":"MODEL","created_at":%q,"tool_calls":[{"name":"write_to_file"},{"name":"run_command"}]}`
	)
	tests := []struct {
		name      string
		lines     []string
		wantRecs  int
		wantKeys  []string
		wantCalls []int64
	}{
		{
			name: "a split response is one turn carrying both halves' calls",
			lines: []string{
				fmt.Sprintf(content, 2, "2026-09-01T12:00:00Z"),
				fmt.Sprintf(acting, 3, "2026-09-01T12:00:00Z"),
			},
			wantRecs: 1, wantKeys: []string{"conv-1:2"}, wantCalls: []int64{2},
		},
		{
			name: "two responses a second apart stay two turns",
			lines: []string{
				fmt.Sprintf(acting, 1, "2026-09-01T12:00:00Z"),
				fmt.Sprintf(acting, 2, "2026-09-01T12:00:01Z"),
			},
			wantRecs: 2, wantKeys: []string{"conv-1:1", "conv-1:2"}, wantCalls: []int64{2, 2},
		},
		{
			// The stated cost of grouping on a timestamp: two responses that each did work,
			// inside one second, are reported as one turn that did all of it. The tool calls
			// survive; the turn count is one short.
			name: "two genuine responses inside one second fold, and the calls are kept",
			lines: []string{
				fmt.Sprintf(acting, 1, "2026-09-01T12:00:00Z"),
				fmt.Sprintf(acting, 2, "2026-09-01T12:00:00Z"),
			},
			wantRecs: 1, wantKeys: []string{"conv-1:1"}, wantCalls: []int64{4},
		},
		{
			name: "a second returning after another opens a new turn",
			lines: []string{
				fmt.Sprintf(acting, 1, "2026-09-01T12:00:00Z"),
				fmt.Sprintf(acting, 2, "2026-09-01T12:00:01Z"),
				fmt.Sprintf(acting, 3, "2026-09-01T12:00:00Z"),
			},
			wantRecs: 3, wantKeys: []string{"conv-1:1", "conv-1:2", "conv-1:3"}, wantCalls: []int64{2, 2, 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recs, skipped, err := ParseTranscript(strings.NewReader(strings.Join(tt.lines, "\n")+"\n"), "conv-1")
			if err != nil {
				t.Fatal(err)
			}
			if skipped != 0 {
				t.Fatalf("skipped = %d, want a folded entry counted into its turn rather than dropped", skipped)
			}
			if len(recs) != tt.wantRecs {
				t.Fatalf("recs = %d, want %d: %+v", len(recs), tt.wantRecs, recs)
			}
			for i := range recs {
				if recs[i].DedupeKey != tt.wantKeys[i] {
					t.Errorf("record %d key = %q, want %q", i, recs[i].DedupeKey, tt.wantKeys[i])
				}
				if recs[i].ToolCalls != tt.wantCalls[i] {
					t.Errorf("record %d ToolCalls = %d, want %d", i, recs[i].ToolCalls, tt.wantCalls[i])
				}
			}
		})
	}
}

// TestToolCallsAreCountedByPurpose: the vendor names its tool calls, so the class split is read
// from the name and the name is then dropped. finish is not in the shared allowlist and lands in
// Other by construction rather than being guessed at.
func TestToolCallsAreCountedByPurpose(t *testing.T) {
	var writes, commands, other, edits, calls int64
	for _, fixture := range []string{"session", "interrupted"} {
		for _, r := range parseFixture(t, fixture, "conv-1") {
			writes += r.ToolWrites
			commands += r.ToolCommands
			other += r.ToolOther
			edits += r.Edits
			calls += r.ToolCalls
		}
	}
	if writes != 3 || commands != 1 || other != 1 {
		t.Errorf("writes=%d commands=%d other=%d, want 3 write_to_file, 1 run_command, 1 finish",
			writes, commands, other)
	}
	if edits != writes {
		t.Errorf("Edits = %d, want the %d write-class calls it is defined as", edits, writes)
	}
	if calls != writes+commands+other {
		t.Errorf("ToolCalls = %d, want the %d classified calls to sum to it", calls, writes+commands+other)
	}
}

// TestRecordsAreTurnGranularity: each entry is one model response with its own timestamp, so the
// per-turn signals may read them. A session total would have to be labelled "session".
func TestRecordsAreTurnGranularity(t *testing.T) {
	recs := parseFixture(t, "session", "conv-1")
	for i := range recs {
		if recs[i].Granularity != "turn" {
			t.Fatalf("Granularity = %q, want turn", recs[i].Granularity)
		}
	}
	if !recs[0].Timestamp.Before(recs[len(recs)-1].Timestamp) {
		t.Error("the fixture's turns must span time for the active-minutes signal to be readable")
	}
}

// TestNoProjectIsGuessed: the transcript carries no working directory, and the only path in the
// file is a tool argument naming the vendor's own scratch directory. Leaving Cwd and Project
// empty puts agy sessions in the unattributed bucket, which is where a source that cannot say
// belongs.
func TestNoProjectIsGuessed(t *testing.T) {
	for _, r := range parseFixture(t, "session", "conv-1") {
		if r.Cwd != "" || r.Project != "" || r.Subpath != "" {
			t.Fatalf("record names a project the transcript does not: %+v", r)
		}
	}
}

// TestDedupeKeysAreStableAndDistinct is what makes a re-import idempotent.
func TestDedupeKeysAreStableAndDistinct(t *testing.T) {
	first := parseFixture(t, "session", "conv-1")
	second := parseFixture(t, "session", "conv-1")
	seen := map[string]bool{}
	for i := range first {
		if first[i].DedupeKey != second[i].DedupeKey {
			t.Fatalf("re-parsing an unchanged file changed key %d", i)
		}
		if seen[first[i].DedupeKey] {
			t.Fatalf("duplicate dedupe key %q", first[i].DedupeKey)
		}
		if !strings.HasPrefix(first[i].DedupeKey, "conv-1:") {
			t.Fatalf("dedupe key %q must carry the conversation it belongs to", first[i].DedupeKey)
		}
		seen[first[i].DedupeKey] = true
	}
}

// A conversation with no id would key every one of its turns as ":<step>", collapsing every
// unidentified conversation onto one stored row. Yielding nothing is the honest outcome.
func TestConversationWithoutAnIDYieldsNothing(t *testing.T) {
	recs, skipped, err := ParseTranscript(strings.NewReader(
		`{"step_index":1,"source":"MODEL","created_at":"2026-09-01T12:00:00Z"}`+"\n",
	), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 || skipped != 0 {
		t.Fatalf("recs=%d skipped=%d, want nothing from an unidentified conversation", len(recs), skipped)
	}
}

// A record stamped with the zero time lands in every window that reaches back far enough and in
// none that does not, so an unreadable clock drops the turn and counts it.
func TestUndatableTurnIsSkippedAndCounted(t *testing.T) {
	recs, skipped, err := ParseTranscript(strings.NewReader(
		`{"step_index":1,"source":"MODEL","created_at":"whenever"}`+"\n",
	), "conv-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 || skipped != 1 {
		t.Fatalf("recs=%d skipped=%d, want the undatable turn dropped and counted", len(recs), skipped)
	}
}

func TestCorruptLinesAreSkippedAndCounted(t *testing.T) {
	recs, skipped, err := ParseTranscript(strings.NewReader(
		"{not json\n"+`{"step_index":1,"source":"MODEL","created_at":"2026-09-01T12:00:00Z"}`+"\n",
	), "conv-1")
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want 1", skipped)
	}
	if len(recs) != 1 {
		t.Fatalf("recs = %d, want the line after the corrupt one still read", len(recs))
	}
}

// TestRepeatedStepIndexIsSkippedAndCounted: the dedupe key is "<conversation>:<step_index>", so
// a second entry claiming a position already emitted would be dropped by the store's ON CONFLICT
// and counted nowhere. A fuzz seed found it; the corpus keeps the case (0 repeats across the 500
// captured conversations, so a repeat is damage rather than a shape).
func TestRepeatedStepIndexIsSkippedAndCounted(t *testing.T) {
	const turn = `{"step_index":3,"source":"MODEL","created_at":"2026-09-01T12:46:11Z"}`
	recs, skipped, err := ParseTranscript(strings.NewReader(turn+"\n"+turn+"\n"), "conv-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("recs = %d, want the repeat dropped rather than stored under a key it shares", len(recs))
	}
	if skipped != 1 {
		t.Fatalf("skipped = %d, want the repeat counted rather than lost silently", skipped)
	}
}
