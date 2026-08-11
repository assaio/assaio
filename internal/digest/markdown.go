package digest

import (
	"fmt"
	"strings"

	"github.com/assaio/assaio/internal/humanize"
)

// Markdown renders the digest for a mail body, a Slack post or a file. Delivery is the
// person's own script: this writes the message and nothing else.
func (d *Digest) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# assaio digest — %s\n\n", d.Now.TakenAt.Local().Format("2006-01-02"))

	if d.Previous == nil {
		fmt.Fprintf(&b, "Window: last %s. **First digest — nothing to compare against yet.**\n\n", d.Now.Window)
		fmt.Fprintf(&b, "%s\n\nRun this again after the next window and it will report what moved.\n\n", d.stateLine())
		b.WriteString(d.caveatSection())
		return b.String()
	}
	fmt.Fprintf(&b, "Window: last %s, compared against the run of %s.\n\n",
		d.Now.Window, d.Previous.TakenAt.Local().Format("2006-01-02 15:04"))
	b.WriteString(d.totalsSection())
	b.WriteString(d.verdictSection())
	b.WriteString(d.unevaluatedSection())
	b.WriteString(moverSection("Models, by tokens", d.Models))
	b.WriteString(moverSection("Projects, by AI-written lines", d.renamed(d.Projects)))
	b.WriteString(d.caveatSection())
	return b.String()
}

// stateLine states the window as it stands, for the run that has nothing to compare to.
func (d *Digest) stateLine() string {
	return fmt.Sprintf("This window: %s tokens, %s, %s AI-written lines across %s sessions.",
		humanize.Count(d.Now.Tokens), money(d.Now.Cost), humanize.Count(d.Now.Lines),
		humanize.Int(int64(d.Now.Sessions)))
}

func (d *Digest) totalsSection() string {
	p := d.Previous
	var b strings.Builder
	b.WriteString("## What moved\n\n")
	b.WriteString("| | previous | now | change |\n|---|---:|---:|---:|\n")
	fmt.Fprintf(&b, "| tokens | %s | %s | %s |\n",
		humanize.Count(p.Tokens), humanize.Count(d.Now.Tokens), relative(p.Tokens, d.Now.Tokens))
	fmt.Fprintf(&b, "| AI-written lines | %s | %s | %s |\n",
		humanize.Count(p.Lines), humanize.Count(d.Now.Lines), relative(p.Lines, d.Now.Lines))
	fmt.Fprintf(&b, "| sessions | %s | %s | %s |\n",
		humanize.Int(int64(p.Sessions)), humanize.Int(int64(d.Now.Sessions)),
		relative(int64(p.Sessions), int64(d.Now.Sessions)))
	switch {
	case p.Cost != nil && d.Now.Cost != nil:
		fmt.Fprintf(&b, "| estimated cost | %s | %s | %s |\n", money(p.Cost), money(d.Now.Cost),
			relativeFloat(*p.Cost, *d.Now.Cost))
	default:
		// Rendered as the dash it is rather than dropped: a missing row reads as a table that
		// never had a cost, and the caveats below refer to a cost movement that would not be there.
		fmt.Fprintf(&b, "| estimated cost | %s | %s | — |\n", money(p.Cost), money(d.Now.Cost))
	}
	b.WriteString("\n")
	return b.String()
}

func (d *Digest) verdictSection() string {
	var b strings.Builder
	b.WriteString("## Verdict changes\n\n")
	if len(d.Verdicts) == 0 {
		b.WriteString("No validator changed its read since the last run.\n\n")
		return b.String()
	}
	for _, v := range d.Now.LeadsFirst(d.Verdicts) {
		fmt.Fprintf(&b, "- **%s** %s → %s%s\n", v.Name, strings.ToUpper(v.From), strings.ToUpper(v.To),
			confidenceNote(v))
	}
	b.WriteString("\n")
	return b.String()
}

