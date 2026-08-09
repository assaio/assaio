package store

import "context"

// SupersededAggregates lists the stored sub-agent aggregate rows -- Claude Code's
// "agent:<id>" dedupe keys -- whose <id> is in covered, i.e. whose own transcript is now on
// disk and has been read in full. Suppressing the aggregate at parse time only keeps a *new*
// one out; a row written before that file existed, or by a build that could not discover it,
// stayed beside the detailed turns and counted the same work twice.
//
// The scan matches the shape internal/parser/claude writes locally. It is deliberately not
// widened to the "<member>:agent:<id>" form a central store holds: that store's rows belong
// to the member who pushed them, and only that member's own agent knows which of their
// sub-agent transcripts exist.
func (s *Store) SupersededAggregates(ctx context.Context, tool string, covered map[string]struct{}) ([]string, error) {
	if len(covered) == 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT dedupe_key FROM usage_record WHERE tool = ? AND dedupe_key LIKE 'agent:%'`, tool)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		if _, ok := covered[key[len("agent:"):]]; ok {
			out = append(out, key)
		}
	}
	return out, rows.Err()
}

// DeleteDedupeKeys removes tool's rows for the given dedupe keys and reports how many went.
// It is the only path that deletes a stored record outside `clear`, and it exists for one
// situation: a row a later read proves was a summary of usage the store now holds in full. A
// record that is merely re-read is restated, never replaced -- see InsertLocal.
func (s *Store) DeleteDedupeKeys(ctx context.Context, tool string, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, `DELETE FROM usage_record WHERE tool = ? AND dedupe_key = ?`)
	if err != nil {
		return 0, err
	}
	defer func() { _ = stmt.Close() }()
	var deleted int64
	for _, k := range keys {
		res, err := stmt.ExecContext(ctx, tool, k)
		if err != nil {
			return deleted, err
		}
		n, _ := res.RowsAffected()
		deleted += n
	}
	if err := tx.Commit(); err != nil {
		return deleted, err
	}
	return deleted, nil
}
