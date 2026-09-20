package share

import (
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/store"
)

// Basis is what the caller knows that the window does not: whether this is the bundled sample,
// the moment by which every token- or line-counting source in the window had last been read by
// an ingest run (zero when the caller could not say), and whether one of those sources stopped
// being found -- its latest retained run discovered no input where an earlier one did, so its
// rows end where the reading ended, not where the work did.
type Basis struct {
	Sample         bool
	ReadThrough    time.Time
	StoppedReading bool
}

// Trend is the week-over-week block: two moves quoted from the figures analyze published, each on
// its own layer and never merged into one claim. Withheld replaces both when the card itself
// cannot carry a movement, and Title and Clause travel with every rendering of it because the
// image leaves without the page.
type Trend struct {
	Title    string
	Clause   string
	Tokens   Move
	Lines    Move
	Withheld string
}

// Move is one quoted trend figure: Value is the published change or "—", Note the published
// dates, sums and reason, Layer the measurement layer of the validator that published it.
type Move struct {
	Label string
	Value string
	Note  string
	Layer string
}

const (
	trendTitle  = "week over week"
	trendClause = "a change in volume, not cost, value, or efficiency"
)

// buildTrend quotes the two figures as analyze published them. The validators already refuse a
// direction the store cannot support; on top of that the card omits one it cannot vouch for
// in public, because a shared image cannot be recalled and its caption does not travel with it.
func buildTrend(in *analyze.Input, v verdicts, basis Basis) Trend {
	t := Trend{
		Title:  trendTitle,
		Clause: trendClause,
		Tokens: quoteMove(v, "burn-anomaly", "week-over-week tokens", "tokens"),
		Lines:  quoteMove(v, "throughput", "week-over-week AI lines", "AI lines"),
	}
	if why := withholdTrend(in, basis); why != "" {
		t.Withheld = why
		t.Tokens.Value, t.Tokens.Note = "—", ""
		t.Lines.Value, t.Lines.Note = "—", ""
	}
	return t
}

func quoteMove(v verdicts, validator, label, name string) Move {
	m := Move{Label: name, Value: "—", Note: "not measured in this window"}
	if r, ok := v[validator]; ok {
		m.Layer = string(r.Layer)
	}
	if f, ok := v.figure(validator, label); ok {
		m.Value, m.Note = f.Value, f.Note
	}
	return m
}

// withholdTrend names why the card carries no movement even where analyze published one. Each is
// a fact the private surfaces state beside the figure and a public artifact cannot: unknown is
// not unchanged, so a basis the card cannot check is a basis it does not publish on.
func withholdTrend(in *analyze.Input, basis Basis) string {
	switch {
	case basis.Sample:
		return "sample data cannot support a direction"
	case in.HistoryStart.IsZero():
		return "store history could not be read"
	case in.ParsedBy == "":
		return "parser build for this data is not recorded"
	case !store.ReadByOneBuild(in.ParsedBy):
		return "more than one parser build read this data; a change may be a correction"
	case basis.ReadThrough.IsZero():
		return "time of the last store read is unknown"
	case basis.ReadThrough.Before(analyze.TrendCloses(in)):
		return "at least one source was last read " + basis.ReadThrough.UTC().Format("Jan 2") + ", before the recent week ended"
	case basis.StoppedReading:
		return "a source stopped being found; its rows end where the reading ended"
	default:
		return ""
	}
}

// readable reports whether at least one move states a direction, which is what earns the block a
// line in the post: a post of two dashes says nothing the card does not say better.
func (t *Trend) readable() bool {
	return t.Withheld == "" && (t.Tokens.Value != "—" || t.Lines.Value != "—")
}
