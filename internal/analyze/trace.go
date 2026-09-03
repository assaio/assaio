package analyze

import (
	"time"

	"github.com/assaio/assaio/internal/trace"
)

// TraceReader marks a Validator that reads Input.Trace, and makes it name the one scope it reads.
// The scope is part of the interface rather than a sentence in the takeaway because it *is* the
// denominator: 89% of the sequences on the audited store are one-shot SDK calls holding 5.7% of
// its steps, so a detector that forgot to say which population it measured would publish a rate
// over whatever mix the machine happened to hold. Every value trace declares is legal; a scope
// with no sequences yields a no-data read, which is a real answer.
//
// The CLI also reads this to skip the step query when nothing selected needs it: that read costs
// 2.5s on a 339,000-step store, which no `report` run and no non-trace validator should pay.
type TraceReader interface {
	TraceScope() string
}

// cannotDistinguish opens the refusal every detector ships with: the sentence naming the readings
// its pattern cannot be told apart from. A detector fires on a pattern, and a pattern is not a
// fault, so the ambiguity travels beside the finding rather than living in a document nobody opens
// next to it. Shared as a constant rather than typed per detector so the gate asserting it can be
// structural: see TestEveryTraceReaderDeclaresWhatItCannotDistinguish.
const cannotDistinguish = "Cannot distinguish"

// NeedsTrace reports whether any of vs reads the step sequences, so a caller can decide whether
// to load them at all.
func NeedsTrace(vs []Validator) bool {
	for _, v := range vs {
		if _, ok := v.(TraceReader); ok {
			return true
		}
	}
	return false
}

// overScope stamps what a detector's figures were read over: the observations behind them, the
// share of the window's steps the scope covers, and the sentence naming what was left out. Both
// detectors go through this so neither can describe its own population differently from the
// other -- or, worse, quote whichever of the two exclusion shares flattered its finding.
func (r *Result) overScope(in *Input, v *trace.View, unit string) {
	r.restsOn(len(v.Sequences), unit)
	covered := windowCoveredByTrace(in, v.Oldest)
	r.covering((1 - v.ExcludedStepShare()) * covered)
	r.Caveats = append(r.Caveats, v.Caveat())
	if covered < 1 {
		r.Caveats = append(r.Caveats, "Prov.: sequences are kept for `trace.horizon_days` while usage records are kept indefinitely, so this window reaches back further than the sequences behind these figures do -- "+
			v.Oldest.UTC().Format("2006-01-02")+" is as far as they go. The rate above is that span's, not the window's.")
	}
}

// windowCoveredByTrace is the share of the queried window the stored sequences can speak for at
// all, 0..1. Without it a `--since 90d` over a 30-day step horizon reported a high-confidence
// 90-day rate computed from a third of it: the scope share alone answers "how much of the trace",
// which is not the question a confidence figure is read as.
func windowCoveredByTrace(in *Input, oldest time.Time) float64 {
	if in.WindowStart.IsZero() || oldest.IsZero() {
		return 1
	}
	// Whole days, not seconds: a window asked for as "30d" and a horizon of 30 days differ by the
	// hours between the two boundaries, and reporting that as missing coverage would fire the
	// truncation caveat on every ordinary run.
	windowDays := int(in.Now.Sub(in.WindowStart).Hours() / 24)
	if windowDays <= 0 {
		return 1
	}
	missing := int(oldest.Sub(in.WindowStart).Hours() / 24)
	if missing <= 0 {
		return 1
	}
	return clamp01(float64(windowDays-missing) / float64(windowDays))
}

// noSequencesTakeaway separates a store holding no sequences at all from one whose sequences are
// all outside the scope asked for: the first asks for a backfill, the second is a real answer
// about the window. Shared by every detector, because the difference matters identically to all
// of them and two wordings would eventually disagree about which case is which.
func noSequencesTakeaway(set *trace.Set) string {
	if set.Empty() {
		return "No step sequence is stored for this window. Sequences are read from the tool's own transcripts by `backfill`, kept for `trace.horizon_days`, and only Claude Code and Codex record them today; a store filled by `sync` holds none at all, because the team-server contract carries usage records and not sequences (ADR 0012)."
	}
	return "Sequences are stored for this window, but none is a session someone ran from a terminal -- the only scope this reads."
}
