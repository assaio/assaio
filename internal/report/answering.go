package report

import (
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/store"
)

// The capability filter, in one place for both row shapes. A figure averaged over a source
// that never records the thing reads that silence as a zero, which is the one mistake a
// summary can make that nobody can see afterwards (ADR 0011). Exported because the
// validators ask the same question, and two copies of this loop would be two answers to it.

// SessionsAnswering keeps the sessions whose source can answer signal id.
func SessionsAnswering(rows []store.SessionRow, id string) []store.SessionRow {
	out := make([]store.SessionRow, 0, len(rows))
	for i := range rows {
		if parser.Answers(rows[i].Tool, id) {
			out = append(out, rows[i])
		}
	}
	return out
}

// UsageAnswering keeps the usage rows whose source can answer signal id. A rate over a
// stored column needs this as much as a session figure does: a source recording changed
// lines but no rework contributes its lines to the denominator and a structural zero to
// the numerator, which lowers the rate with work nobody measured.
func UsageAnswering(rows []store.UsageRow, id string) []store.UsageRow {
	out := make([]store.UsageRow, 0, len(rows))
	for i := range rows {
		if parser.Answers(rows[i].Tool, id) {
			out = append(out, rows[i])
		}
	}
	return out
}

// RowTokens is a row's billable token total. Reasoning is a subset of output
// (usage.Record) and is never re-added, which is the reason this lives in one place: a
// second copy of the sum is a second answer to what a window cost.
func RowTokens(r *store.UsageRow) int64 {
	return r.In + r.Out + r.CacheRead + r.CacheWrite
}

// TokensIn totals RowTokens across rows, so a caller can state a subset's reach as a share
// of the window.
func TokensIn(rows []store.UsageRow) int64 {
	var sum int64
	for i := range rows {
		sum += RowTokens(&rows[i])
	}
	return sum
}
