package analyze

import (
	"time"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/store"
)

// trendQty is what a week-over-week figure sums: its published label, the signal a source must
// answer to be counted at all, the quantity per row, and how the two sums print.
type trendQty struct {
	label  string
	unit   string
	signal string
	of     func(*store.UsageRow) int64
	count  func(int64) string
}

var (
	// linesTrend is published under one label by adoption and throughput, so the same signal
	// reads the same way in both.
	linesTrend = trendQty{
		label: "week-over-week AI lines", unit: "lines", signal: parser.SignalLinesAdded,
		of: func(r *store.UsageRow) int64 { return r.LinesAdded }, count: humanize.Int,
	}
	tokensTrend = trendQty{
		label: "week-over-week tokens", unit: "tokens", signal: parser.SignalTokensTotal,
		of: rowTokens, count: humanize.Count,
	}
)

// linesTrendFloor is the line count the busier span must reach before a direction is stated:
// 1 to 2 lines is a 100% swing on a trivial sample. The busier span rather than both, so output
// collapsing from thousands of lines to none stays readable.
const linesTrendFloor = 20

// trendGap is why a trend states no direction; trendRead means it states one.
type trendGap int

const (
	trendRead trendGap = iota
	gapWindow
	gapHistory
	gapNoSource
	gapZeroBase
	gapThin
)

// trend is one week-over-week comparison: the two sums over the sources already running when the
// earlier span began, or the gap that stops a direction being read.
type trend struct {
	recent, prior span
	now, was      int64
	gap           trendGap
	// excluded counts the sources with rows in either span that were left out of both sums, and
	// left is what those rows held across the two spans: a count alone cannot say whether the
	// sums cover most of the machine's volume or a sliver of it, and the sign can turn on that.
	excluded    int
	left        int64
	historyFrom time.Time
}

// leftShare is the share of the two spans' volume the sums leave out, 0..1.
func (t *trend) leftShare() float64 {
	if total := t.left + t.now + t.was; total > 0 {
		return float64(t.left) / float64(total)
	}
	return 0
}

// TrendCloses is the moment every week-over-week comparison of in is over: the midnight after the
// recent span's last day. A surface that must know whether the store was read after it asks here
// rather than re-deriving the spans.
func TrendCloses(in *Input) time.Time {
	recent, _ := trendSpans(in.Now, in.Recent)
	return recent.endsAt()
}

func (t *trend) readable() bool { return t.gap == trendRead }

// change is (now-was)/was as a fraction, and zero whenever the trend is not readable.
func (t *trend) change() float64 {
	if !t.readable() {
		return 0
	}
	return float64(t.now-t.was) / float64(t.was)
}

// readTrend compares in.Usage across the two spans. floor is the smallest sum the busier span
// must reach for a direction to mean anything.
func readTrend(in *Input, q trendQty, floor int64) trend {
	t := trend{}
	t.recent, t.prior = trendSpans(in.Now, in.Recent)
	// Usage is queried WHERE ts >= start, so a window opening after the earlier span began holds
	// only part of it, while the store-wide horizon below would still call it covered.
	if !in.WindowStart.IsZero() && in.WindowStart.After(t.prior.startsAt()) {
		t.gap = gapWindow
		return t
	}
	if covers, known := horizonCovers(in); known && !covers {
		t.gap, t.historyFrom = gapHistory, in.HistoryStart.UTC()
		return t
	}
	counted := t.sources(in, q)
	if len(counted) == 0 {
		t.gap = gapNoSource
		return t
	}
	for i := range in.Usage {
		r := &in.Usage[i]
		if !t.recent.holds(r.Day) && !t.prior.holds(r.Day) || !parser.Answers(r.Tool, q.signal) {
			continue
		}
		switch {
		case !counted[r.Tool]:
			t.left += q.of(r)
		case t.recent.holds(r.Day):
			t.now += q.of(r)
		default:
			t.was += q.of(r)
		}
	}
	switch {
	case t.was == 0:
		t.gap = gapZeroBase
	case max(t.now, t.was) < floor:
		t.gap = gapThin
	}
	return t
}

// sources picks what the two sums may count: the sources answering q's signal with rows in either
// span that this window already shows in use when the earlier span began -- a row of theirs
// falls on or before its first day. The evidence is the window's rows, not the source's history,
// so a source quiet on those days is left out as if new; the Note states the share of volume that
// leaves out, and a narrow --since leaves few days to show a source in use. A source that stopped
// still counts, because its silence is the store's to state. One first seen inside the comparison
// is left out of both sides and counted: its first week would otherwise read as growth in how
// much the tools were used, and a source idle until then is left out with it, which can understate
// a rise but never invent one.
func (t *trend) sources(in *Input, q trendQty) map[string]bool {
	running, inSpans := map[string]bool{}, map[string]bool{}
	for i := range in.Usage {
		r := &in.Usage[i]
		if !parser.Answers(r.Tool, q.signal) {
			continue
		}
		if r.Day <= t.prior.from {
			running[r.Tool] = true
		}
		if t.recent.holds(r.Day) || t.prior.holds(r.Day) {
			inSpans[r.Tool] = true
		}
	}
	counted := map[string]bool{}
	for tool := range inSpans {
		if running[tool] {
			counted[tool] = true
		} else {
			t.excluded++
		}
	}
	return counted
}

// trendPurity scores a week-over-week change 0..1 for the dashboard gauge: 0.5 (neutral) when
// the trend is unknown, saturating toward 1 at +100% or more and 0 at -100% or less.
func trendPurity(changePct float64, ok bool) float64 {
	if !ok {
		return 0.5
	}
	return clamp01(0.5 + changePct/2)
}
