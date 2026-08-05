package analyze

import (
	"sort"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

// sessionsAnswering keeps the sessions whose source can answer signal id, and reports what
// share of the window's sessions they are. A source that never records a thing contributes
// no zero to it: "no edit was made" and "this source records no edits" are different facts,
// and a metric that averages them reports the first while measuring the second.
func sessionsAnswering(sessions []store.SessionRow, id string) (kept []store.SessionRow, share float64) {
	kept = report.SessionsAnswering(sessions, id)
	return kept, shareOf(int64(len(kept)), int64(len(sessions)))
}

// sourcesAnswering names every in-tree source that answers any of ids, alphabetically. It is
// what a coverage caveat prints, read from the matrix rather than spelled out in prose a new
// parser makes wrong.
func sourcesAnswering(ids ...string) []string {
	var out []string
	for _, d := range parser.Depths() {
		if answersAny(d.Tool, ids) {
			out = append(out, d.Tool)
		}
	}
	sort.Strings(out)
	return out
}

func answersAny(tool string, ids []string) bool {
	for _, id := range ids {
		if parser.Answers(tool, id) {
			return true
		}
	}
	return false
}
