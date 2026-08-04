package analyze

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// stub is a validator that returns a fixed Result, so a test exercises the envelope rather
// than any real metric's logic.
type stub struct{ samples int }

func (stub) Name() string     { return "stub" }
func (stub) Title() string    { return "Stub" }
func (stub) Describe() string { return "test double" }

func (s stub) Analyze(Input) Result {
	return Result{Name: "stub", Confidence: Confidence{Samples: s.samples, Unit: "sessions"}}
}

// coveringStub declares how much of the window its own subject covers, which is the one
// axis the framework cannot compute for a validator.
type coveringStub struct{ share float64 }

func (coveringStub) Name() string     { return "covering-stub" }
func (coveringStub) Title() string    { return "Covering Stub" }
func (coveringStub) Describe() string { return "test double" }

func (s coveringStub) Analyze(Input) Result {
	r := Result{Name: "covering-stub"}
	r.restsOn(10, "sessions")
	r.covering(s.share)
	return r
}

// A figure can rest on a sliver of a perfectly covered window: reasoning-share read 20% off
// under 1% of the output and still carried "high", while `signals coverage` called the very
// same signal partially supported. Only the validator knows how much of the window its own
// subject describes, so it is the one axis it has to declare.
func TestSignalCoverageLowersTheLabelOnAnOtherwiseCoveredWindow(t *testing.T) {
	in := window(t, []store.UsageRow{
		{Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", In: 1000},
	})

	if got := Evaluate(stub{samples: 10}, &in); got.Confidence.Label != ConfidenceHigh {
		t.Fatalf("a validator declaring no signal coverage = %q, want %q unchanged",
			got.Confidence.Label, ConfidenceHigh)
	}

	got := Evaluate(coveringStub{share: 0.008}, &in)
	if got.Confidence.Label != ConfidenceLow {
		t.Errorf("Label = %q, want %q for a figure covering under 1%% of the window",
			got.Confidence.Label, ConfidenceLow)
	}
	if summary := ConfidenceSummary(&got.Confidence); !strings.Contains(summary, "signal coverage") {
		t.Errorf("summary = %q, want it to name the axis holding the label down", summary)
	}
}

// Covering none of the window is not a thin verdict, it is no verdict: nothing in this
// window could answer the question, which is exactly what insufficient means.
func TestSignalCoverageOfNoneReadsInsufficient(t *testing.T) {
	in := window(t, []store.UsageRow{
		{Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", In: 1000},
	})

	got := Evaluate(coveringStub{share: 0}, &in)
	if got.Confidence.Label != ConfidenceInsufficient {
		t.Fatalf("Label = %q, want %q when no source in the window can answer at all",
			got.Confidence.Label, ConfidenceInsufficient)
	}
}

// A coverage share that rounds to zero is not zero. "signal coverage 0%" reads as an absent
// figure where the truth is a real one describing a sliver -- the exact rounding edge the
// share formatter already refuses everywhere else a percentage is printed.
func TestConfidenceSummaryNeverRoundsARealShareDownToZero(t *testing.T) {
	c := Confidence{Label: ConfidenceLow, Samples: 5, Unit: "sessions", Activity: 1, Priced: 1, Turn: 1}
	c.Covering(0.001)
	got := ConfidenceSummary(&c)
	if strings.Contains(got, "0%") {
		t.Fatalf("summary = %q, want a real share not rounded to an absent-looking 0%%", got)
	}
	if !strings.Contains(got, "<1%") {
		t.Errorf("summary = %q, want it to read <1%%", got)
	}
}

// window builds an Input whose coverage axes are controllable: claude-code carries
// activity, gemini-cli does not, and an unpriced model lowers priced coverage.
func window(t *testing.T, rows []store.UsageRow) Input {
	t.Helper()
	return BuildInput(rows, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
}

func TestConfidenceLabelFollowsTheWeakestAxis(t *testing.T) {
	tests := []struct {
		name string
		rows []store.UsageRow
		want string
	}{
		{
			name: "everything covered",
			rows: []store.UsageRow{
				{Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", In: 1000},
			},
			want: ConfidenceHigh,
		},
		{
			name: "a third of the tokens have no activity signals",
			rows: []store.UsageRow{
				{Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", In: 700},
				{Tool: "gemini-cli", Model: "claude-sonnet-4-5", Granularity: "turn", In: 300},
			},
			want: ConfidenceMedium,
		},
		{
			name: "most of the window is unpriced",
			rows: []store.UsageRow{
				{Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", In: 200},
				{Tool: "claude-code", Model: "unknown-model", Granularity: "turn", In: 800},
			},
			want: ConfidenceLow,
		},
		{
			name: "most of the window is session aggregates",
			rows: []store.UsageRow{
				{Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", In: 200},
				{Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "session", In: 800},
			},
			want: ConfidenceLow,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := window(t, tt.rows)
			got := Evaluate(stub{samples: 10}, &in)
			if got.Confidence.Label != tt.want {
				t.Fatalf("Label = %q, want %q (envelope %+v)", got.Confidence.Label, tt.want, got.Confidence)
			}
		})
	}
}

// A source that records changed lines and nothing else answers no edit, tool-call or rework
// signal, so it cannot make an activity-covered window. The three-axis Activity bit reads
// true for it, which is why the envelope asks the per-signal question instead (ADR 0008).
func TestPartialActivityDoesNotCountAsActivityCoverage(t *testing.T) {
	tests := []struct {
		name string
		rows []store.UsageRow
		want float64
	}{
		{
			name: "a lines-only source covers no activity figure",
			rows: []store.UsageRow{
				{Tool: "copilot-cli", Model: "claude-sonnet-4-5", Granularity: "session", In: 1000, LinesAdded: 50},
			},
			want: 0,
		},
		{
			name: "it lowers the share rather than passing for full capture",
			rows: []store.UsageRow{
				{Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", In: 700},
				{Tool: "copilot-cli", Model: "claude-sonnet-4-5", Granularity: "session", In: 300},
			},
			want: 0.7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := window(t, tt.rows)
			if got := Evaluate(stub{samples: 10}, &in).Confidence.Activity; got != tt.want {
				t.Fatalf("Activity = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestConfidenceIsInsufficientWhenNothingWasCounted separates "the data says no" from
// "there was nothing to say it about" -- the distinction a caveat cannot carry.
func TestConfidenceIsInsufficientWhenNothingWasCounted(t *testing.T) {
	in := window(t, []store.UsageRow{
		{Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", In: 1000},
	})
	got := Evaluate(stub{samples: 0}, &in)
	if got.Confidence.Label != ConfidenceInsufficient {
		t.Fatalf("Label = %q, want %q", got.Confidence.Label, ConfidenceInsufficient)
	}
}

// TestFullCoverageCannotRescueTooFewObservations is the cap: no coverage figure makes a
// verdict computed from two observations a confident one.
func TestFullCoverageCannotRescueTooFewObservations(t *testing.T) {
	in := window(t, []store.UsageRow{
		{Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", In: 1000},
	})
	got := Evaluate(stub{samples: 2}, &in)
	if got.Confidence.Label != ConfidenceLow {
		t.Fatalf("Label = %q, want %q on 2 samples despite full coverage", got.Confidence.Label, ConfidenceLow)
	}
}

// TestComponentsStayInspectable is the deliberate non-goal: the label may summarize, but a
// reader must still be able to see which axis is the thin one.
func TestComponentsStayInspectable(t *testing.T) {
	in := window(t, []store.UsageRow{
		{Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", In: 700},
		{Tool: "gemini-cli", Model: "unknown-model", Granularity: "session", In: 300},
	})
	got := Evaluate(stub{samples: 10}, &in).Confidence
	if got.Activity != 0.7 {
		t.Errorf("Activity = %v, want 0.7", got.Activity)
	}
	if got.Priced != 0.7 {
		t.Errorf("Priced = %v, want 0.7", got.Priced)
	}
	if got.Turn != 0.7 {
		t.Errorf("Turn = %v, want 0.7", got.Turn)
	}
	if got.Unit != "sessions" {
		t.Errorf("Unit = %q, want the validator's own unit preserved", got.Unit)
	}
}

func TestEvaluateCarriesProvenanceFromTheInput(t *testing.T) {
	in := window(t, []store.UsageRow{
		{Tool: "claude-code", Model: "claude-sonnet-4-5", Granularity: "turn", In: 1000},
	})
	in.Ingested = time.Date(2026, 7, 31, 9, 0, 0, 0, time.UTC)
	in.ParsedBy = "v0.5.0"
	got := Evaluate(stub{samples: 10}, &in).Confidence
	if !got.Ingested.Equal(in.Ingested) || got.ParsedBy != "v0.5.0" {
		t.Fatalf("provenance = %+v, want it carried through", got)
	}
}

// TestEveryValidatorSaysWhatItCounted is the forcing function for the envelope's per-metric
// half: coverage is uniform across the window, but sample size is not, and only the
// validator knows whether its verdict rested on 40 sessions or 2 projects.
func TestEveryValidatorSaysWhatItCounted(t *testing.T) {
	in := everySignalInput()
	for _, v := range Validators() {
		if strings.HasSuffix(v.Name(), "-test-fake") {
			continue
		}
		got := Evaluate(v, &in)
		if got.Confidence.Unit == "" {
			t.Errorf("%s: Confidence.Unit is empty -- it must name what it counted", v.Name())
		}
		if got.Confidence.Samples == 0 {
			t.Errorf("%s: counted nothing on a favorable window, so every verdict reads insufficient", v.Name())
		}
	}
}

// TestValidatorsOnAnEmptyWindowReadInsufficient separates "no answer" from "not enough data
// to ask" for every metric at once.
func TestValidatorsOnAnEmptyWindowReadInsufficient(t *testing.T) {
	in := BuildInput(nil, nil, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	for _, v := range Validators() {
		if strings.HasSuffix(v.Name(), "-test-fake") {
			continue
		}
		if got := Evaluate(v, &in); got.Confidence.Label != ConfidenceInsufficient {
			t.Errorf("%s on an empty window = %q, want %q", v.Name(), got.Confidence.Label, ConfidenceInsufficient)
		}
	}
}

// everySignalInput is a window that carries every signal any validator reads: classified
// tool calls, failures, per-model turn counts and attribution labels. The shared favorable
// fixture deliberately does not, so the four metrics built on those signals would report
// "counted nothing" there -- an honest answer to that window, and a useless one for
// checking that each validator declares its own subject.
func everySignalInput() Input {
	in := favorableInput()
	for i := range in.Usage {
		u := &in.Usage[i]
		u.ToolReads, u.ToolSearches, u.ToolCommands, u.ToolWrites = 4, 3, 2, 3
		u.ToolErrors = 1
	}
	in.TurnSizing = []store.ModelTurns{{Model: "claude-opus-4-5", Turns: 40, SmallTurns: 4}}
	in.Skills = []store.AttributionRow{{Name: "code-review", Tokens: 1200, Lines: 40, Records: 6, Sessions: 2}}
	in.Agents = []store.AttributionRow{{Name: "general-purpose", Tokens: 800, Lines: 20, Records: 4, Sessions: 1}}
	built := BuildInput(in.Usage, in.Sessions, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	return withSignals(&built, in.TurnSizing, in.Skills, in.Agents)
}

// withSignals copies the fields BuildInput does not assemble, which the CLI sets after it.
func withSignals(in *Input, turns []store.ModelTurns, skills, agents []store.AttributionRow) Input {
	in.TurnSizing, in.Skills, in.Agents = turns, skills, agents
	return *in
}
