package store

import (
	"context"
	"time"
)

// SessionSignalRow is one distinct combination of the columns a label can be derived from,
// for one session. A session that ran on one branch and used three skills yields several
// rows; the caller collapses them per source, and a session carrying two different branches
// is exactly the ambiguity the derivation has to see rather than have hidden from it.
type SessionSignalRow struct {
	SessionID string
	// Member is carried because a session id is only unique per member: on a store that
	// collected more than one machine's rows, keying on the id alone would let one member's
	// branch derive the label written onto another member's session. Every other label path
	// correlates on (session_id, member) for the same reason.
	Member     string
	Branch     string
	Skill      string
	Agent      string
	Entrypoint string
}

// The query is spelled out in full for the reason attribution.go states.
const sessionSignalsQuery = `
        SELECT DISTINCT session_id, member, git_branch, skill, agent, entrypoint
        FROM usage_record
        WHERE ts >= ?
        ORDER BY session_id, member`

// SessionSignals returns what each session in the window recorded on the columns a label
// may be derived from. Nothing here is inferred: every value was written down by the tool
// that produced the session.
func (s *Store) SessionSignals(ctx context.Context, since time.Time) ([]SessionSignalRow, error) {
	rows, err := s.db.QueryContext(ctx, sessionSignalsQuery, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []SessionSignalRow
	for rows.Next() {
		var r SessionSignalRow
		if err := rows.Scan(&r.SessionID, &r.Member, &r.Branch, &r.Skill, &r.Agent, &r.Entrypoint); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
