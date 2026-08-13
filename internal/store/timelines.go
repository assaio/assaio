package store

import (
	"context"
	"time"
)

// TimelineStep is one step as a reader of the sequence needs it. The identity columns sit on the
// Timeline holding it: a sequence of two thousand steps carries its session id once rather than
// two thousand times.
type TimelineStep struct {
	Ordinal   int64
	Timestamp time.Time
	Kind      string
	// Outcome is "" when the source did not say, which is a different fact from ok (usage.Step).
	Outcome string
	Model   string
	Tokens  int64
	// TargetRef is the integer standing for the file this step named, 0 for none. Comparable
	// only within one Timeline -- see usage.Step.TargetRef for why it is not a digest.
	TargetRef int64
}

// Timeline is one sequence: a session's main loop, or one sub-agent inside it.
//
// Entrypoint and Project come from the session's usage records rather than from its steps,
// because both are properties of the session and storing them per step would let the two
// disagree. Measured on the maintainer's store, no session carries more than one entrypoint, and
// sessions.go states the same of project, so the MAX each is read with resolves nothing in
// practice and is there to keep the query strict-SQL. A session with no record in the window
// leaves both "" rather than being guessed at. That is a scope of its own (trace.Unstated) rather
// than a guess -- though the reader every surface uses narrows to the sessions the rest of the
// Input describes (trace.Set.ForSessions), so in practice such a sequence is dropped there rather
// than reaching a detector unattributed. The empty value is what makes it droppable on purpose
// instead of being folded into a population nothing measured it into.
//
// A non-empty Timeline is a sub-agent's sequence: every one of the 2,468 on that store belongs
// to a session that also holds a sidechain record. The converse does not hold -- 5 sessions
// carry a sidechain record and no sequence of their own, since a sub-agent contributes one only
// when its transcript was read -- so this identifies a sub-agent's *sequence*, not every
// sub-agent that ran.
type Timeline struct {
	Tool      string
	SessionID string
	Member    string
	Timeline  string
	// Entrypoint is how the session was started ("cli", "sdk-py", ...), "" when unknown.
	Entrypoint string
	// Project is the repository the session ran in, "" when unknown.
	Project string
	Steps   []TimelineStep
}

// SubAgent reports whether this sequence is a sub-agent's rather than a session's main loop.
func (t *Timeline) SubAgent() bool { return t.Timeline != "" }

const (
	timelinesQuery = `
        SELECT tool, session_id, member, timeline,
               ordinal, ts, kind, outcome, model, tokens, target_ref
        FROM session_step
        WHERE ts >= ?
        ORDER BY tool, session_id, member, timeline, ordinal`

	// sessionScopeQuery is deliberately a second round trip rather than a join onto the query
	// above: joining the aggregate onto every step row cost 1.05s against 0.31s for the plain
	// scan on a 340,000-step store, to attach two values that repeat for a whole sequence.
	sessionScopeQuery = `
        SELECT session_id, member, MAX(entrypoint), MAX(project)
        FROM usage_record
        WHERE ts >= ?
        GROUP BY session_id, member`
)

// sessionScope is the per-session facts a step row does not carry, keyed the way every read
// correlates: a session id is unique per member, never on its own.
type sessionScope struct{ entrypoint, project string }

type sessionKey struct{ session, member string }

// Timelines returns the stored sequences holding a step inside the window, each ordered by
// ordinal. A returned sequence may legitimately start above ordinal 1 and may be missing steps a
// detector would expect between two it holds: the retention horizon cuts the opening off a
// sequence straddling it, and so does a window narrower than the sequence. A reader allows for
// that rather than reading it as loss.
func (s *Store) Timelines(ctx context.Context, since time.Time) ([]Timeline, error) {
	sinceStr := since.UTC().Format(time.RFC3339)
	scopes, err := s.sessionScopes(ctx, sinceStr)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, timelinesQuery, sinceStr)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []Timeline
	// The kind, outcome and model vocabularies are closed and tiny, and the driver hands back a
	// fresh string per row: interning them turns 340,000 copies of "assistant" into one.
	seen := map[string]string{}
	for rows.Next() {
		var t Timeline
		var st TimelineStep
		var ts string
		if err := rows.Scan(&t.Tool, &t.SessionID, &t.Member, &t.Timeline,
			&st.Ordinal, &ts, &st.Kind, &st.Outcome, &st.Model, &st.Tokens, &st.TargetRef); err != nil {
			return nil, err
		}
		st.Timestamp = parseStoredTime(&ts)
		st.Kind, st.Outcome, st.Model = intern(seen, st.Kind), intern(seen, st.Outcome), intern(seen, st.Model)
		if n := len(out); n > 0 && sameSequence(&out[n-1], &t) {
			out[n-1].Steps = append(out[n-1].Steps, st)
			continue
		}
		scope := scopes[sessionKey{t.SessionID, t.Member}]
		t.Entrypoint, t.Project = scope.entrypoint, scope.project
		t.Steps = []TimelineStep{st}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) sessionScopes(ctx context.Context, since string) (map[sessionKey]sessionScope, error) {
	rows, err := s.db.QueryContext(ctx, sessionScopeQuery, since)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := map[sessionKey]sessionScope{}
	for rows.Next() {
		var k sessionKey
		var v sessionScope
		if err := rows.Scan(&k.session, &k.member, &v.entrypoint, &v.project); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// sameSequence reports whether a scanned row belongs to the sequence already being built. The
// query orders by exactly these four columns, so one comparison against the last sequence
// replaces a map of every sequence seen.
func sameSequence(built, row *Timeline) bool {
	return built.Tool == row.Tool && built.SessionID == row.SessionID &&
		built.Member == row.Member && built.Timeline == row.Timeline
}

func intern(seen map[string]string, s string) string {
	if kept, ok := seen[s]; ok {
		return kept
	}
	seen[s] = s
	return s
}
