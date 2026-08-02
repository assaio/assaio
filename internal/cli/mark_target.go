package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/projectid"
	"github.com/assaio/assaio/internal/store"
)

// resolveMarkTarget picks the session to act on: the named prefix, the most recent session
// anywhere, or -- the default -- the most recent one in the repository holding the working
// directory. The project name is derived exactly as ingest derives it, so what is typed here
// and what was stored cannot disagree.
func resolveMarkTarget(cmd *cobra.Command, st *store.Store, args []string, last bool) (store.SessionRef, error) {
	if len(args) == 1 && last {
		return store.SessionRef{}, errors.New("--last targets the newest session and cannot be combined with a session id")
	}
	if len(args) == 1 {
		return resolveSessionPrefix(cmd, st, args[0])
	}
	project := ""
	if !last {
		cwd, err := os.Getwd()
		if err != nil {
			return store.SessionRef{}, err
		}
		root, _ := projectid.Resolve(cwd)
		project = filepath.Base(root)
	}
	ref, ok, err := st.LatestSession(cmd.Context(), project)
	if err != nil {
		return store.SessionRef{}, err
	}
	if !ok && project != "" {
		return store.SessionRef{}, fmt.Errorf("no stored session for project %q -- pass --last, a session id, or run 'mark --list'", project)
	}
	if !ok {
		return store.SessionRef{}, errors.New(emptyStoreHint(cmd, "No session to mark."))
	}
	return ref, nil
}

// resolveSessionPrefix resolves a session id prefix the way git resolves a short revision:
// an ambiguous prefix is reported with its candidates rather than silently resolved.
func resolveSessionPrefix(cmd *cobra.Command, st *store.Store, prefix string) (store.SessionRef, error) {
	matches, err := st.MatchSessions(cmd.Context(), prefix)
	if err != nil {
		return store.SessionRef{}, err
	}
	switch len(matches) {
	case 0:
		return store.SessionRef{}, fmt.Errorf("no stored session starts with %q (see 'mark --list')", prefix)
	case 1:
		return matches[0], nil
	default:
		ids := make([]string, 0, len(matches))
		for _, m := range matches {
			ids = append(ids, shortSessionID(m.SessionID))
		}
		return store.SessionRef{}, fmt.Errorf("%q matches %d sessions: %s", prefix, len(matches), strings.Join(ids, ", "))
	}
}
