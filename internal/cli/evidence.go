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
	var withGitHub bool
	c := &cobra.Command{
		Use:   "evidence",
		Short: "Local session-to-commit candidates with explicit ambiguity and coverage",
		Long: `Compare content-free local session observations with commits reachable from a local
repository. The result is an observation from project and time proximity, never proof that an
AI session caused a commit. Competing candidates stay ambiguous and missing evidence stays
unmatched. Nothing is stored, and this command has no team-store mode. Nothing is sent unless
--github is given: then your own gh asks GitHub for this repository's pull requests updated in
the window, naming the repository and never a commit or a session.`,
		Example: `  assaio-agent evidence --repo . --since 30d
  assaio-agent evidence --repo ../service --format json
  assaio-agent evidence --repo . --github`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runEvidence(cmd, since, repo, format, withGitHub)
		},
	}
	c.Flags().StringVar(&since, "since", "30d", "time window, e.g. 7d")
	c.Flags().StringVar(&repo, "repo", ".", "path to the local git repository")
	c.Flags().StringVar(&format, "format", "text", "output format: text|json")
	c.Flags().BoolVar(&withGitHub, "github", false,
		"also read the repository's pull requests through your own gh (a network request to GitHub)")
	return c
}

func runEvidence(cmd *cobra.Command, since, repo, format string, withGitHub bool) error {
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
	st, err := openReportStore(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	here, err := repositoryAt(cmd.Context(), st, root)
	if err != nil {
		return err
	}
	if here.ID == 0 {
		cmd.PrintErrln("note: " + unresolvedNote(&here))
	}
	rows, err := st.Sessions(cmd.Context(), start)
	if err != nil {
		return err
	}
	pop, err := evidenceSessions(rows, here.ID, here.Name, here.Unplaced)
	if err != nil {
		return err
	}
	commits, skipped, err := vcs.Collect(cmd.Context(), root, here.Name, start, now, version.Version)
	if err != nil {
		return err
	}
	var prs *attribution.PullRequests
	if withGitHub {
		var listedSkipped int
		if prs, commits, listedSkipped, err = readPullRequests(cmd, root, here.Name, start, now, commits); err != nil {
			return err
		}
		skipped += listedSkipped
	}
	results := attribution.Match(pop.sessions, commits, nil, now)
	doc := attribution.Document{
		Algorithm: attribution.Algorithm, Project: here.Name, Since: start, ObservedAt: now,
		MaxGapSeconds: int64(attribution.DefaultMaxGap.Seconds()), WindowSessions: len(rows),
		OtherProjects: pop.other, ProjectUnknown: pop.unknown, IdentityUnresolved: pop.unresolved,
		SkippedCommits: skipped,
	}
	if prs != nil {
		attribution.LinkChanges(results, commits, prs)
		doc.Algorithm, doc.Changes = attribution.AlgorithmChanges, attribution.ChangesOf(prs, results, "gh")
	}
	doc.Forge, doc.Summary, doc.Results = attribution.ForgeOf(commits, results), attribution.Summarize(results), results
	if format == "text" {
		return attribution.RenderText(cmd.OutOrStdout(), &doc)
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

// evidencePopulation is the window's sessions sorted by what they are to the repository the
// command stands in.
type evidencePopulation struct {
	sessions                   []attribution.Session
	other, unknown, unresolved int
}

// evidenceSessions keeps the sessions whose rows resolved to repository id -- never one merely
// sharing its name. A session under the name its unplaced rows carry, with no repository of its
// own, is counted unresolved: it may be this repository's, and a graph edge does not guess.
func evidenceSessions(rows []store.SessionRow, id int64, project, unplaced string) (evidencePopulation, error) {
	pop := evidencePopulation{sessions: make([]attribution.Session, 0, len(rows))}
	for i := range rows {
		row := &rows[i]
		if row.Member != "" {
			return evidencePopulation{}, errors.New("evidence is local-only and cannot read member rows from a team store")
		}
		switch {
		case row.Project == "":
			pop.unknown++
		case id > 0 && row.RepoID == id:
			pop.sessions = append(pop.sessions, attribution.Session{
				ID: row.SessionID, Project: project, Tool: row.Tool,
				StartedAt: row.FirstTs, EndedAt: row.LastTs,
			})
		case row.RepoID == 0 && row.Project == unplaced:
			pop.unresolved++
		default:
			pop.other++
		}
	}
	return pop, nil
}
