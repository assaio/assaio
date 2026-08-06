package analyze

import (
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

// sessionsAnswering keeps the sessions whose source can answer every id, and reports what
// share of the window's sessions they are. A source that never records a thing contributes
// no zero to it: "no edit was made" and "this source records no edits" are different facts,
// and a metric that averages them reports the first while measuring the second.
//
// Several ids are an AND, because a figure reading two fields is only honest where both
// were recorded -- a one-shot rate divides edits by turns, and a source answering the first
// but not the second would make every session read as landing in zero turns.
func sessionsAnswering(sessions []store.SessionRow, ids ...string) (kept []store.SessionRow, share float64) {
	kept = sessions
	for _, id := range ids {
		kept = report.SessionsAnswering(kept, id)
	}
	return kept, shareOf(int64(len(kept)), int64(len(sessions)))
}

// sourcesAnswering names the sources a coverage caveat credits, read from the matrix rather
// than spelled out in prose a new parser makes wrong.
func sourcesAnswering(ids ...string) []string { return parser.SourcesAnswering(ids...) }
