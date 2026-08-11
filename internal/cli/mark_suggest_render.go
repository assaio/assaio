package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/label"
	"github.com/assaio/assaio/internal/store"
)

// suggestionsShown bounds the listing; the count that follows always states the whole,
// so a truncated list never reads as the complete one.
const suggestionsShown = 30

func renderSuggestions(cmd *cobra.Command, found []suggestion, sessions []store.SessionRow, since string) error {
	lw := &lineWriter{w: cmd.OutOrStdout()}
	lw.printf("suggested labels · last %s · %s sessions\n\n", since, humanize.Int(int64(len(sessions))))

	if len(found) == 0 {
		lw.printf("  No session derives a label.\n\n")
		lw.printf("%s\n", noSuggestionExplanation(sessions))
		return lw.err
	}
	for i, f := range found {
		if i == suggestionsShown {
			lw.printf("  … and %s more\n", humanize.Int(int64(len(found)-suggestionsShown)))
			break
		}
		lw.printf("  %-8s  %s  %-16s  %s\n", shortSessionID(f.ref.SessionID),
			f.ref.LastTs.Local().Format("2006-01-02 15:04"), truncate(f.ref.Project, 16), describeAxes(f.axes))
		lw.printf("            ↳ %s\n", f.axes[0].Reason)
	}
	lw.printf("\n%s of %s sessions derive a label; %s derive nothing.\n",
		humanize.Int(int64(len(found))), humanize.Int(int64(len(sessions))),
		humanize.Int(int64(len(sessions)-len(found))))
	lw.printf("%s\n", noSuggestionExplanation(sessions))
	lw.printf("\nNothing has been written. Accept them with:\n")
	lw.printf("  assaio-agent mark --accept-suggested --since %s\n", since)
	lw.printf("A label made by hand is never replaced, and an accepted one can be changed with 'mark <id>'.\n")
	return lw.err
}

// noSuggestionExplanation says why the blanks are blank, because "derives nothing" is a
// measurement of the repository's naming, not a failure of the derivation -- and the
// difference decides whether a person should write a rule or ignore the feature.
func noSuggestionExplanation(sessions []store.SessionRow) string {
	labeled := 0
	for i := range sessions {
		if sessions[i].Task != "" || sessions[i].Outcome != "" || sessions[i].Difficulty != "" {
			labeled++
		}
	}
	msg := "A session derives nothing when its branch follows no convention this rule set knows -- " +
		"which is the intended answer rather than a missing one. Add a rule under labels.rules " +
		"in the config to teach it yours."
	if labeled > 0 {
		msg += "\nSessions already labeled by hand are left alone: " +
			humanize.Int(int64(labeled)) + " in this window."
	}
	return msg
}

func describeAxes(axes []label.Suggestion) string {
	parts := make([]string, 0, len(axes))
	for _, a := range axes {
		parts = append(parts, a.Axis+"="+a.Value)
	}
	return strings.Join(parts, " ")
}
