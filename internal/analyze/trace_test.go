package analyze

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/trace"
)

// traceReaders is every registered detector, and the assertion that there is at least one: a build
// with none would carry the whole sequence read for nothing, and every gate below would pass by
// having nothing to check.
func traceReaders(t *testing.T) []Validator {
	t.Helper()
	var out []Validator
	for _, v := range Validators() {
		if _, ok := v.(TraceReader); ok {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		t.Fatal("no validator reads the step sequences, so the trace read is dead weight and these gates check nothing")
	}
	return out
}

// The refusal is the contract, not a nicety: a detector fires on a pattern, and a hard bug looks
// exactly like a loop. A detector that publishes a finding without naming what it cannot tell that
// finding apart from is the one failure mode the whole design is against.
func TestEveryTraceReaderDeclaresWhatItCannotDistinguish(t *testing.T) {
	in := Input{Trace: favorableTrace(t)}
	for _, v := range traceReaders(t) {
		got := Evaluate(v, &in)
		if !hasCaveatContaining(got.Caveats, cannotDistinguish) {
			t.Errorf("%s publishes a finding without saying what its pattern cannot be told apart from: %v",
				v.Name(), got.Caveats)
		}
	}
}

// The scope is the denominator, so every detector has to state which population it read and what
// that left out -- 89% of the audited store's sequences are excluded from the interactive scope,
// and a figure that does not say so reads as a fact about everything.
func TestEveryTraceReaderStatesItsPopulation(t *testing.T) {
	in := Input{Trace: favorableTrace(t)}
	for _, v := range traceReaders(t) {
		got := Evaluate(v, &in)
		if !hasCaveatContaining(got.Caveats, "Scope: ") {
			t.Errorf("%s does not state the population it read: %v", v.Name(), got.Caveats)
		}
		if got.Confidence.Unit == "" || got.Confidence.Samples == 0 {
			t.Errorf("%s counted nothing on a window of eight sessions: %+v", v.Name(), got.Confidence)
		}
	}
}

// A scope outside the vocabulary reads an empty view, which would publish "nothing in scope" over
// a full store -- a silent zero rather than an error.
func TestEveryTraceReaderDeclaresAKnownScope(t *testing.T) {
	known := map[string]bool{
		trace.Interactive: true, trace.SubAgent: true,
		trace.Programmatic: true, trace.Unstated: true,
	}
	for _, v := range traceReaders(t) {
		scope := v.(TraceReader).TraceScope()
		if !known[scope] {
			t.Errorf("%s declares scope %q, which internal/trace does not define", v.Name(), scope)
		}
	}
}

// Every detector must survive a window that holds sequences but none of its own scope, and say
// which of the two silences it hit -- a missing backfill is actionable, a window of SDK calls is
// not.
func TestEveryTraceReaderSeparatesAnEmptyStoreFromAnEmptyScope(t *testing.T) {
	for _, v := range traceReaders(t) {
		empty := Evaluate(v, &Input{})
		if empty.Confidence.Label != ConfidenceInsufficient {
			t.Errorf("%s on a store with no sequences = %q, want insufficient", v.Name(), empty.Confidence.Label)
		}
		if !strings.Contains(empty.Takeaway, "backfill") {
			t.Errorf("%s does not point at backfill when nothing is stored: %q", v.Name(), empty.Takeaway)
		}
		outOfScope := Evaluate(v, &Input{Trace: traceOf(programmaticSequence(t, "script", "a e1 c a e1"))})
		if strings.Contains(outOfScope.Takeaway, "backfill") {
			t.Errorf("%s asks for a backfill over a window that has sequences: %q", v.Name(), outOfScope.Takeaway)
		}
	}
}

// NeedsTrace is what keeps a 2.5-second read off every command that does not want it, so it has to
// answer for the real registry rather than for a hand-built list.
func TestNeedsTraceFollowsTheRegistry(t *testing.T) {
	if !NeedsTrace(Validators()) {
		t.Error("NeedsTrace says no registered validator reads sequences, but detectors are registered")
	}
	if NeedsTrace([]Validator{frictionValidator{}}) {
		t.Error("NeedsTrace claims a validator that never touches Input.Trace needs the read")
	}
}

// A window whose sessions vary widely puts the MAD floor above 1, where no rate can reach it. The
// answer has to be "this window cannot be judged", not the favourable read and a full purity gauge
// earned by a threshold the arithmetic made unreachable.
func TestEditLoopsWithholdsAVerdictWhenTheFloorLeavesTheRateDomain(t *testing.T) {
	// Eight sessions spread from 0% to ~90%: median .25, MAD large enough to push the floor past 1.
	specs := []string{
		"e1 c e2 c e3 c e4 c e5 c e6 c e7 c e8 c e9 c e10",
		"e1 c e1 c e1 c e1 c e1 c e1 c e1 c e1 c e1 c e1",
		"e1 c e2 c e3 c e4 c e5 c e1 c e2 c e3 c e4 c e5",
		"e1 c e1 c e1 c e1 c e1 c e2 c e3 c e4 c e5 c e6",
	}
	var sequences []store.Timeline
	for i := range 8 {
		sequences = append(sequences, sequence(t, "s"+strconv.Itoa(i), "p", specs[i%len(specs)]))
	}
	in := Input{Trace: traceOf(sequences...)}

	p := buildRepeats(interactiveView(&in.Trace))
	rates := make([]float64, len(p.Judged))
	for i := range p.Judged {
		rates[i] = p.Judged[i].Rate()
	}
	floor, ok := outlierFloor(rates)
	// A hard failure rather than a skip: a fixture that stopped producing an out-of-domain floor
	// would leave every assertion below true for the wrong reason, and the guard untested.
	if !ok || floor < 1 {
		t.Fatalf("this fixture no longer exercises the guard: floor = %v (ok=%v), want >= 1", floor, ok)
	}
	if p.Spread {
		t.Errorf("the window reports a verdict against a floor of %v, which no rate can reach", floor)
	}
	got := Evaluate(editLoopsValidator{}, &in)
	if got.Read != noDataRead {
		t.Errorf("Read = %+v, want no verdict when nothing could have stood out", got.Read)
	}
	if got.Purity != 0.5 {
		t.Errorf("Purity = %v, want the neutral 0.5 rather than a full gauge earned by arithmetic", got.Purity)
	}
}

// Narrowing a set must not move "now". The open-session tail is measured against the store's own
// newest step, so the same session cannot be finished on one panel and still running on another.
func TestNarrowingASetKeepsTheHorizonOfTheSetItCameFrom(t *testing.T) {
	old := sequenceEndedAgo(t, "old", "api", "a c!err", 30*24*time.Hour)
	recent := sequence(t, "recent", "web", "a c")
	full := traceOf(old, recent)

	narrowed := full.ForSessions([]store.SessionRow{{SessionID: "old"}})

	if !narrowed.Newest.Equal(full.Newest) {
		t.Errorf("narrowed Newest = %v, want the full set's %v", narrowed.Newest, full.Newest)
	}
	whole := buildAftermath(interactiveView(&full), full.Newest)
	part := buildAftermath(interactiveView(&narrowed), narrowed.Newest)
	if len(whole.Abandoned) != 1 || len(part.Abandoned) != 1 {
		t.Errorf("the same session is abandoned in one view (%d) and not the other (%d)",
			len(whole.Abandoned), len(part.Abandoned))
	}
}

// A window reaching back further than the sequences do must not report the scope share as if it
// covered the whole of it: the figure is that span's, and the confidence has to say so.
func TestCoverageFallsWhenTheWindowOutreachesTheStoredSequences(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	in := Input{
		Now:         now,
		WindowStart: now.AddDate(0, 0, -90),
		Trace:       traceOf(sequence(t, "s", "p", "a e1 c a e1")),
	}
	oldest := now.AddDate(0, 0, -30)

	if got := windowCoveredByTrace(&in, oldest); got < 0.32 || got > 0.34 {
		t.Errorf("window coverage = %v, want about a third: 30 days of sequences in a 90-day window", got)
	}
	in.WindowStart = now.AddDate(0, 0, -30)
	if got := windowCoveredByTrace(&in, oldest); got != 1 {
		t.Errorf("window coverage = %v, want 1 when the window and the horizon are the same length: "+
			"the hours between the two boundaries are not missing history", got)
	}
}

// A window narrower than twice the trend span holds no rows before its own start, by construction:
// usage is queried WHERE ts >= start. Reading the horizon off those rows made every `--since 7d`,
// every label filter and every drill accuse the source of deleting history the store still holds.
func TestTheHorizonLineDoesNotBlameRetentionForANarrowWindow(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	in := Input{
		Now: now, Recent: 7 * 24 * time.Hour, WindowStart: now.AddDate(0, 0, -7),
		// The window's own rows can only start inside it; the store goes back much further.
		Usage:        []store.UsageRow{{Day: "2026-08-06", Tool: "claude-code", Model: "m", LinesAdded: 10}},
		HistoryStart: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
	}
	var r Result
	stampHorizon(&r, &in)

	if len(r.Caveats) != 1 {
		t.Fatalf("caveats = %v, want one horizon line", r.Caveats)
	}
	if strings.Contains(r.Caveats[0], "never held") || strings.Contains(r.Caveats[0], "already deleted") {
		t.Errorf("a 7-day window over a store reaching back to 2025 is told its history was deleted: %q", r.Caveats[0])
	}
	if !strings.Contains(r.Caveats[0], "2025-03-01") {
		t.Errorf("the line does not state where the store's history begins: %q", r.Caveats[0])
	}
}

// The other direction still has to fire: a store that genuinely begins inside the earlier span is
// the case B156 exists for.
func TestTheHorizonLineStillFiresWhenTheStoreBeginsInsideThePriorSpan(t *testing.T) {
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	in := Input{
		Now: now, Recent: 7 * 24 * time.Hour, WindowStart: now.AddDate(0, 0, -30),
		Usage:        []store.UsageRow{{Day: "2026-08-06", Tool: "claude-code", Model: "m", LinesAdded: 10}},
		HistoryStart: now.AddDate(0, 0, -10),
	}
	var r Result
	stampHorizon(&r, &in)

	if len(r.Caveats) != 1 || !strings.Contains(r.Caveats[0], "never held") {
		t.Errorf("caveats = %v, want the line naming a base the store never held", r.Caveats)
	}
}

// A scope with no edits already carries the scope sentence from overScope; appending it again
// printed the same paragraph twice on the same panel.
func TestTheScopeSentenceIsPrintedOnce(t *testing.T) {
	in := Input{Trace: traceOf(
		sequence(t, "a", "p", "a c r1 a c r2"),
		sequence(t, "b", "p", "a c r1 a c r2"),
	)}
	got := Evaluate(editLoopsValidator{}, &in)
	var scopes int
	for _, c := range got.Caveats {
		if strings.HasPrefix(c, "Scope: ") {
			scopes++
		}
	}
	if scopes != 1 {
		t.Errorf("the scope sentence appears %d times: %v", scopes, got.Caveats)
	}
}

// The reachable boundary is not 1: a sequence's first edit of a file is never a repeat, so n edits
// top out at (n-1)/n. A floor between that and 1 finds nothing by arithmetic, and saying "too few
// sessions" about forty ranked ones is its own wrong answer.
func TestAFloorAboveWhatTheWindowCouldReachWithholdsAVerdictAndSaysWhy(t *testing.T) {
	// Chosen so the floor lands BETWEEN the reachable rate and 1 (0.932 against 11/12 = 0.917):
	// a fixture whose floor merely exceeded 1 would pass under the old domain check too, and prove
	// nothing about the boundary this guards.
	p := repeatProfile{Judged: []repeatEdits{
		{Project: "p", Edits: 12, Repeats: 4},
		{Project: "p", Edits: 12, Repeats: 5},
		{Project: "p", Edits: 12, Repeats: 6},
		{Project: "p", Edits: 12, Repeats: 6},
		{Project: "p", Edits: 12, Repeats: 6},
		{Project: "p", Edits: 12, Repeats: 7},
		{Project: "p", Edits: 12, Repeats: 9},
		{Project: "p", Edits: 12, Repeats: 10},
	}}
	p.Edits, p.Repeats = 96, 53
	p.selectOutliers()

	rates := make([]float64, len(p.Judged))
	for i := range p.Judged {
		rates[i] = p.Judged[i].Rate()
	}
	floor, ok := outlierFloor(rates)
	if !ok || floor <= p.reachableRate() || floor >= 1 {
		t.Fatalf("this fixture no longer isolates the boundary: floor = %v (ok=%v), reachable = %v; "+
			"want reachable < floor < 1, or the old domain check would pass it too", floor, ok, p.reachableRate())
	}
	if p.Spread {
		t.Errorf("a floor of %v was accepted over sessions that top out at %v", p.Floor, p.reachableRate())
	}
	if !p.Unreachable {
		t.Fatal("the profile does not record why it withheld a verdict")
	}
	if takeaway := editLoopsTakeaway(&p); strings.Contains(takeaway, "Too few sessions") {
		t.Errorf("takeaway = %q, want it to explain the unreachable line rather than blame sample size", takeaway)
	}
}
