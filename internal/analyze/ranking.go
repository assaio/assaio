package analyze

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/assaio/assaio/internal/humanize"
)

// RankedMax is how many findings lead a window. Twenty-one reads render as twenty-one reads, and
// a reader facing a wall of equally weighted verdicts acts on none of them -- past some count
// each additional correct measurement is worth less than the one before it. The rest are not
// hidden by this: they are the same report, one screen down.
const RankedMax = 3

// Ranked is one finding worth a week's attention, with the reasons that put it there.
// Deliberately no score: a number would invite comparison between findings measuring
// different things, and the ordering is only as good as the facts behind it.
type Ranked struct {
	Result  *Result  `json:"-"`
	Name    string   `json:"name"`
	Title   string   `json:"title"`
	Reasons []string `json:"reasons"`
}

// MarkLead stamps Lead on the few results worth a week's attention, so a machine reader gets
// the same ordering the terminal shows without re-deriving it from the fields -- and gets it
// without the output changing shape, which would break every script already reading it.
func MarkLead(results []Result) {
	// Cleared first: a Result that arrived carrying a Lead nobody computed -- an exec metric
	// plugin's, before the wire learned to drop it -- would otherwise be encoded as a rank
	// this ordering never produced, beside the ranks it did.
	for i := range results {
		results[i].Lead = nil
	}
	for i, r := range Rank(results) {
		r.Result.Lead = &Lead{Rank: i + 1, Reasons: r.Reasons}
	}
}

// Rank orders the findings a window can act on, most worth a week's attention first, and
// returns at most RankedMax of them. It ranks by what is already knowable and nothing else:
// whether the verdict asks for anything to be done, how strong its evidence is, and how much
// of the window its subject reaches. Expected impact is deliberately absent -- it needs the
// outcome link, and guessing it would be the fabricated number this project refuses.
//
// A window whose findings are all weak returns none rather than promoting the least weak one:
// leading with a verdict that rests on too little is how a reader learns to distrust the
// order itself.
func Rank(results []Result) []Ranked {
	actionable := make([]*Result, 0, len(results))
	for i := range results {
		if actionableFinding(&results[i]) {
			actionable = append(actionable, &results[i])
		}
	}
	sort.SliceStable(actionable, func(i, j int) bool {
		return moreWorthAttention(actionable[i], actionable[j])
	})
	if len(actionable) > RankedMax {
		actionable = actionable[:RankedMax]
	}
	out := make([]Ranked, 0, len(actionable))
	for _, r := range actionable {
		out = append(out, Ranked{Result: r, Name: r.Name, Title: r.Title, Reasons: rankReasons(r)})
	}
	return out
}

// actionableFinding reports whether a verdict is one a week could act on. Two facts decide
// it, and both are already on the Result: the read has to be asking for something -- a
// favorable read is a confirmation and a withheld one is a silence, neither is work -- and
// the evidence has to be strong enough that acting on it is not acting on noise.
func actionableFinding(r *Result) bool {
	return r.Read.Key == "watch" && promotableConfidence(r.Confidence.Label)
}

// promotableConfidence is the floor for leading a report. "low" stays in the ledger where its
// own confidence line is beside it; it never leads.
func promotableConfidence(label string) bool {
	return label == ConfidenceHigh || label == ConfidenceMedium
}

// moreWorthAttention orders two actionable findings: stronger evidence first, then the one
// whose subject reaches more of the window, then by name so the order is stable and diffable.
func moreWorthAttention(a, b *Result) bool {
	if ea, eb := evidenceRank(a.Confidence.Label), evidenceRank(b.Confidence.Label); ea != eb {
		return ea > eb
	}
	if ra, rb := a.Confidence.signalShare(), b.Confidence.signalShare(); ra != rb {
		return ra > rb
	}
	if wa, wb := weakestShare(&a.Confidence), weakestShare(&b.Confidence); wa != wb {
		return wa > wb
	}
	return a.Name < b.Name
}

func evidenceRank(label string) int {
	switch label {
	case ConfidenceHigh:
		return 2
	case ConfidenceMedium:
		return 1
	default:
		return 0
	}
}

// weakestShare is the thinnest coverage axis behind a verdict, or 1 when none is thin.
func weakestShare(c *Confidence) float64 {
	if _, share, weak := weakestAxis(c); weak {
		return share
	}
	return 1
}

// RenderRankingText writes the lead section: what is worth a week's attention, in order, each
// with the reasons that put it there -- and, when nothing qualifies, the sentence saying so
// rather than a promoted verdict nobody should act on. total is how many reads the window
// produced, so the reader knows what the few were chosen from.
func RenderRankingText(w io.Writer, ranked []Ranked, total int) error {
	reads := "reads"
	if total == 1 {
		reads = "read"
	}
	if len(ranked) == 0 {
		_, err := fmt.Fprintf(w,
			"Worth a week's attention: nothing here. All %d %s are either fine, withheld, or resting on too little to act on.\n\n",
			total, reads)
		return err
	}
	if _, err := fmt.Fprintf(w, "Worth a week's attention, of %d %s:\n", total, reads); err != nil {
		return err
	}
	for i := range ranked {
		if _, err := fmt.Fprintf(w, "  %d. %s — %s\n",
			i+1, ranked[i].Title, strings.Join(ranked[i].Reasons, " · ")); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w,
		"  Order is these reasons, not a score; findings whose reasons match are ordered by name. Every read is below.\n\n")
	return err
}

// rankReasons says why this finding leads, in the order the ranking weighed them. Showing
// them is the point: an ordering a reader cannot audit is a score wearing a list's clothes.
func rankReasons(r *Result) []string {
	reasons := []string{
		"flagged for a closer look",
		r.Confidence.Label + " confidence · " + humanize.Int(int64(r.Confidence.Samples)) + " " + r.Confidence.Unit,
		reachReason(&r.Confidence),
	}
	if axis, share, weak := weakestAxis(&r.Confidence); weak {
		reasons = append(reasons, "thinnest evidence: "+axis+" coverage "+humanize.Percent(share))
	}
	return reasons
}

// reachReason states how much of the window the subject reaches, always -- a reason left out
// where it is strongest would make the order unauditable exactly where it is most confident.
func reachReason(c *Confidence) string {
	if c.Signal == nil {
		return "covers the whole window"
	}
	return "covers " + humanize.Percent(*c.Signal) + " of the window"
}
