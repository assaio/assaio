package store

import (
	"context"
	"sort"
)

// A usage row keeps the label it was resolved with in project and names the repository it came
// from in repo_id (migration 0014): 0 unresolved, -1 ambiguous, otherwise a repository row.
// Which name a reader sees is decided once, in shownNames.

// shownNames turns a stored (label, repo_id) into the name a reader sees, as CTEs a read joins
// through shownNameJoin; a pair it does not list is shown as its label. Only live repositories
// count -- some usage row still carries the id -- so one whose rows are all gone stops splitting
// its label. A label with one live repository is that label on every row, resolved or not: the
// store holds no evidence of a second one. A label with two or more is split: the first by
// ordinal keeps the bare label, so an established name never renames, the rest are "label (n)",
// and rows that cannot be placed are "label (?)". A generated name some row already carries as
// its own label becomes "label (#n)", so one name never stands for two repositories.
const shownNames = `
        repo_live(id, label, ordinal) AS (
            SELECT id, label, ordinal FROM repository
            WHERE EXISTS (SELECT 1 FROM usage_record live
                          WHERE live.repo_id > 0 AND live.repo_id = repository.id)
        ),
        repo_rank(id, label, rank, live) AS (
            SELECT id, label,
                   ROW_NUMBER() OVER (PARTITION BY label ORDER BY ordinal),
                   COUNT(*) OVER (PARTITION BY label)
            FROM repo_live
        ),
        repo_suffix(label, repo_id, suffix) AS (
            SELECT label, id, CAST(rank AS TEXT) FROM repo_rank WHERE rank > 1
            UNION ALL SELECT label, 0, '?' FROM repo_rank WHERE rank = 1 AND live > 1
            UNION ALL SELECT label, -1, '?' FROM repo_rank WHERE rank = 1 AND live > 1
        ),
        shown_name(label, repo_id, name) AS (
            SELECT label, repo_id,
                   CASE WHEN EXISTS (SELECT 1 FROM usage_record taken
                                     WHERE taken.project = label || ' (' || suffix || ')')
                        THEN label || ' (#' || suffix || ')'
                        ELSE label || ' (' || suffix || ')' END
            FROM repo_suffix
        )`

// shownNameJoin attaches shown_name to an unaliased usage_record; the name is then
// COALESCE(shown_name.name, project).
const shownNameJoin = `
        LEFT JOIN shown_name ON shown_name.label = usage_record.project
                            AND shown_name.repo_id = usage_record.repo_id`

// Session-level reads name a session from its rows the way Sessions always has -- no label when
// its rows disagree or a re-read found competing claims -- and from the one positive repo_id its
// rows carry. An unresolved row in a resolved session follows the session.
const (
	sessionLabelExpr = `CASE WHEN MAX(project_conflict) = 1 OR COUNT(DISTINCT NULLIF(project, '')) > 1
                    THEN '' ELSE MAX(project) END`
	sessionRepoExpr = `CASE WHEN COUNT(DISTINCT CASE WHEN repo_id > 0 THEN repo_id END) = 1
                    THEN MAX(repo_id) ELSE 0 END`
)

// SplitLabels returns, sorted, the labels two or more live repositories share -- the names
// every report shows split.
func (s *Store) SplitLabels(ctx context.Context) ([]string, error) {
	splits, err := s.SplitComposition(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(splits))
	for label := range splits {
		out = append(out, label)
	}
	sort.Strings(out)
	return out, nil
}

// SplitComposition maps each split label to the ids of its live repositories in ordinal order.
// Two runs that show the same labels split can still mean different repositories by them -- the
// first one's rows cleared, the second takes the bare name -- and this is what tells them apart.
func (s *Store) SplitComposition(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `WITH`+shownNames+`
        SELECT label, GROUP_CONCAT(id, ',') FROM (SELECT label, id FROM repo_live ORDER BY label, ordinal)
        GROUP BY label HAVING COUNT(*) > 1`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]string{}
	for rows.Next() {
		var label, ids string
		if err := rows.Scan(&label, &ids); err != nil {
			return nil, err
		}
		out[label] = ids
	}
	return out, rows.Err()
}
