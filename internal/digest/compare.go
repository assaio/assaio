package digest

import (
	"sort"
)

// moversShown bounds each mover list. The render always states the whole count beside it,
// so a bounded list never reads as the complete one.
const moversShown = 5

// Mover is one dimension's movement between two runs. Was is the previous value and Now the
// current one; a dimension absent from either side moved from or to nothing, which is a
// different fact from moving to zero and is rendered as such.
type Mover struct {
	Name     string
	Was, Now int64
	Appeared bool
	Vanished bool
}

// Delta is the movement itself, positive or negative.
func (m Mover) Delta() int64 { return m.Now - m.Was }

// VerdictChange is one validator whose read changed. From and To are read keys; Confidence
// is what the current read rests on, because a move from good to watch means one thing on
// high confidence and is noise on insufficient.
type VerdictChange struct {
	Name       string
	From, To   string
	Confidence string
}

// Weak reports whether this change rests on too little to act on. Such a change is still
// shown -- hiding it would be a second silent filter -- but it is shown as what it is.
func (v VerdictChange) Weak() bool {
	return v.Confidence == "" || v.Confidence == "insufficient" || v.Confidence == "low"
}

// GainedBasis reports a move out of or into "neutral", which is what a validator reads when
// the window holds nothing it can answer. That is data arriving or leaving, not a verdict
// improving, and rendering it as good news would be the fabricated improvement this project
// refuses.
func (v VerdictChange) GainedBasis() bool { return v.From == "neutral" || v.To == "neutral" }

// Digest is one comparison. Previous is nil on a first run, which the render says plainly
// instead of comparing against zero and reporting everything as new.
type Digest struct {
	Now      Snapshot
	Previous *Snapshot
	Models   []Mover
	Projects []Mover
	Verdicts []VerdictChange
	Caveats  []string
	// Unevaluated names metrics this run could not produce -- a metric plugin that failed,
	// typically. Their absence would otherwise read as a verdict that vanished, so they are
	// named as what they are and excluded from the changes.
	Unevaluated []string
	// Pseudonym renames a project at render time. It is deliberately not applied when the
	// snapshot is stored: a stored pseudonym would change key the moment privacy.anonymize
	// is toggled, and every project would then read as one that appeared and one that
	// vanished. Model names are never renamed -- which model ran is a fact about the vendor.
	Pseudonym func(project string) string
}

// WithPseudonym sets the render-time renamer and returns the digest, so a caller can build
// and configure it in one expression.
func (d *Digest) WithPseudonym(f func(string) string) *Digest {
	d.Pseudonym = f
	return d
}

// Compare builds the digest. Everything it refuses to compare becomes a caveat rather than
// a silent omission: the reader has to be able to tell "nothing moved" from "this could not
// be compared".
// unevaluated are metrics that did not run this time (a failed metric plugin), named by the
// caller. Their verdicts are missing because the metric was not evaluated, not because it
// stopped applying, and reporting that as a change would be movement that never happened.
func Compare(now *Snapshot, previous *Snapshot, unevaluated []string) Digest {
	d := Digest{Now: *now, Previous: previous, Unevaluated: unevaluated}
	if previous == nil {
		return d
	}
	d.Caveats = comparabilityCaveats(now, previous)
	d.Models = movers(previous.Models, now.Models)
	d.Projects = movers(previous.Projects, now.Projects)
	d.Verdicts = verdictChanges(previous.Verdicts, now.Verdicts, now.Confidence, unevaluated)
	return d
}

// movers ranks dimensions by the size of their movement, largest first, keeping the
// direction. A name present on one side only is reported as appearing or vanishing.
func movers(was, now map[string]int64) []Mover {
	seen := make(map[string]bool, len(was)+len(now))
	out := make([]Mover, 0, len(was)+len(now))
	for name, v := range now {
		seen[name] = true
		m := Mover{Name: name, Was: was[name], Now: v}
		_, existed := was[name]
		m.Appeared = !existed
		out = append(out, m)
	}
	for name, v := range was {
		if seen[name] {
			continue
		}
		out = append(out, Mover{Name: name, Was: v, Now: 0, Vanished: true})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := abs(out[i].Delta()), abs(out[j].Delta())
		if a != b {
			return a > b
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// verdictChanges reports validators whose read moved, and those that stopped or started
// being reported at all -- a metric that disappeared is a change worth seeing.
func verdictChanges(was, now, confidence map[string]string, unevaluated []string) []VerdictChange {
	skip := make(map[string]bool, 2*len(unevaluated))
	for _, name := range unevaluated {
		skip[name], skip["plugin:"+name] = true, true
	}
	out := make([]VerdictChange, 0)
	for name, to := range now {
		if from, ok := was[name]; ok && from != to {
			out = append(out, VerdictChange{Name: name, From: from, To: to, Confidence: confidence[name]})
		}
	}
	for name, from := range was {
		if _, ok := now[name]; !ok && !skip[name] {
			out = append(out, VerdictChange{Name: name, From: from, To: "—"})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
