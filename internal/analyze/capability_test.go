package analyze

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// The three source shapes a session figure has to tell apart: one recording everything, one
// recording turns but no edits or compaction, and one that totals a whole session.
const (
	deepTool         = "claude-code"
	costOnlyTool     = "cline"
	sessionTotalTool = "copilot-cli"
)

func sessionsFrom(tool string, n int) []store.SessionRow {
	out := make([]store.SessionRow, 0, n)
	for i := range n {
		out = append(out, store.SessionRow{
			SessionID: tool + string(rune('a'+i)), Project: "p", Tool: tool,
			FirstTs:      validatorsTestNow.Add(-time.Duration(i) * 3 * time.Hour),
			LastTs:       validatorsTestNow,
			Turns:        4,
			OutputTokens: 1000,
		})
	}
	return out
}

// A source that records no edits leaves every session at zero, and bucketing those as
// "conversational" states what kind of work someone did from a field their tool never kept.
func TestSessionTaxonomyRefusesToCallUnrecordedEditsConversational(t *testing.T) {
	in := BuildInput(nil, sessionsFrom(costOnlyTool, 8), testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := Evaluate(mustGet(t, taxonomyName), &in)

	if got.Read.Key != "neutral" {
		t.Errorf("Read = %q, want a withheld verdict when no source records edits", got.Read.Key)
	}
	if strings.Contains(figureValues(got.Figures), "100%") {
		t.Errorf("Figures = %q, want no share computed from sessions nothing recorded", figureValues(got.Figures))
	}
	if !strings.Contains(got.Takeaway, "records file edits") {
		t.Errorf("Takeaway = %q, want it to name the missing capture", got.Takeaway)
	}
	if got.Confidence.Label != ConfidenceInsufficient {
		t.Errorf("Confidence = %q, want %q", got.Confidence.Label, ConfidenceInsufficient)
	}
}

// The mix is real for the sessions that carry edit capture; what must not happen is the
// other sessions joining the conversational bucket and halving the edit-led share.
func TestSessionTaxonomyMixesOnlyTheSessionsThatCanAnswer(t *testing.T) {
	sessions := sessionsFrom(deepTool, 6)
	for i := range sessions {
		sessions[i].Edits = int64(i) // 1 conversational, 5 editing
	}
	sessions = append(sessions, sessionsFrom(costOnlyTool, 6)...)
	in := BuildInput(nil, sessions, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := Evaluate(mustGet(t, taxonomyName), &in)

	if !strings.Contains(figureValues(got.Figures), "17%") { // 1 of 6, not 7 of 12
		t.Errorf("Figures = %q, want the mix over the six sessions with edit capture", figureValues(got.Figures))
	}
	if got.Confidence.Signal == nil || *got.Confidence.Signal != 0.5 {
		t.Errorf("signal coverage = %v, want half the window declared", got.Confidence.Signal)
	}
}

// Compaction is the verdict, so a window whose sources never mark one has no context health
// to report -- silence is not a session that comfortably fit.
func TestContextWithholdsHealthWhenNoSourceMarksCompaction(t *testing.T) {
	in := BuildInput(nil, sessionsFrom(costOnlyTool, 8), testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := Evaluate(mustGet(t, contextName), &in)

	if got.Read.Key != "neutral" {
		t.Errorf("Read = %q, want no health verdict without compaction capture", got.Read.Key)
	}
	if strings.Contains(figureValues(got.Figures), "0%") {
		t.Errorf("Figures = %q, want no rate printed where nothing was recorded", figureValues(got.Figures))
	}
	if !strings.Contains(got.Takeaway, "marks a context compaction") {
		t.Errorf("Takeaway = %q, want it to name the missing marker", got.Takeaway)
	}
}

// A whole-session record has no second turn, so turn depth, peak context and focused time
// are absent from it rather than zero.
func TestContextExcludesSessionTotalsFromPerTurnFigures(t *testing.T) {
	in := BuildInput(nil, sessionsFrom(sessionTotalTool, 5), testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := Evaluate(mustGet(t, contextName), &in)

	for _, label := range []string{"median turns", "peak context", "active work"} {
		if figureFor(got.Figures, label).Value != "—" {
			t.Errorf("%s = %q, want it withheld for a source that totals a session", label, figureFor(got.Figures, label).Value)
		}
	}
}

// When work starts is readable from any source; how long it ran is not. The timing half must
// survive, and the length half must abstain rather than report an instant.
func TestRhythmKeepsTimingWhenLengthIsUnrecorded(t *testing.T) {
	in := BuildInput(nil, sessionsFrom(sessionTotalTool, 8), testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	got := Evaluate(mustGet(t, rhythmName), &in)

	if figureFor(got.Figures, "off-hours").Value == "" {
		t.Error("off-hours must still be reported: a start timestamp is real at any grain")
	}
	if v := figureFor(got.Figures, "marathons").Value; v != "—" {
		t.Errorf("marathons = %q, want it withheld where focused minutes are not recorded", v)
	}
	if got.Read.Key != "neutral" {
		t.Errorf("Read = %q, want the verdict withheld while half of it cannot be measured", got.Read.Key)
	}
}

func figureFor(figures []Figure, label string) Figure {
	for _, f := range figures {
		if f.Label == label {
			return f
		}
	}
	return Figure{}
}

// The invariant ADR 0011 exists for, asserted generically because a twentieth validator is
// exactly what would break it: a source that answers none of the per-session signals must
// contribute no value to any figure over them, so the same window with those fields filled
// in and with them at zero has to produce identical results.
func TestNoValidatorReadsAFieldItsSourceCannotRecord(t *testing.T) {
	silent := sessionsFrom(sessionTotalTool, 6)
	filled := sessionsFrom(sessionTotalTool, 6)
	for i := range filled {
		filled[i].Turns = 40
		filled[i].ActiveMinutes = 300
		filled[i].PeakContextTokens = 900_000
		filled[i].Edits = 25
		filled[i].Compactions = 4
	}
	usage := []store.UsageRow{{
		Day: validatorsTestNow.Format("2006-01-02"), Tool: sessionTotalTool,
		Model: "claude-sonnet-4-5", Project: "p", Granularity: "session",
		In: 1000, Out: 1000, LinesAdded: 300,
	}}

	quiet := BuildInput(usage, silent, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	loud := BuildInput(usage, filled, testPrices(), validatorsTestNow, 7*24*time.Hour, Delegation{})
	for _, v := range Validators() {
		got, want := render(t, v, &loud), render(t, v, &quiet)
		if got != want {
			t.Errorf("%s reads a field %s does not record:\n with values:\n%s\n with zeros:\n%s",
				v.Name(), sessionTotalTool, got, want)
		}
	}
}

func render(t *testing.T, v Validator, in *Input) string {
	t.Helper()
	var buf bytes.Buffer
	if err := RenderResultText(&buf, Evaluate(v, in)); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
