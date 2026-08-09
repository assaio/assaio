package calibration

import (
	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

// Window is the one figure every surface has to agree on for a given window: what the tokens
// and the AI lines were. It is deliberately small -- a surface disagreeing about the headline
// is the failure worth catching, and a wider comparison would mostly test that structs have
// the fields they have.
type Window struct {
	In, Out, CacheRead, CacheWrite int64
	LinesAdded, LinesRemoved       int64
}

// Tokens is the total a report header, a status line and a dashboard tile all print.
func (w *Window) Tokens() int64 { return w.In + w.Out + w.CacheRead + w.CacheWrite }

// FromRows is the window as the store hands it over -- the input every surface starts from,
// and the answer all of them are measured against.
func FromRows(rows []store.UsageRow) Window {
	var w Window
	for i := range rows {
		r := &rows[i]
		w.In += r.In
		w.Out += r.Out
		w.CacheRead += r.CacheRead
		w.CacheWrite += r.CacheWrite
		w.LinesAdded += r.LinesAdded
		w.LinesRemoved += r.LinesRemoved
	}
	return w
}

// FromReport is the window as `report` renders it, after grouping and pricing. A row is a
// group rather than a record, so this is the arithmetic the table, the JSON and the CSV all
// print from. It carries no line counts -- `report` is about what was spent.
func FromReport(rows []report.Row) Window {
	var w Window
	for i := range rows {
		r := &rows[i]
		w.In += r.In
		w.Out += r.Out
		w.CacheRead += r.CacheRead
		w.CacheWrite += r.CacheWrite
	}
	return w
}

// FromEffectiveness is the window as `effectiveness` renders it: the surface about what the
// spend produced, so it carries the line counts `report` does not.
func FromEffectiveness(rows []report.EffRow) Window {
	var w Window
	for i := range rows {
		w.LinesAdded += rows[i].LinesAdded
		w.LinesRemoved += rows[i].LinesRemoved
	}
	return w
}

// FromAnalyze is the window as the validators, the dashboard and every metric plugin see it:
// analyze.Input.Totals is what a plugin is handed on the wire and what a faceplate divides
// by. Totals.Lines is added lines only, so a removed-line total is not among its figures.
func FromAnalyze(in *analyze.Input) Window {
	return Window{
		In:         in.Totals.Input,
		Out:        in.Totals.Output,
		CacheRead:  in.Totals.CacheRead,
		CacheWrite: in.Totals.CacheWrite,
		LinesAdded: in.Totals.Lines,
	}
}

// The figures a surface can publish. A surface is compared only on the ones it actually
// renders: `report` prints tokens and cost and no line counts at all, so holding it to a
// line total would report a shape as a disagreement. What each surface claims to carry is
// named at its call site, which is also where a new column has to be added.
const (
	FigIn           = "in"
	FigOut          = "out"
	FigCacheRead    = "cache_read"
	FigCacheWrite   = "cache_write"
	FigTokens       = "tokens"
	FigLinesAdded   = "lines_added"
	FigLinesRemoved = "lines_removed"
)

// TokenFigures are the four classes plus their total -- what any surface printing a token
// count or a cost has to agree on.
var TokenFigures = []string{FigIn, FigOut, FigCacheRead, FigCacheWrite, FigTokens}

// LineFigures are the changed-line counts, published only by the surfaces about output.
var LineFigures = []string{FigLinesAdded, FigLinesRemoved}

// Disagreements names every named figure on which two windows differ, so a failure says
// which surface lost which number.
func Disagreements(a, b *Window, figures ...string) []string {
	values := func(w *Window) map[string]int64 {
		return map[string]int64{
			FigIn: w.In, FigOut: w.Out, FigCacheRead: w.CacheRead, FigCacheWrite: w.CacheWrite,
			FigTokens: w.Tokens(), FigLinesAdded: w.LinesAdded, FigLinesRemoved: w.LinesRemoved,
		}
	}
	av, bv := values(a), values(b)
	var out []string
	for _, f := range figures {
		if av[f] != bv[f] {
			out = append(out, f+" differs")
		}
	}
	return out
}
