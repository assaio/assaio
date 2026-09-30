package attribution

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// renderChanges states what the pull-request read covered, what it added to the candidates, and
// every count behind an absence, so a candidate in no commit list read is never read as one that
// was never in a pull request, and a v3 coverage figure is never set beside a v2 one.
func renderChanges(w io.Writer, c *Changes, since time.Time) error {
	if c == nil {
		return nil
	}
	bound := ""
	if c.ReadBackTo.After(since) {
		bound = "; stopped at the 20-page bound"
	}
	_, err := fmt.Fprintf(w,
		"  pull requests (%s, read %s, back to %s%s): %d read · %d commit list(s) cut at 100 (newest cut) · "+
			"%d session(s) in one pull request · %d across several\n"+
			"  listed commits: %d made here, added as candidates · %d in no HEAD reflog here "+
			"(fetched, only checked out, made in a removed worktree, or expired) · %d not local · "+
			"%d unreadable · %d made before the window\n"+
			"  Coverage above counts the %d listed commit(s) made here; compare it only with another %s document.\n",
		c.Source, c.ObservedAt.Format(time.RFC3339), c.ReadBackTo.Format(time.RFC3339), bound, c.PullRequestsRead,
		c.CommitListsCut, c.SessionsInOneChange, c.SessionsAcrossChanges,
		c.ListedMadeHere, c.ListedNotInReflog, c.ListedNotLocal, c.ListedUnreadable, c.ListedBeforeWindow,
		c.ListedMadeHere, AlgorithmChanges)
	return err
}

// changeLinks is the tail of a candidate line: the pull requests the forge says hold the commit and
// how, never their state -- a state printed beside a session reads as the session's outcome.
func changeLinks(c *Candidate, withChanges bool) string {
	switch {
	case !withChanges:
		return ""
	case len(c.Changes) == 0:
		return " · in no pull request commit list read"
	}
	parts := make([]string, 0, len(c.Changes))
	for _, l := range c.Changes {
		parts = append(parts, fmt.Sprintf("#%d %s", l.Number, l.Via))
	}
	return " · pull request " + strings.Join(parts, ", ") + " (source=" + c.Changes[0].Source.Name + ")"
}

// renderLinked lists the pull requests a result names, with the state the forge reported when
// read, apart from every session line.
func renderLinked(w io.Writer, c *Changes) error {
	if c == nil {
		return nil
	}
	if _, err := fmt.Fprintf(w, "\n  Pull requests named above, state as %s reported it at %s "+
		"(the pull request's outcome, not a session's):\n", c.Source, c.ObservedAt.Format(time.RFC3339)); err != nil {
		return err
	}
	for _, pr := range c.Linked {
		merged := ""
		if !pr.MergedAt.IsZero() {
			merged = " " + pr.MergedAt.Format(time.RFC3339)
		}
		if _, err := fmt.Fprintf(w, "    #%d %s%s (named by %s)\n", pr.Number, pr.State, merged, pr.NamedBy); err != nil {
			return err
		}
	}
	return nil
}

func renderOutcomeScope(w io.Writer, withChanges bool) error {
	if !withChanges {
		_, err := fmt.Fprintln(w,
			"  Pull requests are read only with --github; review, CI and downstream outcomes are not part of this command.")
		return err
	}
	_, err := fmt.Fprintln(w, `  A merged pull request says the change landed, not that this session's lines are in it.
  Review, CI and downstream outcomes are not part of this command.`)
	return err
}
