package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/survival"
	"github.com/assaio/assaio/internal/vcs"
	"github.com/assaio/assaio/internal/version"
)

func newSurvivalCmd() *cobra.Command {
	var since, repo string
	c := &cobra.Command{
		Use:   "survival",
		Short: "Directional AI-code survival: how much of a repo's recent commits still live in HEAD, beside your AI usage",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSurvival(cmd, since, repo)
		},
	}
	c.Flags().StringVar(&since, "since", "90d", "window, e.g. 90d")
	c.Flags().StringVar(&repo, "repo", ".", "path to the git repository to analyze")
	addDBFlag(c)
	return c
}

func runSurvival(cmd *cobra.Command, since, repo string) error {
	start, err := parseSinceAt(since, time.Now())
	if err != nil {
		return err
	}
	root, err := vcs.RepoRoot(cmd.Context(), repo)
	if err != nil {
		return err
	}
	aiLines, err := projectAILines(cmd, vcs.Project(root), start)
	if err != nil {
		return err
	}
	commits, skipped, err := vcs.Collect(cmd.Context(), root, start, time.Now(), version.Version)
	if err != nil {
		return err
	}
	files, err := vcs.TouchedFiles(cmd.Context(), root, start)
	if err != nil {
		return err
	}
	res, err := survival.Analyze(cmd.Context(), root, commits, files, aiLines)
	if err != nil {
		return err
	}
	return renderSurvival(cmd, &res, since, skipped)
}

// projectAILines sums the AI lines the store recorded for project since start.
func projectAILines(cmd *cobra.Command, project string, start time.Time) (int64, error) {
	st, err := openReportStore(cmd)
	if err != nil {
		return 0, err
	}
	defer func() { _ = st.Close() }()
	rows, err := st.Usage(cmd.Context(), start)
	if err != nil {
		return 0, err
	}
	var lines int64
	for i := range rows {
		if rows[i].Project == project {
			lines += rows[i].LinesAdded
		}
	}
	return lines, nil
}

func renderSurvival(cmd *cobra.Command, res *survival.Result, since string, skipped int) error {
	rate := "—"
	if res.SurvivalRate >= 0 {
		rate = fmt.Sprintf("%.0f%%", res.SurvivalRate*100)
	}
	cmd.Printf("Survival · %s   window: %s\n", res.Project, windowLabel(since))
	cmd.Printf("  %d commits in window · %d files blamed%s%s\n",
		res.Commits, res.Files, unreadableNote(res.Unreadable), skippedNote(skipped))
	cmd.Printf("  changed files:       %s\n", categoryLine(&res.Changed))
	if res.Reverts > 0 {
		cmd.Printf("  reverts:             %d commit(s) git itself labelled a revert\n", res.Reverts)
	}
	if res.Merges > 0 {
		cmd.Printf("  merges:              %d commit(s) holding %d line(s) in HEAD, counted in neither figure below\n",
			res.Merges, res.MergeLines)
	}
	cmd.Println()
	cmd.Printf("  AI lines (assaio):   %d\n", res.AILines)
	cmd.Printf("  Lines added (git):   %d\n", res.GitAdded)
	cmd.Printf("  Surviving in HEAD:   %d  (%s)\n", res.Surviving, rate)
	cmd.Printf("%s\n\n", survivalAgeLine(res, time.Now()))
	cmd.Println("  Directional: assaio counts AI lines but cannot tell which git lines were AI-written,")
	cmd.Println("  so this is repo-wide survival of the window's commits shown beside your AI usage -- a")
	cmd.Println("  correlation to read, not a per-line AI-survival number. File categories come from a")
	cmd.Println("  naming heuristic, and no path, branch name or commit message leaves the repository.")
	cmd.Println("  git reports no line counts for a merge, so a conflict resolution sits outside both")
	cmd.Println("  the added and the surviving figure rather than inflating either.")
	cmd.Println("  Age-matched bug/quality impact and team-wide DORA signals are the server stage.")
	return nil
}

// survivalAgeLine states how old the commits behind the rate are, because the rate is monotonic
// in that age: the same repository reads near 100% over a week and far lower over a year, so a
// figure whose age is unstated invites a comparison between two windows that measured different
// things (B179). A window whose commits carry no time says so rather than inventing an age.
func survivalAgeLine(res *survival.Result, now time.Time) string {
	const wrap = "\n  "
	head := "  Age of what was measured:"
	median, ok := res.MedianCommitAgeDays(now)
	if !ok {
		return head + " unknown -- no commit in this window carries a time." + wrap +
			"Survival rises the younger the commits are, so a rate without an age is not comparable" + wrap +
			"to another window's."
	}
	line := head + " median commit is " + strconv.Itoa(median) + " day(s) old"
	if span, spanOK := res.AgeSpanDays(); spanOK {
		line += ", spanning " + strconv.Itoa(span) + " day(s)"
	}
	return line + "." + wrap +
		"Survival is monotonic in commit age -- a line written yesterday has had no time to be" + wrap +
		"rewritten -- so this rate is comparable only against a window of about the same age, never" + wrap +
		"against the same repository read over a longer --since."
}

// categoryLine renders the window's changed files per category, naming only the categories
// that occur: a row of zeros reads like a measurement where it is an absence.
func categoryLine(c *event.FileCategories) string {
	buckets := []struct {
		label string
		n     int64
	}{
		{"source", c.Source},
		{"test", c.Test},
		{"docs", c.Docs},
		{"config", c.Config},
		{"generated", c.Generated},
		{"other", c.Other},
	}
	var parts []string
	for _, b := range buckets {
		if b.n > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", b.label, b.n))
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, " · ")
}

// unreadableNote names touched files git could not blame, so a rate computed over part of
// the window's files never reads as one computed over all of them.
func unreadableNote(unreadable int) string {
	if unreadable == 0 {
		return ""
	}
	return fmt.Sprintf(" · %d file(s) unreadable, outside the rate", unreadable)
}

// skippedNote names commits git printed in a shape this build could not read, so a short
// history is never quietly short.
func skippedNote(skipped int) string {
	if skipped == 0 {
		return ""
	}
	return fmt.Sprintf(" · %d commit(s) unreadable and skipped", skipped)
}
