package trace

import "github.com/assaio/assaio/internal/humanize"

// scopeCovers says what a scope holds in a reader's terms rather than the constant's.
var scopeCovers = map[string]string{
	Interactive:  "sessions a person ran from a terminal",
	SubAgent:     "sequences a sub-agent ran inside another session",
	Programmatic: "runs an SDK caller made, with nobody there to interrupt",
	Unstated:     "runs whose source does not record how they were started",
}

// Caveat states the population a figure was read over and the one it leaves out, with both counts
// in the same sentence. Every detector renders this rather than its own wording: the sequence
// share and the step share differ by an order of magnitude -- 89% against 5.7% on the audited
// store -- and a detector free to phrase its own exclusion would sooner or later quote whichever
// of the two flattered the finding.
func (v *View) Caveat() string {
	covers := scopeCovers[v.Scope]
	if covers == "" {
		covers = v.Scope
	}
	if v.ExcludedSequences == 0 {
		return "Scope: " + covers + " -- " + humanize.Int(int64(len(v.Sequences))) +
			" sequence(s), which is every sequence in this window."
	}
	return "Scope: " + covers + " -- " + humanize.Int(int64(len(v.Sequences))) +
		" sequence(s) holding " + humanize.Percent(1-v.ExcludedStepShare()) +
		" of the window's steps. The other " + humanize.Int(int64(v.ExcludedSequences)) +
		" sequence(s) are excluded from every figure here rather than averaged into it: a " +
		"sub-agent's run and an SDK caller's are different work and share no rate with a person's."
}
