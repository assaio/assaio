package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/attribution"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/vcs"
	"github.com/assaio/assaio/internal/version"
)

func newEvidenceCmd() *cobra.Command {
	var since, repo, format string
	c := &cobra.Command{
		Use:   "evidence",
		Short: "Local session-to-commit candidates with explicit ambiguity and coverage",
		Long: `Compare content-free local session observations with commits reachable from a local
repository. The result is an observation from project and time proximity, never proof that an
AI session caused a commit. Competing candidates stay ambiguous and missing evidence stays
unmatched. Nothing is stored or sent, and this command has no team-store mode.`,
		Example: `  assaio-agent evidence --repo . --since 30d
  assaio-agent evidence --repo ../service --format json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runEvidence(cmd, since, repo, format)
		},
	}
	c.Flags().StringVar(&since, "since", "30d", "time window, e.g. 7d")
	c.Flags().StringVar(&repo, "repo", ".", "path to the local git repository")
	c.Flags().StringVar(&format, "format", "text", "output format: text|json")
	return c
}

func runEvidence(cmd *cobra.Command, since, repo, format string) error {
	if format != "text" && format != "json" {
		return fmt.Errorf("unknown format %q (want text|json)", format)
	}
	if err := resolveSince(cmd, &since); err != nil {
		return err
	}
	now := time.Now().UTC()
	start, err := parseSinceAt(since, now)
	if err != nil {
		return err
	}
	root, err := vcs.RepoRoot(cmd.Context(), repo)
	if err != nil {
		return err
	}
	project := vcs.Project(root)
	st, err := openReportStore(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	rows, err := st.Sessions(cmd.Context(), start)
	if err != nil {
		return err
	}
	sessions, otherProjects, unknownProjects, err := evidenceSessions(rows, project)
	if err != nil {
		return err
	}
	commits, skipped, err := vcs.Collect(cmd.Context(), root, start, now, version.Version)
	if err != nil {
		return err
	}
	results := attribution.Match(sessions, commits, nil, now)
	doc := attribution.Document{
		Algorithm: attribution.Algorithm, Project: project, Since: start, ObservedAt: now,
		MaxGapSeconds: int64(attribution.DefaultMaxGap.Seconds()), WindowSessions: len(rows),
		OtherProjects: otherProjects, ProjectUnknown: unknownProjects, SkippedCommits: skipped,
		Summary: attribution.Summarize(results), Results: results,
	}
	if format == "text" {
		return attribution.RenderText(cmd.OutOrStdout(), &doc)
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

func evidenceSessions(rows []store.SessionRow, project string) ([]attribution.Session, int, int, error) {
	sessions := make([]attribution.Session, 0, len(rows))
	var otherProjects, unknownProjects int
	for i := range rows {
		row := &rows[i]
		if row.Member != "" {
			return nil, 0, 0, errors.New("evidence is local-only and cannot read member rows from a team store")
		}
		switch {
		case row.Project == "":
			unknownProjects++
		case row.Project != project:
			otherProjects++
		default:
			sessions = append(sessions, attribution.Session{
				ID: row.SessionID, Project: row.Project, Tool: row.Tool,
				StartedAt: row.FirstTs, EndedAt: row.LastTs,
			})
		}
	}
	return sessions, otherProjects, unknownProjects, nil
}
