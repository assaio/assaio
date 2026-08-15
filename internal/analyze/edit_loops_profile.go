package analyze

import (
	"sort"

	"github.com/assaio/assaio/internal/trace"
)

// The window's side of repeat edits: which sequences carry a rate at all, what the window's own
// spread makes an outlier, and how the judged ones roll up per repository. Counting one sequence
// is edit_loops_repeats.go; deciding what the counts mean across a window is here.

// repeatProfile is one scope's repeat-edit picture for the window.
type repeatProfile struct {
	// Judged holds the sequences with enough edits to carry a rate; Thin counts the ones left
	// out for having too few, so a window of short sequences reports that rather than a rate
	// read off three edits.
	Judged []repeatEdits
	Thin   int
	// Edits, Repeats and Untargeted are the window totals across every sequence in scope,
	// including the thin ones: the window rate is a fact about the work, not about which
	// sequences were thick enough to rank.
	Edits, Repeats, Untargeted int
	// Outliers are the judged sequences whose rate stands outside this window's own spread.
	// Empty is a real answer -- an even window has none -- and is not the same as Spread being
	// false, which means the window had no spread to measure against at all.
	Outliers []repeatEdits
	Floor    float64
	Spread   bool
	// Unreachable records that a floor was computed and then discarded for sitting at or above the
	// highest rate this window's sequences could possibly reach. It is kept apart from Spread
	// because the two silences have different explanations, and reporting "too few sessions" over
	// forty ranked ones is its own wrong answer.
	Unreachable bool
}

// editLoopsMinEdits is the per-sequence edit floor below which a rate says nothing: one repeat out
// of three edits is 33% and describes a coincidence.
const editLoopsMinEdits = 10

// buildRepeats reads the scope's sequences into the window's profile. Outliers come from the
// window's own median and spread rather than a fixed ceiling, because no honest ceiling exists:
// the maintainer's own store sits at a 25% repeat rate, and shipping that as the line between
// healthy and not would publish one machine's habits as everyone's threshold.
func buildRepeats(v *trace.View) repeatProfile {
	var p repeatProfile
	for i := range v.Sequences {
		r := countRepeatEdits(&v.Sequences[i])
		p.Edits += r.Edits
		p.Repeats += r.Repeats
		p.Untargeted += r.Untargeted
		if r.Edits < editLoopsMinEdits {
			if r.Edits > 0 || r.Untargeted > 0 {
				p.Thin++
			}
			continue
		}
		p.Judged = append(p.Judged, r)
	}
	p.selectOutliers()
	return p
}

// selectOutliers fills Outliers, Floor and Spread from the judged sequences' own distribution.
func (p *repeatProfile) selectOutliers() {
	if len(p.Judged) < editLoopsMinSequences {
		return
	}
	rates := make([]float64, len(p.Judged))
	for i := range p.Judged {
		rates[i] = p.Judged[i].Rate()
	}
	floor, ok := outlierFloor(rates)
	if !ok {
		return
	}
	// The domain boundary is not 1: a sequence's first edit of a file can never be a repeat, so a
	// sequence of n edits tops out at (n-1)/n, and on a window whose thickest session has 12 edits
	// nothing can exceed 0.917. A floor above whatever this window could actually reach finds
	// nothing by arithmetic and would publish the favourable read with a full purity gauge -- a
	// clean bill of health on exactly the windows with the most variation. The shared rule comes
	// from unbounded token counts, where the question does not arise.
	if floor >= p.reachableRate() {
		p.Unreachable = true
		return
	}
	p.Floor, p.Spread = floor, true
	for i := range p.Judged {
		if p.Judged[i].Rate() > floor {
			p.Outliers = append(p.Outliers, p.Judged[i])
		}
	}
	sort.Slice(p.Outliers, func(i, j int) bool { return p.Outliers[i].Rate() > p.Outliers[j].Rate() })
}

// reachableRate is the highest repeat rate any judged sequence in this window could reach, since
// the first edit of each file is never a repeat.
func (p *repeatProfile) reachableRate() float64 {
	var top float64
	for i := range p.Judged {
		if n := p.Judged[i].Edits; n > 0 {
			if r := float64(n-1) / float64(n); r > top {
				top = r
			}
		}
	}
	return top
}

// Rate is the window's repeat share across every sequence in scope.
func (p *repeatProfile) Rate() float64 { return shareOf(int64(p.Repeats), int64(p.Edits)) }

// Worst is the highest rate any judged sequence reached, and whether there was one to ask of.
func (p *repeatProfile) Worst() (repeatEdits, bool) {
	var worst repeatEdits
	var found bool
	for i := range p.Judged {
		if !found || p.Judged[i].Rate() > worst.Rate() {
			worst, found = p.Judged[i], true
		}
	}
	return worst, found
}

// OutlierEdits is how much of the window's editing sits inside the sequences that stand out.
func (p *repeatProfile) OutlierEdits() int {
	var n int
	for i := range p.Outliers {
		n += p.Outliers[i].Edits
	}
	return n
}

// byProject rolls the judged sequences up per repository, keeping only those with enough edits to
// carry a rate for the same reason a sequence needs them. Sorted by rate descending.
func (p *repeatProfile) byProject() []repeatEdits {
	totals := make(map[string]*repeatEdits)
	for i := range p.Judged {
		name := p.Judged[i].Project
		if name == "" {
			continue
		}
		if totals[name] == nil {
			totals[name] = &repeatEdits{Project: name}
		}
		totals[name].Edits += p.Judged[i].Edits
		totals[name].Repeats += p.Judged[i].Repeats
	}
	out := make([]repeatEdits, 0, len(totals))
	for _, r := range totals {
		if r.Edits >= editLoopsMinEdits {
			out = append(out, *r)
		}
	}
	// Rate alone is not a total order over a map-iteration-ordered slice: 2/10 and 3/15 both
	// give 0.2 and editLoopsMinEdits is 10, so ties are ordinary and the top-5 cut changed
	// between identical runs. The name settles them, as it does in copilot.dominantModel,
	// dashboard.TopProject and cache.missCauseFigure.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rate() != out[j].Rate() {
			return out[i].Rate() > out[j].Rate()
		}
		return out[i].Project < out[j].Project
	})
	return out
}
