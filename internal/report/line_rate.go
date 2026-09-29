package report

import "github.com/assaio/assaio/internal/parser"

// LineRateBasis is the population a cost-per-line figure divides: priced usage from sources
// that record added lines. A cost from a source that records no line has no line to be the
// cost of, and a line whose usage has no price has no cost to set against it, so both sides of
// the ratio come from these rows only and everything else is counted beside them (ADR 0011).
type LineRateBasis struct {
	// Cost, Lines and Tokens are the priced line-recording rows'.
	Cost   float64
	Lines  int64
	Tokens int64
	// LineRows counts every row from a source that records added lines, priced or not, and
	// LineBlindRows every other row.
	LineRows, LineBlindRows int
	// LineBlindTokens are rows from sources that record no added line, and LineBlindCost the
	// priced part of their cost.
	LineBlindTokens int64
	LineBlindCost   float64
	// UnpricedLineTokens and UnpricedLines are line-recording rows whose model has no known
	// price; their lines are left out of the ratio with their tokens.
	UnpricedLineTokens int64
	UnpricedLines      int64
	// SharedTokens and SharedLines are rows of a session-credited source from a group that ran
	// several models (LineCredit), left out whole when its lines cannot be paired with its cost:
	// in a basis that is one model's share, or when one of its models has no known price.
	SharedTokens int64
	SharedLines  int64
}

// add folds one row into b; cost and priced are the row's price lookup. Callers go through
// LineCredit.Fold, which settles a session-credited row's price before it lands here.
func (b *LineRateBasis) add(tool string, lines, tokens int64, cost float64, priced bool) {
	switch {
	case !parser.RecordsLines(tool):
		b.LineBlindRows++
		b.LineBlindTokens += tokens
		if priced {
			b.LineBlindCost += cost
		}
	case !priced:
		b.LineRows++
		b.UnpricedLineTokens += tokens
		b.UnpricedLines += lines
	default:
		b.LineRows++
		b.Cost += cost
		b.Lines += lines
		b.Tokens += tokens
	}
}

// Merge folds o into b.
func (b *LineRateBasis) Merge(o *LineRateBasis) {
	b.Cost += o.Cost
	b.Lines += o.Lines
	b.Tokens += o.Tokens
	b.LineRows += o.LineRows
	b.LineBlindRows += o.LineBlindRows
	b.LineBlindTokens += o.LineBlindTokens
	b.LineBlindCost += o.LineBlindCost
	b.UnpricedLineTokens += o.UnpricedLineTokens
	b.UnpricedLines += o.UnpricedLines
	b.SharedTokens += o.SharedTokens
	b.SharedLines += o.SharedLines
}

// Per100 is the basis's cost per 100 added lines; nil when it holds no line or no token, so a
// priced $0 over line-bearing rows that spent nothing never reads as a rate.
func (b *LineRateBasis) Per100() *float64 {
	if b.Lines == 0 || b.Tokens == 0 {
		return nil
	}
	r := b.Cost / (float64(b.Lines) / 100)
	return &r
}

// LeavesOut reports whether the population holds usage the ratio does not cover.
func (b *LineRateBasis) LeavesOut() bool {
	return b.LineBlindTokens > 0 || b.UnpricedLineTokens > 0 || b.SharedTokens > 0
}

// Partial reports whether there is a ratio and it leaves usage out -- the ratio a "†" marks.
func (b *LineRateBasis) Partial() bool { return b.Per100() != nil && b.LeavesOut() }

// LineCost is the numerator Per100 divides, nil when there is no rate to divide it into.
func (b *LineRateBasis) LineCost() *float64 {
	if b.Per100() == nil {
		return nil
	}
	c := b.Cost
	return &c
}

// OwnTokens and OwnLines are the line-recording usage whose lines are its own, priced or not:
// the population a rate that needs no price divides, such as lines per token or a line share.
func (b *LineRateBasis) OwnTokens() int64 { return b.Tokens + b.UnpricedLineTokens }

// OwnLines is OwnTokens' line count.
func (b *LineRateBasis) OwnLines() int64 { return b.Lines + b.UnpricedLines }
