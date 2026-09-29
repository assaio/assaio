package ingest

import (
	"os"
	"path/filepath"

	"github.com/assaio/assaio/internal/projectid"
	"github.com/assaio/assaio/internal/usage"
)

// resolution is one cached projectid.Resolve outcome. guessed means the working directory no
// longer exists: whatever Resolve finds now was computed with less of the filesystem than a
// read made while it existed. key is the repository's local identity, "" when no repository was
// found or the directory was gone -- a root found by walking up from a missing directory is a
// guess, and a guessed identity would join a stranger's usage to a repository.
type resolution struct {
	root, subpath, key string
	guessed            bool
}

// projectCache memoizes projectid.Resolve by working directory. Many records — every
// turn in a session, every session under one repo — share a Cwd, so caching turns a
// 100k-record backfill into at most one filesystem walk per distinct working directory.
// salt keys every repository identity it derives (store.RepositorySalt).
type projectCache struct {
	salt  []byte
	byCwd map[string]resolution
}

func newProjectCache(salt []byte) *projectCache {
	return &projectCache{salt: salt, byCwd: make(map[string]resolution)}
}

// resolveProjects fills Project, Subpath and RepoKey on every record with a non-empty Cwd, by
// resolving Cwd to its git repository root (internal/projectid). A record whose Cwd is empty
// keeps whatever Project its parser set; one outside any repository is named after its own
// directory and has no key. resolveProjects never errors. Cwd is never copied onto the stored
// record — it is not a persisted field, see usage.Record.Cwd.
func resolveProjects(recs []usage.Record, cache *projectCache) {
	for i := range recs {
		r := &recs[i]
		if r.Cwd == "" {
			continue
		}
		res := cache.resolve(r.Cwd)
		if res.root == "" {
			continue
		}
		r.Project = filepath.Base(res.root)
		r.Subpath = res.subpath
		r.RepoKey = res.key
		r.ProjectGuessed = res.guessed
	}
}

// resolve returns projectid.Resolve(cwd) with its key, computing and caching it on first use.
func (c *projectCache) resolve(cwd string) resolution {
	if res, ok := c.byCwd[cwd]; ok {
		return res
	}
	root, subpath, found := projectid.Resolve(cwd)
	res := resolution{root: root, subpath: subpath, guessed: !exists(cwd)}
	if found && !res.guessed {
		res.key = projectid.Key(c.salt, root)
	}
	c.byCwd[cwd] = res
	return res
}

// exists reports whether path can be stat'ed. A path that cannot be is treated as gone: its
// repository cannot be verified either way.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
