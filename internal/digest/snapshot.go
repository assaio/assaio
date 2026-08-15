// Package digest reports what moved between one run and the next, rather than restating
// what is. It owns no measurement of its own: every figure here was produced by the same
// window the other surfaces read, and the only thing digest adds is the comparison -- and
// the honesty about when that comparison cannot be trusted.
package digest

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/report"

	"github.com/assaio/assaio/internal/analyze"
)

// SnapshotVersion guards the stored payload. A digest that cannot read the previous
// snapshot reports a first run rather than comparing against a shape it misunderstands.
const SnapshotVersion = 1

// Snapshot is what one digest recorded: the totals, the per-dimension weights it ranks
// movers by, and each validator's verdict. Deliberately no prose and no sample rows -- a
// snapshot is a comparison basis, not a second copy of the report.
type Snapshot struct {
	Version  int       `json:"version"`
	TakenAt  time.Time `json:"takenAt"`
	Window   string    `json:"window"`
	ParsedBy string    `json:"parsedBy"`
	Tokens   int64     `json:"tokens"`
	Cost     *float64  `json:"cost"`
	// Priced is false when the window holds any row on a model the price table does not
	// carry. UnpricedShare is the different and stricter question -- how much of the
	// window's tokens that leaves uncosted -- and it is the one a cost comparison turns on:
	// a row with no price and no tokens costs the total nothing.
	Priced        bool    `json:"priced"`
	UnpricedShare float64 `json:"unpricedShare"`
	// UnpricedNote is the disclosure the cost surfaces print, stored verbatim so the digest
	// quotes the same sentence rather than composing a second one around it.
	UnpricedNote string `json:"unpricedNote"`
	Lines        int64  `json:"lines"`
	// LinesCapable reports whether any source in the window records a changed line. False means
	// Lines is an absence, not a zero -- and this is the one surface written to be read out of
	// context, where the reader cannot check (ADR 0011).
	LinesCapable bool              `json:"linesCapable"`
	Sessions     int               `json:"sessions"`
	Models       map[string]int64  `json:"models"`
	Projects     map[string]int64  `json:"projects"`
	Verdicts     map[string]string `json:"verdicts"`
	// Confidence is each verdict's own label. A read that moved while resting on
	// insufficient data is noise in a thin window, and reporting it beside a read that
	// moved on high confidence would present the two as the same event.
	Confidence map[string]string `json:"confidence"`
	// Leads are the validators the window's own ranking put first. A digest leads with the
	// same findings `analyze` leads with, rather than inventing a second order of
	// importance for the same data.
	Leads []string `json:"leads"`
}

// Options is what the caller knows that the window does not. UnpricedNote is the disclosure
// the cost surfaces render, passed in rather than recomputed so every surface says the same
// sentence about the same gap.
type Options struct {
	Window        string
	UnpricedNote  string
	UnpricedShare float64
	At            time.Time
}

// Take reads the window every other surface reads. results are the validator Results the
// same run produced, so a digest never re-derives a verdict a different way than `analyze`
// would have.
func Take(in *analyze.Input, results []analyze.Result, opts Options) Snapshot {
	window, unpricedNote, at := opts.Window, opts.UnpricedNote, opts.At
	s := Snapshot{
		Version: SnapshotVersion, TakenAt: at, Window: window, ParsedBy: in.ParsedBy,
		Tokens: in.Totals.Tokens, Cost: in.Totals.Cost, Priced: in.Totals.Priced,
		UnpricedShare: opts.UnpricedShare, UnpricedNote: unpricedNote,
		Lines:        in.Totals.Lines,
		LinesCapable: report.AnySourceAnswers(in.Usage, parser.SignalLinesAdded),
		Sessions:     len(in.Sessions),
		Models:       make(map[string]int64, len(in.ByModel)),
		Projects:     make(map[string]int64, len(in.ByProject)),
		Verdicts:     make(map[string]string, len(results)),
		Confidence:   make(map[string]string, len(results)),
	}
	for _, m := range in.ByModel {
		s.Models[m.Model] = m.Tokens
	}
	for _, p := range in.ByProject {
		s.Projects[p.Project] = p.Lines
	}
	type ranked struct {
		name string
		rank int
	}
	leads := make([]ranked, 0, len(results))
	for i := range results {
		s.Verdicts[results[i].Name] = results[i].Read.Key
		s.Confidence[results[i].Name] = results[i].Confidence.Label
		if results[i].Lead != nil {
			leads = append(leads, ranked{results[i].Name, results[i].Lead.Rank})
		}
	}
	// Ordered by the rank MarkLead stamped, not by the order the results arrive in:
	// analyze.Validators() is name-sorted, so appending as they come would store an
	// alphabetical list and the digest would lead with a different finding than `analyze`.
	sort.Slice(leads, func(i, j int) bool { return leads[i].rank < leads[j].rank })
	for _, l := range leads {
		s.Leads = append(s.Leads, l.name)
	}
	return s
}

// LeadsFirst orders changes the way the window's own ranking did, so the digest and
// `analyze` lead with the same finding. Everything else keeps its alphabetical order.
func (s *Snapshot) LeadsFirst(changes []VerdictChange) []VerdictChange {
	rank := make(map[string]int, len(s.Leads))
	for i, name := range s.Leads {
		rank[name] = i + 1
	}
	out := append([]VerdictChange(nil), changes...)
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := rank[out[i].Name], rank[out[j].Name]
		switch {
		case ri != 0 && rj != 0:
			return ri < rj
		case ri != 0 || rj != 0:
			return ri != 0
		default:
			return false
		}
	})
	return out
}

// Marshal encodes the snapshot for storage as the next run's comparison basis.
func (s *Snapshot) Marshal() ([]byte, error) { return json.Marshal(s) }

// Parse decodes a stored snapshot, refusing one written by a different version rather than
// comparing against fields that may have moved.
func Parse(payload []byte) (Snapshot, bool) {
	var s Snapshot
	if err := json.Unmarshal(payload, &s); err != nil || s.Version != SnapshotVersion {
		return Snapshot{}, false
	}
	return s, true
}
