package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/assaio/assaio/internal/projectid"
	"github.com/assaio/assaio/internal/store"
)

// localRepository is the repository a directory belongs to, as the store sees it: whether a .git
// was found at all, and what st holds for it.
type localRepository struct {
	inRepo bool
	store.Repository
}

// repositoryAt resolves dir the way ingest resolves a session's working directory -- a worktree
// to its main checkout -- and returns what st holds for that repository. The commands that start
// from a directory (evidence, survival, mark, repos) share it so they cannot disagree with ingest,
// or with each other, about which stored rows are this repository's.
func repositoryAt(ctx context.Context, st *store.Store, dir string) (localRepository, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return localRepository{}, err
	}
	root, _, found := projectid.Resolve(abs)
	key := ""
	if found {
		salt, err := st.RepositorySalt(ctx)
		if err != nil {
			return localRepository{}, err
		}
		key = projectid.Key(salt, root)
	}
	rep, err := st.RepositoryAt(ctx, filepath.Base(root), key)
	return localRepository{inRepo: found, Repository: rep}, err
}

// unresolvedNote says why a repository has no stored usage of its own, for the commands that
// then withhold a figure rather than print another repository's.
func unresolvedNote(here *localRepository) string {
	if here.Others > 0 {
		return fmt.Sprintf("no stored usage resolved to this repository; another repository has usage "+
			"under %q ('assaio-agent repos' shows which directory is which)", here.Label)
	}
	return "no stored usage resolved to this repository"
}
