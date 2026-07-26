package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// toolMixInput seeds one usage row whose tool-call taxonomy is exactly the counts given.
// ToolCalls is set to their sum, the invariant the parsers guarantee.
func toolMixInput(reads, searches, commands, writes, other int64) Input {
	row := store.UsageRow{
		Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web",
		In: 1000, Out: 500, LinesAdded: 40,
		ToolReads: reads, ToolSearches: searches, ToolCommands: commands,
		ToolWrites: writes, ToolOther: other,
		ToolCalls: reads + searches + commands + writes + other,
	}
	return BuildInput([]store.UsageRow{row}, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
}

// TestExploreProduceBalancedWhenWritesFollowReads asserts a window that reads a lot but
// still writes reads as balanced -- exploring is how an agent earns the right to write.
func TestExploreProduceBalancedWhenWritesFollowReads(t *testing.T) {
	got := mustGet(t, exploreName).Analyze(toolMixInput(60, 20, 30, 20, 10))

	if got.Read.Label != "BALANCED" {
		t.Fatalf("Read = %q, want BALANCED when writes accompany heavy reading", got.Read.Label)
	}
	if !strings.Contains(figureValues(got.Figures), "4.0×") {
		t.Fatalf("Figures = %q, want 4.0× reads+searches per write", figureValues(got.Figures))
	}
}

// TestExploreProduceFlagsEndlessSearching asserts a window that explores heavily and almost
// never writes is the case this metric exists to surface.
func TestExploreProduceFlagsEndlessSearching(t *testing.T) {
	got := mustGet(t, exploreName).Analyze(toolMixInput(300, 200, 50, 1, 10))

	if got.Read.Label != "WATCH" {
		t.Fatalf("Read = %q, want WATCH when almost nothing is written", got.Read.Label)
	}
	if !strings.Contains(got.Takeaway, "stuck exploring") {
		t.Fatalf("Takeaway = %q, want the stuck-exploring message", got.Takeaway)
	}
}

// TestExploreProduceThinSampleIsUndefined asserts a handful of calls yields the neutral
// read rather than a verdict from almost no evidence.
func TestExploreProduceThinSampleIsUndefined(t *testing.T) {
	got := mustGet(t, exploreName).Analyze(toolMixInput(3, 1, 1, 0, 0))

	if got.Read != noDataRead {
		t.Fatalf("Read = %+v, want the neutral read below the %d-call floor", got.Read, exploreMinCalls)
	}
}

// TestExploreProduceUnclassifiedUsageIsNotZero asserts a window whose tools never name their
// tool calls says so, rather than rendering a 0% produce share that looks like a finding.
func TestExploreProduceUnclassifiedUsageIsNotZero(t *testing.T) {
	row := store.UsageRow{
		Day: "2026-07-10", Tool: "gemini-cli", Model: "claude-sonnet-4-5", Project: "web",
		In: 5000, Out: 2000, ToolCalls: 0,
	}
	in := BuildInput([]store.UsageRow{row}, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, exploreName).Analyze(in)

	if got.Read != noDataRead {
		t.Fatalf("Read = %+v, want the neutral read when nothing is classified", got.Read)
	}
	if !strings.Contains(got.Takeaway, "records what its tool calls were for") {
		t.Fatalf("Takeaway = %q, want the no-attribution explanation", got.Takeaway)
	}
	if len(got.Caveats) == 0 {
		t.Fatal("a window with no classified calls must carry the coverage caveat")
	}
}

// TestExploreProduceCoverageCaveatWhenMixed asserts a window mixing a naming tool with a
// non-naming one discloses that the split covers only part of the calls.
func TestExploreProduceCoverageCaveatWhenMixed(t *testing.T) {
	rows := []store.UsageRow{
		{
			Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web",
			ToolReads: 40, ToolWrites: 20, ToolCalls: 60,
		},
		{Day: "2026-07-10", Tool: "cline", Model: "claude-sonnet-4-5", Project: "web", ToolCalls: 40},
	}
	in := BuildInput(rows, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, exploreName).Analyze(in)

	joined := strings.Join(got.Caveats, " ")
	if !strings.Contains(joined, "60%") {
		t.Fatalf("Caveats = %q, want the 60%% coverage disclosure", got.Caveats)
	}
}

// TestExploreProduceRatioDashWithoutWrites asserts the reads-per-write ratio renders "—"
// rather than dividing by zero writes.
func TestExploreProduceRatioDashWithoutWrites(t *testing.T) {
	got := mustGet(t, exploreName).Analyze(toolMixInput(60, 0, 0, 0, 0))

	if !strings.Contains(figureValues(got.Figures), "—") {
		t.Fatalf("Figures = %q, want a dash for reads-per-write with no writes", figureValues(got.Figures))
	}
}

// TestExploreProduceEmptyInputSafe asserts the zero-value Input renders the no-data block.
func TestExploreProduceEmptyInputSafe(t *testing.T) {
	got := mustGet(t, exploreName).Analyze(Input{})
	if got.Read != noDataRead || got.Takeaway != "No usage in this window." {
		t.Fatalf("empty Input = %+v, want the no-data read and takeaway", got)
	}
}