// confidenceNote marks a change that rests on too little to act on. It is shown rather than
// filtered, because a silently dropped change is a second hidden judgement -- but a move on
// insufficient data must not read like a move on strong data.
func confidenceNote(v VerdictChange) string {
	switch {
	case v.To == "—":
		return ""
	case v.GainedBasis():
		return " *(this metric gained or lost the data it reads, rather than improving or worsening)*"
	case v.Confidence == "":
		return " *(the new read states no confidence)*"
	case v.Weak():
		return " *(on " + v.Confidence + " confidence — likely noise in a thin window)*"
	default:
		return " *(" + v.Confidence + " confidence)*"
	}
}

// unevaluatedSection names metrics that did not run, so their absence is never read as a
// verdict that changed. The cron recipe this feature ships redirects stdout only, so a
// warning on stderr would be discarded exactly when it matters.
func (d *Digest) unevaluatedSection() string {
	if len(d.Unevaluated) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("### Not evaluated this run\n\n")
	for _, name := range d.Unevaluated {
		fmt.Fprintf(&b, "- `%s` did not run, so it has no verdict here — this is a missing metric, not a changed one.\n", name)
	}
	b.WriteString("\n")
	return b.String()
}

func moverSection(title string, movers []Mover) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### %s\n\n", title)
	moved := make([]Mover, 0, len(movers))
	for _, m := range movers {
		if m.Delta() != 0 {
			moved = append(moved, m)
		}
	}
	if len(moved) == 0 {
		b.WriteString("Nothing moved.\n\n")
		return b.String()
	}
	b.WriteString("| | previous | now | change |\n|---|---:|---:|---:|\n")
	for i, m := range moved {
		if i == moversShown {
			fmt.Fprintf(&b, "\n%s more moved, not shown.\n", humanize.Int(int64(len(moved)-moversShown)))
			break
		}
		fmt.Fprintf(&b, "| %s%s | %s | %s | %s |\n", m.Name, mark(m),
			humanize.Count(m.Was), humanize.Count(m.Now), relative(m.Was, m.Now))
	}
	b.WriteString("\n")
	return b.String()
}

// renamed applies the render-time pseudonym, so a project name is hidden in the output
// without the stored comparison key depending on whether it was hidden.
func (d *Digest) renamed(movers []Mover) []Mover {
	if d.Pseudonym == nil {
		return movers
	}
	out := make([]Mover, len(movers))
	copy(out, movers)
	for i := range out {
		out[i].Name = d.Pseudonym(out[i].Name)
	}
	return out
}

// mark names the two movements a percentage cannot describe: something that was not there
// before, and something that stopped appearing entirely.
func mark(m Mover) string {
	switch {
	case m.Appeared:
		return " *(new)*"
	case m.Vanished:
		return " *(gone)*"
	default:
		return ""
	}
}

func (d *Digest) caveatSection() string {
	var b strings.Builder
	b.WriteString("## Read this with\n\n")
	for _, c := range d.Caveats {
		fmt.Fprintf(&b, "- %s\n", c)
	}
	b.WriteString("- Cost is an estimate at public pay-as-you-go API prices — not actual spend; " +
		"a subscription bills a flat rate and differs.\n")
	return b.String()
}

// relative renders a movement as both its direction and its size. A previous value of zero
// has no percentage -- "up from nothing" is the honest phrasing, not an infinite rise.
func relative(was, now int64) string {
	switch {
	case was == 0 && now == 0:
		return "—"
	case was == 0:
		return "new"
	case now == 0:
		return "gone"
	}
	return fmt.Sprintf("%+.0f%%", 100*float64(now-was)/float64(was))
}

func relativeFloat(was, now float64) string {
	if was == 0 {
		if now == 0 {
			return "—"
		}
		return "new"
	}
	return fmt.Sprintf("%+.0f%%", 100*(now-was)/was)
}

func money(v *float64) string {
	if v == nil {
		return "—"
	}
	return humanize.USDCompact(*v)
}
