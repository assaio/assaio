package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// The split is taken over calls a source names, which can be a fraction of the window's
// calls; a caveat already says so, and the envelope has to agree rather than reporting the
// verdict as resting on the whole window.
func TestExploreProduceCarriesItsClassifiedShare(t *testing.T) {
	rows := []store.UsageRow{
		{
			Day: "2026-07-08", Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", Project: "web",
			In: 1000, Out: 500, ToolReads: 40, ToolWrites: 40, ToolCalls: 80,
		},
		{
			Day: "2026-07-09", Tool: "codex", Model: "claude-sonnet-4-5", Granularity: "turn", Project: "web",
			In: 1000, Out: 500, ToolCalls: 500,
		},
		{
			Day: "2026-07-10", Tool: "codex", Model: "claude-sonnet-4-5", Granularity: "turn", Project: "web",
			In: 1000, Out: 500, ToolCalls: 500,
		},
	}
	in := BuildInput(rows, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := Evaluate(mustGet(t, exploreName), &in)

	if got.Confidence.signalShare() >= 1 {
		t.Fatalf("Signal = %v, want the classified share of the window's calls", got.Confidence.signalShare())
	}
	if got.Confidence.Label == ConfidenceHigh {
		t.Errorf("Label = %q for a split read off %.0f%% of the calls, want it held down",
			got.Confidence.Label, got.Confidence.signalShare()*100)
	}
}

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

// TestExploreProduceReportsTheSplitWithoutGradingIt asserts the split is described and not
// judged (B177): a window that writes freely and one that almost never writes get the same
// read, and the reads-per-write ratio is what tells them apart.
func TestExploreProduceReportsTheSplitWithoutGradingIt(t *testing.T) {
	writing := mustGet(t, exploreName).Analyze(toolMixInput(60, 20, 30, 20, 10))
	if writing.Read != reportedRead {
		t.Fatalf("Read = %+v, want the descriptive read when writes accompany heavy reading", writing.Read)
	}
	if !strings.Contains(figureValues(writing.Figures), "4.0×") {
		t.Fatalf("Figures = %q, want 4.0× reads+searches per write", figureValues(writing.Figures))
	}

	searching := mustGet(t, exploreName).Analyze(toolMixInput(300, 200, 50, 1, 10))
	if searching.Read != reportedRead {
		t.Fatalf("Read = %+v, want the same descriptive read when almost nothing is written", searching.Read)
	}
	if !strings.Contains(searching.Takeaway, "reads per write") {
		t.Fatalf("Takeaway = %q, want the ratio stated rather than judged", searching.Takeaway)
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

// TestExploreProduceCoverageCaveatWhenMixed asserts the split discloses that it covers only
// part of the calls. What narrows it is history: usage from a naming source, ingested before
// the purpose split was captured, carries calls with no purpose on them.
func TestExploreProduceCoverageCaveatWhenMixed(t *testing.T) {
	rows := []store.UsageRow{
		{
			Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web",
			ToolReads: 40, ToolWrites: 20, ToolCalls: 60,
		},
		{Day: "2026-07-09", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", ToolCalls: 40},
	}
	in := BuildInput(rows, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := mustGet(t, exploreName).Analyze(in)

	joined := strings.Join(got.Caveats, " ")
	if !strings.Contains(joined, "60%") {
		t.Fatalf("Caveats = %q, want the 60%% coverage disclosure", got.Caveats)
	}
}

// A source that names no tool call cannot lower the coverage figure: it records no call at
// all, so counting its work as "unclassified" would report a gap in a capture that was never
// attempted. The caveat's own wording already said this; the arithmetic did not.
func TestExploreProduceIgnoresSourcesThatNameNoToolCalls(t *testing.T) {
	rows := []store.UsageRow{
		{
			Day: "2026-07-10", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web",
			ToolReads: 40, ToolWrites: 20, ToolCalls: 60,
		},
		{Day: "2026-07-10", Tool: "cline", Model: "claude-sonnet-4-5", Project: "web", ToolCalls: 40},
	}
	in := BuildInput(rows, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := Evaluate(mustGet(t, exploreName), &in)

	if got.Confidence.Signal == nil || *got.Confidence.Signal != 1 {
		t.Fatalf("signal coverage = %v, want the full reach: every call that could be named was",
			got.Confidence.Signal)
	}
	for _, c := range got.Caveats {
		if strings.Contains(c, "Prov.:") {
			t.Fatalf("Caveats = %q, want no coverage gap disclosed for a source that records no calls", got.Caveats)
		}
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
