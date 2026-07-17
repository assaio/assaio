package ingest

import (
	"path/filepath"

	"github.com/assaio/assaio/internal/projectid"
	"github.com/assaio/assaio/internal/usage"
)

// resolution is one cached projectid.Resolve outcome.
type resolution struct{ root, subpath string }

// projectCache memoizes projectid.Resolve by working directory. Many records — every
// turn in a session, every session under one repo — share a Cwd, so caching turns a
// 100k-record backfill into at most one filesystem walk per distinct working directory.
type projectCache map[string]resolution

// resolveProjects fills Project and Subpath on every record with a non-empty Cwd, by
// resolving Cwd to its git repository root (internal/projectid). A record whose Cwd is
// empty, or whose Cwd resolves to no repository root, keeps whatever Project its parser
// already set as a fallback. resolveProjects never errors. Cwd is never copied onto the
// stored record — it is not a persisted field, see usage.Record.Cwd.
func resolveProjects(recs []usage.Record, cache projectCache) {
	for i := range recs {
		r := &recs[i]
		if r.Cwd == "" {
			continue
		}
		res := cachedResolve(r.Cwd, cache)
		if res.root == "" {
			continue
		}
		r.Project = filepath.Base(res.root)
		r.Subpath = res.subpath
	}
}

// cachedResolve returns projectid.Resolve(cwd), computing and caching it on first use.
func cachedResolve(cwd string, cache projectCache) resolution {
	if res, ok := cache[cwd]; ok {
		return res
	}
	root, subpath := projectid.Resolve(cwd)
	res := resolution{root, subpath}
	cache[cwd] = res
	return res
}
