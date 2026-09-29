package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Repository is what the store holds for the repository at one directory. Label is the name its
// rows carry -- the registry's own when the key was stored under another spelling of the path.
// ID is its registered id while some row carries it, 0 when no stored usage resolved to it. Name
// is what a reader sees for its rows, "" when ID is 0. Unplaced is the name the label's
// unresolved rows are shown under, and Others counts the other live repositories holding the
// label.
type Repository struct {
	Label, Name, Unplaced string
	ID                    int64
	Others                int
}

// RepositoryAt returns what the store holds for the repository with label and key; key is ""
// for a directory outside any repository. The key is looked up first, so a directory reached
// through another spelling of its path finds the rows its canonical spelling stored.
func (s *Store) RepositoryAt(ctx context.Context, label, key string) (Repository, error) {
	r := Repository{Label: label}
	if key != "" {
		err := s.db.QueryRowContext(ctx, `WITH`+shownNames+`
            SELECT repo_live.id, repo_live.label FROM repo_live JOIN repository USING (id)
            WHERE repository.repo_key = ?
            ORDER BY repo_live.label = ? DESC, repo_live.ordinal
            LIMIT 1`, key, label).Scan(&r.ID, &r.Label)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return Repository{}, err
		}
	}
	if r.Label == "" {
		return Repository{}, nil
	}
	err := s.db.QueryRowContext(ctx, `WITH`+shownNames+`
        SELECT (SELECT COUNT(*) FROM repo_live WHERE label = ? AND id <> ?),
               COALESCE((SELECT name FROM shown_name WHERE label = ? AND repo_id = ?), ?),
               COALESCE((SELECT name FROM shown_name WHERE label = ? AND repo_id = 0), ?)`,
		r.Label, r.ID, r.Label, r.ID, r.Label, r.Label, r.Label).Scan(&r.Others, &r.Name, &r.Unplaced)
	if r.ID == 0 {
		r.Name = ""
	}
	return r, err
}

// RepositoryLines sums, over rows with timestamp >= since, the AI lines resolved to repository
// id and the lines under its label whose repository is unresolved or ambiguous. The second can
// be this repository's or another's; nothing stored says which.
func (s *Store) RepositoryLines(ctx context.Context, r Repository, since time.Time) (resolved, unplaced int64, err error) {
	err = s.db.QueryRowContext(ctx, `
        SELECT COALESCE(SUM(CASE WHEN ? > 0 AND repo_id = ? THEN lines_added END), 0),
               COALESCE(SUM(CASE WHEN repo_id <= 0 THEN lines_added END), 0)
        FROM usage_record
        WHERE project = ? AND ts >= ?`,
		r.ID, r.ID, r.Label, since.UTC().Format(time.RFC3339)).Scan(&resolved, &unplaced)
	return resolved, unplaced, err
}
