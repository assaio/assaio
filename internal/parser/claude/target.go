package claude

import (
	gopath "path"
	"strings"
)

// What file a step named, reduced to an identity two steps can be compared on. It is a pure
// function of two strings and holds none of the recorder's state, which is why it sits apart
// from it: the cross-platform reasoning below -- UNC shares, drive letters, and why a slash
// path is cleaned with gopath rather than filepath -- is the whole of this file's subject and
// none of the recorder's.

// targetKey folds a call's path and the session's cwd into the one string this parse uses as an
// identity, "" when it cannot.
//
// Deliberately not path/filepath: those answer "is this rooted" for the platform *assaio* runs on,
// while a transcript's paths come from wherever the *agent* ran. A POSIX path read on Windows was
// taken for relative and joined onto the cwd, so one file took two integers -- and a store synced
// from a mixed team holds both spellings in one table, where the host's opinion is meaningless.
// Nothing here ever reaches the filesystem: the result is a map key and is discarded with the
// parse, so it must be deterministic rather than canonical.
func targetKey(target, cwd string) string {
	target = strings.ReplaceAll(target, "\\", "/")
	if target == "" {
		return ""
	}
	if !rooted(target) {
		cwd = strings.ReplaceAll(cwd, "\\", "/")
		if !rooted(cwd) {
			return ""
		}
		target = cwd + "/" + target
	}
	// Clean folds a leading "//" to one slash, which would let a UNC share and a POSIX path of the
	// same name share an integer and read as one file revisited. Cheaper to keep the prefix than to
	// reason about how unlikely that is.
	if strings.HasPrefix(target, "//") {
		return "//" + strings.TrimPrefix(gopath.Clean(target), "/")
	}
	return gopath.Clean(target)
}

// rooted reports whether a slash-normalised path is absolute in either platform's spelling: a POSIX
// root, a UNC share (which starts with one too), or a drive letter.
func rooted(p string) bool {
	if strings.HasPrefix(p, "/") {
		return true
	}
	return len(p) >= 3 && p[1] == ':' && p[2] == '/' &&
		(p[0] >= 'A' && p[0] <= 'Z' || p[0] >= 'a' && p[0] <= 'z')
}
