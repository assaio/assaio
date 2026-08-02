package cli

import (
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/label"
	"github.com/assaio/assaio/internal/store"
)

// runMarkList prints the window's sessions newest first with whatever labels they carry, so
// mark is never a write-only surface: what was labeled, and what still is not, is visible
// from the same command that does the labeling.
func runMarkList(cmd *cobra.Command, st *store.Store, since string) error {
	start, err := parseSinceAt(since, time.Now())
	if err != nil {
		return err
	}
	sessions, err := st.Sessions(cmd.Context(), start)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		cmd.Println(emptyStoreHint(cmd, "No sessions in this window."))
		return nil
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].LastTs.After(sessions[j].LastTs) })

	lw := &lineWriter{w: cmd.OutOrStdout()}
	lw.printf("sessions · last %s · %d found\n\n", since, len(sessions))
	labeled := 0
	for i := range sessions {
		s := &sessions[i]
		labels := describeLabel(store.SessionLabel{Task: s.Task, Outcome: s.Outcome, Difficulty: s.Difficulty})
		if labels == "" {
			labels = "—"
		} else {
			labeled++
		}
		lw.printf("  %-8s  %s  %-16s  %-12s  %s\n", shortSessionID(s.SessionID),
			s.LastTs.Local().Format("2006-01-02 15:04"), truncate(s.Project, 16), s.Tool, labels)
	}
	lw.printf("\n%d of %d labeled. Label one with: assaio-agent mark <id> --%s %s\n",
		labeled, len(sessions), label.Task, label.Axes[label.Task][0])
	return lw.err
}

// describeSession names the session a command acted on, so an implicit target is never a
// silent one.
func describeSession(ref store.SessionRef) string {
	s := shortSessionID(ref.SessionID)
	if ref.Project != "" {
		s += " · " + ref.Project
	}
	if !ref.LastTs.IsZero() {
		s += " · " + ref.LastTs.Local().Format("2006-01-02 15:04")
	}
	return s
}

// describeLabel renders the axes that carry a value, always in label.Names order.
func describeLabel(l store.SessionLabel) string {
	byAxis := map[string]string{label.Task: l.Task, label.Outcome: l.Outcome, label.Difficulty: l.Difficulty}
	parts := make([]string, 0, len(label.Names))
	for _, axis := range label.Names {
		if v := byAxis[axis]; v != "" {
			parts = append(parts, axis+"="+v)
		}
	}
	return strings.Join(parts, " ")
}

// shortSessionIDLen is how much of a session id is shown and is enough to type back as a
// prefix; ids are locally generated UUIDs, so a collision inside one store is not realistic.
const shortSessionIDLen = 8

func shortSessionID(id string) string {
	if len(id) <= shortSessionIDLen {
		return id
	}
	return id[:shortSessionIDLen]
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}
