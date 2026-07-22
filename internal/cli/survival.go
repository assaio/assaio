package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/survival"
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
	root, err := survival.RepoRoot(cmd.Context(), repo)
	if err != nil {
		return err
	}
	aiLines, err := projectAILines(cmd, survival.Project(root), start)
	if err != nil {
		return err
	}
	res, err := survival.Analyze(cmd.Context(), root, start, aiLines)
	if err != nil {
		return err
	}
	return renderSurvival(cmd, &res, since)
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

func renderSurvival(cmd *cobra.Command, res *survival.Result, since string) error {
	rate := "—"
	if res.SurvivalRate >= 0 {
		rate = fmt.Sprintf("%.0f%%", res.SurvivalRate*100)
	}
	cmd.Printf("Survival · %s   window: %s\n", res.Project, windowLabel(since))
	cmd.Printf("  %d commits in window · %d files blamed\n\n", res.Commits, res.Files)
	cmd.Printf("  AI lines (assaio):   %d\n", res.AILines)
	cmd.Printf("  Lines added (git):   %d\n", res.GitAdded)
	cmd.Printf("  Surviving in HEAD:   %d  (%s)\n\n", res.Surviving, rate)
	cmd.Println("  Directional: assaio counts AI lines but cannot tell which git lines were AI-written,")
	cmd.Println("  so this is repo-wide survival of the window's commits shown beside your AI usage -- a")
	cmd.Println("  correlation to read, not a per-line AI-survival number. Age-matched bug/quality impact")
	cmd.Println("  and team-wide DORA signals are the server stage (see ROADMAP).")
	return nil
}
