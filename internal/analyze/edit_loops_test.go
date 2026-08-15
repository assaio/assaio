package analyze

import (
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/store"
)

// The definition is CodeBurn's, and these are the cases that separate it from the naive one:
// adjacency is not enough, sameness of file is required, and the command has to fall *between*
// the two edits rather than anywhere in the sequence.
func TestRepeatEditsFollowThePublishedDefinition(t *testing.T) {
	tests := []struct {
		name                       string
		spec                       string
		edits, repeats, untargeted int
	}{
		{name: "a file re-edited after a command", spec: "e1 c e1", edits: 2, repeats: 1},
		{name: "re-edited with nothing run in between", spec: "e1 e1", edits: 2, repeats: 0},
		{name: "a different file after a command", spec: "e1 c e2", edits: 2, repeats: 0},
		{name: "the command runs after both edits", spec: "e1 e1 c", edits: 2, repeats: 0},
		{name: "a read in between is not a command", spec: "e1 r1 e1", edits: 2, repeats: 0},
		{name: "three passes, two of them repeats", spec: "e1 c e1 c e1", edits: 3, repeats: 2},
		{name: "two files interleaved", spec: "e1 e2 c e1 e2", edits: 4, repeats: 2},
		{name: "an edit naming no file is not in the denominator", spec: "e0 c e0", untargeted: 2},
		{name: "a failed edit still counts as a pass", spec: "e1!err c e1", edits: 2, repeats: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := sequence(t, "s", "p", tt.spec)
			got := countRepeatEdits(&seq)
			if got.Edits != tt.edits || got.Repeats != tt.repeats || got.Untargeted != tt.untargeted {
				t.Errorf("edits/repeats/untargeted = %d/%d/%d, want %d/%d/%d",
					got.Edits, got.Repeats, got.Untargeted, tt.edits, tt.repeats, tt.untargeted)
			}
		})
	}
}

// The scope is the denominator. A sub-agent's sequence repeats at a third of a person's rate on
// the audited store, so blending the two would report a number describing neither.
func TestEditLoopsReadsOnlyTheScopeItDeclares(t *testing.T) {
	const looping = "e1 c e1 c e1 c e1 c e1 c e1 c e1 c e1 c e1 c e1 c e1"
	in := Input{Trace: traceOf(
		sequence(t, "mine", "assaio", looping),
		subAgentSequence(t, "mine", "agent-7", "assaio", looping),
		programmaticSequence(t, "script", looping),
	)}

	got := Evaluate(editLoopsValidator{}, &in)

	if got.Confidence.Samples != 1 {
		t.Errorf("counted %d sessions, want 1: only the interactive sequence is in scope", got.Confidence.Samples)
	}
	if !strings.Contains(figureNote(t, &got, "repeat-edit rate"), "of 11 edits") {
		t.Errorf("denominator spans more than the declared scope: %+v", got.Figures)
	}
	if !hasCaveatContaining(got.Caveats, "excluded from every figure here") {
		t.Errorf("no caveat states what the scope left out: %v", got.Caveats)
	}
}

// A detector fires on a pattern, and a pattern is not a fault. The refusal is part of the
// contract, so it is asserted rather than left to review.
func TestEditLoopsDeclaresWhatItCannotDistinguish(t *testing.T) {
	in := Input{Trace: favorableTrace(t)}
	got := Evaluate(editLoopsValidator{}, &in)
	for _, want := range []string{"Cannot distinguish", "never that it was a test", "arXiv:2205.06537"} {
		if !hasCaveatContaining(got.Caveats, want) {
			t.Errorf("caveats do not mention %q: %v", want, got.Caveats)
		}
	}
	if !hasCaveatContaining(got.Caveats, "No cost is claimed") {
		t.Errorf("the measured absence of a cost premium is not stated: %v", got.Caveats)
	}
}

// The looping sequence is the one that stands out; the seven ordinary ones must not, or the
// finding is "every session is a finding" and the reader learns nothing.
func TestEditLoopsNamesOnlyTheSessionsStandingOutsideTheWindow(t *testing.T) {
	in := Input{Trace: favorableTrace(t)}
	got := Evaluate(editLoopsValidator{}, &in)

	if value := figureValue(t, &got, "sessions standing out"); value != "1" {
		t.Errorf("sessions standing out = %q, want 1 of the eight", value)
	}
	if got.Read.Key != "watch" {
		t.Errorf("Read = %+v, want a watch: one session sits far outside this window", got.Read)
	}
}

