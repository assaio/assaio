package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/store"
)

// resolveMarkTarget picks the session to act on: the named prefix, the most recent session
// anywhere, or -- the default -- the most recent one in the repository holding the working
// directory, found the way ingest resolves a session's repository (repositoryAt).
func resolveMarkTarget(cmd *cobra.Command, st *store.Store, args []string, last bool) (store.SessionRef, error) {
	if len(args) == 1 && last {
		return store.SessionRef{}, errors.New("--last targets the newest session and cannot be combined with a session id")
	}
	if len(args) == 1 {
		return resolveSessionPrefix(cmd, st, args[0])
	}
	if !last {
		return latestHere(cmd, st)
	}
	ref, ok, err := st.LatestSession(cmd.Context())
	if err != nil {
		return store.SessionRef{}, err
	}
	if !ok {
		return store.SessionRef{}, errors.New(emptyStoreHint(cmd, "No session to mark."))
	}
	return ref, nil
}

// latestHere is the newest session with usage resolved to the repository holding the working
// directory. A session that only shares the repository's name is never it.
func latestHere(cmd *cobra.Command, st *store.Store) (store.SessionRef, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return store.SessionRef{}, err
	}
	here, err := repositoryAt(cmd.Context(), st, cwd)
	if err != nil {
		return store.SessionRef{}, err
	}
	if here.ID == 0 {
		if n, err := st.Count(cmd.Context()); err != nil || n == 0 {
			return store.SessionRef{}, errors.Join(err, errors.New(emptyStoreHint(cmd, "No session to mark.")))
		}
		return store.SessionRef{}, fmt.Errorf("%s -- pass --last, a session id, or run 'mark --list'", unresolvedNote(&here))
	}
	ref, ok, err := st.LatestSessionIn(cmd.Context(), here.ID)
	if err != nil {
		return store.SessionRef{}, err
	}
	if !ok {
		return store.SessionRef{}, fmt.Errorf("no stored session for project %q -- pass --last, a session id, or run 'mark --list'", here.Name)
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
		return store.SessionRef{}, fmt.Errorf("%q matches %d sessions: %s -- use a longer prefix",
			prefix, len(matches), listCandidates(matches))
	}
}

// ambiguityShown caps the candidate list an ambiguous prefix prints. A one-character prefix
// matches dozens of sessions on a real store, and a wall of ids is not a suggestion.
const ambiguityShown = 6

func listCandidates(matches []store.SessionRef) string {
	shown := matches
	if len(shown) > ambiguityShown {
		shown = shown[:ambiguityShown]
	}
	ids := make([]string, 0, len(shown))
	for _, m := range shown {
		ids = append(ids, shortSessionID(m.SessionID))
	}
	list := strings.Join(ids, ", ")
	if rest := len(matches) - len(shown); rest > 0 {
		list += fmt.Sprintf(" and %d more", rest)
	}
	return list
}
