package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/assaio/assaio/internal/usage"
)

// RepositorySalt returns the salt this store's repository keys are derived with.
func (s *Store) RepositorySalt(ctx context.Context) ([]byte, error) {
	var salt []byte
	err := s.db.QueryRowContext(ctx, `SELECT salt FROM repository_salt WHERE id = 1`).Scan(&salt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("store has no repository salt; migration 0014 did not run")
	}
	return salt, err
}

// repoRef is one (label, key) pair a record offers.
type repoRef struct{ label, key string }

// registerSQL adds a pair to the registry with the next ordinal its label has never used.
const registerSQL = `
        INSERT INTO repository (label, repo_key, ordinal)
        SELECT ?, ?, COALESCE(MAX(ordinal), 0) + 1 FROM repository WHERE label = ?
        ON CONFLICT (label, repo_key) DO NOTHING`

// registerRepositories registers, inside tx, every repository recs resolved to and returns the
// id of each. A record with no key -- no repository, a guessed directory, a plugin or synced
// record -- registers nothing and maps to 0.
func registerRepositories(ctx context.Context, tx *sql.Tx, recs []usage.Record) (map[repoRef]int64, error) {
	ids := map[repoRef]int64{}
	for i := range recs {
		ref := repoRef{recs[i].Project, recs[i].RepoKey}
		if ref.label == "" || ref.key == "" {
			continue
		}
		if _, ok := ids[ref]; ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, registerSQL, ref.label, ref.key, ref.label); err != nil {
			return nil, fmt.Errorf("register repository: %w", err)
		}
		var id int64
		if err := tx.QueryRowContext(ctx,
			`SELECT id FROM repository WHERE label = ? AND repo_key = ?`, ref.label, ref.key).Scan(&id); err != nil {
			return nil, err
		}
		ids[ref] = id
	}
	return ids, nil
}

// repoID is the registered id of r's repository, 0 when it offers none.
func repoID(ids map[repoRef]int64, r *usage.Record) int64 {
	return ids[repoRef{r.Project, r.RepoKey}]
}

// forgetRepositories empties the registry and draws a new salt, so a store cleared of every
// usage row keeps no repository name and no keyed hash of a path either.
func forgetRepositories(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM repository`); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `UPDATE repository_salt SET salt = randomblob(32) WHERE id = 1`)
	return err
}