// Too few sessions to have a typical one is not "none stand out": the figure has to say it could
// not ask, or an unjudgeable window reads as a clean bill of health.
func TestEditLoopsWithholdsAVerdictWhenTheWindowHasNoSpread(t *testing.T) {
	const spec = "e1 c e1 c e1 c e1 c e1 c e1 c e1 c e1 c e1 c e1 c e1"
	in := Input{Trace: traceOf(sequence(t, "only", "assaio", spec))}

	got := Evaluate(editLoopsValidator{}, &in)

	if value := figureValue(t, &got, "sessions standing out"); value != "—" {
		t.Errorf("sessions standing out = %q, want a dash: one session has nothing to stand out from", value)
	}
	if got.Read != noDataRead {
		t.Errorf("Read = %+v, want no verdict", got.Read)
	}
	if got.Purity != 0.5 {
		t.Errorf("Purity = %v, want the neutral 0.5 rather than a gauge earned by having no data", got.Purity)
	}
}

// A store with no sequences and a store whose sequences are all out of scope need different
// answers: one asks for a backfill, the other is a fact about the window.
func TestEditLoopsSeparatesNoSequencesFromNoneInScope(t *testing.T) {
	empty := Evaluate(editLoopsValidator{}, &Input{})
	if !strings.Contains(empty.Takeaway, "backfill") {
		t.Errorf("takeaway on a store with no sequences = %q, want it to name backfill", empty.Takeaway)
	}
	outOfScope := Input{Trace: traceOf(programmaticSequence(t, "script", "e1 c e1"))}
	got := Evaluate(editLoopsValidator{}, &outOfScope)
	if strings.Contains(got.Takeaway, "backfill") {
		t.Errorf("takeaway = %q, want it to say the sequences are out of scope rather than missing", got.Takeaway)
	}
}

// programmaticSequence is an SDK caller's run: the scope holding 89% of the audited store's main-loop
// sessions and 5% of its steps.
func programmaticSequence(t *testing.T, id, spec string) store.Timeline {
	t.Helper()
	seq := sequence(t, id, "assaio", spec)
	seq.Entrypoint = "sdk-py"
	return seq
}

func figureValue(t *testing.T, r *Result, label string) string {
	t.Helper()
	for i := range r.Figures {
		if r.Figures[i].Label == label {
			return r.Figures[i].Value
		}
	}
	t.Fatalf("no figure labelled %q in %+v", label, r.Figures)
	return ""
}

func figureNote(t *testing.T, r *Result, label string) string {
	t.Helper()
	for i := range r.Figures {
		if r.Figures[i].Label == label {
			return r.Figures[i].Note
		}
	}
	t.Fatalf("no figure labelled %q in %+v", label, r.Figures)
	return ""
}

// TestEditLoopsProjectBarsAreDeterministicOnTiedRates is the determinism rule: byProject sorted on Rate alone,
// over a slice built from map iteration. 2/10 and 3/15 are both 0.2 and editLoopsMinEdits is 10,
// so ties are ordinary -- and the top-5 cut then changed between two runs over identical data.
// The repo holds this line everywhere else (copilot.dominantModel, dashboard.TopProject,
// cache.missCauseFigure).
func TestEditLoopsProjectBarsAreDeterministicOnTiedRates(t *testing.T) {
	p := repeatProfile{Judged: []repeatEdits{
		{Project: "delta", Edits: 10, Repeats: 2},
		{Project: "alpha", Edits: 15, Repeats: 3},
		{Project: "charlie", Edits: 20, Repeats: 4},
		{Project: "bravo", Edits: 10, Repeats: 5},
	}}
	want := []string{"bravo", "alpha", "charlie", "delta"}
	for run := range 40 {
		got := p.byProject()
		names := make([]string, len(got))
		for i := range got {
			names[i] = got[i].Project
		}
		if strings.Join(names, ",") != strings.Join(want, ",") {
			t.Fatalf("run %d ordered the tied projects %v, want %v", run, names, want)
		}
	}
}
