package report

import (
	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/store"
)

// ChurnStat is the aggregate rework/thrash signal across a set of usage rows: how much
// AI-added code got removed again within the same transcript it was added in -- the
// honest local proxy for "AI wrote code that didn't stick" (see usage.Record.ReworkLines).
type ChurnStat struct {
	// LinesAdded is total AI-added code lines.
	LinesAdded int64
	// ReworkLines is the subset of LinesAdded later removed within the same transcript+file.
	// Per transcript it can never exceed that file's additions (internal/parser.Rework spends
	// a budget); across a window whose start cuts between an addition and its removal it can.
	ReworkLines int64
	// ReworkRate is ReworkLines / LinesAdded; 0 when LinesAdded is zero, never a
	// divide-by-zero panic. That 0 is a placeholder for an undefined ratio, not a
	// measured rate -- a renderer must check LinesAdded itself (e.g. via
	// humanize.PercentOrDash on the raw counts) and CutsASession before formatting this as
	// a confident percentage; see internal/analyze/rework.go's "rework" Figure.
	ReworkRate float64
	// Rows and Tokens are what the rate rests on: how many usage rows came from a source
	// that records an undone line at all, and their tokens. Zero rows means the window
	// could not answer the question, which is a different fact from a window with no churn.
	Rows   int
	Tokens int64
}

// BuildChurn aggregates rework signals across the rows whose source records one. The gate
// is inside rather than at each call site, because `status` and the `rework` validator both
// print this number and a filter applied in one of them would be two answers to the same
// question (ADR 0011). Pure and empty-safe: no capable rows yields a zero-value ChurnStat.
func BuildChurn(rows []store.UsageRow) ChurnStat {
	capable := UsageAnswering(rows, parser.SignalReworkLines)
	s := ChurnStat{Rows: len(capable), Tokens: TokensIn(capable)}
	for i := range capable {
		s.LinesAdded += capable[i].LinesAdded
		s.ReworkLines += capable[i].ReworkLines
	}
	s.ReworkRate = reworkRate(s.ReworkLines, s.LinesAdded)
	return s
}

// ExceedsItsWhole reports that the removals outnumber the additions this window counted, so the
// ratio is not a share of anything in it. The usual cause is a window opening between an
// addition and its removal, but the counts cannot show that, so the name states what they do
// show. Derived rather than stored, so a ChurnStat built anywhere answers from its own counts.
func (c *ChurnStat) ExceedsItsWhole() bool { return c.ReworkLines > c.LinesAdded }

// reworkRate is rework/added, 0 when added is zero -- never a divide-by-zero.
func reworkRate(rework, added int64) float64 {
	if added == 0 {
		return 0
	}
	return float64(rework) / float64(added)
}

// ChurnBoundaryNote states why an unmeasurable rate is withheld on a window that does record
// undone lines: the counts are real, the ratio between them is not.
func ChurnBoundaryNote(c *ChurnStat) string {
	return humanize.Int(c.ReworkLines) + " lines undone against " + humanize.Int(c.LinesAdded) +
		" added inside this window, so there is no share to state -- most often a window that opened between an addition and its removal"
}
