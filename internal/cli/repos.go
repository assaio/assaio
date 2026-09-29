package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/store"
)

func newReposCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "repos [directory...]",
		Short: "Show which project name each directory's usage is stored under and which names repositories share",
		Long: `List the project names your local sessions are stored under. When repositories on this
machine share a name, the name is split: the first repository keeps it, the others are
numbered, and usage that cannot be placed under one of them is shown with "?".

The store keeps no directory path, so naming directories is how you find out which is which:
each directory is shown with the name its usage is stored under. Nothing is stored or sent.`,
		Example: `  assaio-agent repos
  assaio-agent repos ~/work/api ~/scratch/api`,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := openReportStore(cmd)
			if err != nil {
				return err
			}
			defer func() { _ = st.Close() }()
			if len(args) > 0 {
				return runReposAt(cmd, st, args)
			}
			return runReposList(cmd, st)
		},
	}
}

// runReposAt prints the name each directory's usage is stored under.
func runReposAt(cmd *cobra.Command, st *store.Store, dirs []string) error {
	lw := &lineWriter{w: cmd.OutOrStdout()}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			return err
		}
		here, err := repositoryAt(cmd.Context(), st, dir)
		if err != nil {
			return err
		}
		switch {
		case !here.inRepo:
			lw.printf("  %s  (not in a git repository; its usage is shown under %s)\n", dir, orDash(here.Unplaced))
		case here.ID == 0:
			lw.printf("  %s  %s\n", dir, unresolvedNote(&here))
		default:
			lw.printf("  %s  %s\n", dir, here.Name)
		}
	}
	return lw.err
}

// runReposList prints every stored name with its sessions, marks the working directory's, and
// states how many sessions name their repository.
func runReposList(cmd *cobra.Command, st *store.Store) error {
	rows, err := st.Repositories(cmd.Context())
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		cmd.Println(emptyStoreHint(cmd, "No sessions stored."))
		return nil
	}
	here := ""
	if cwd, err := os.Getwd(); err == nil {
		rep, err := repositoryAt(cmd.Context(), st, cwd)
		if err != nil {
			return err
		}
		if rep.ID > 0 {
			here = rep.Name
		}
	}
	lw := &lineWriter{w: cmd.OutOrStdout()}
	lw.printf("  %-32s  %8s  %10s  %s\n", "NAME", "SESSIONS", "UNRESOLVED", "LAST ACTIVE")
	for i := range rows {
		r := &rows[i]
		mark := ""
		if r.Name != "" && r.Name == here {
			mark = "   ← this directory"
		}
		lw.printf("  %-32s  %8s  %10s  %s%s\n", truncate(orDash(r.Name), 32), humanize.Count(r.Sessions),
			humanize.Count(r.Unresolved), r.LastTs.Local().Format("2006-01-02"), mark)
	}
	split, err := st.SplitLabels(cmd.Context())
	if err != nil {
		return err
	}
	if len(split) > 0 {
		lw.printf("\nProject names shared by more than one repository: %s. Name the directories to see which is which.\n",
			strings.Join(split, ", "))
	}
	c, err := st.IdentityCoverage(cmd.Context())
	if err != nil {
		return err
	}
	lw.printf("\n%s of sessions (%s of %s) and %s of usage rows record their repository.\n"+
		"UNRESOLVED counts the rest: sessions stored before v0.35.0, run outside any git repository, read\n"+
		"from a log with no working directory or after their directory was gone, imported by a plugin, or\n"+
		"spanning two repositories. Under a name only one repository holds, they are counted with it, as\n"+
		"every report does.\n",
		humanize.PercentOrDash(c.SessionsResolved, c.Sessions, 0), humanize.Count(c.SessionsResolved),
		humanize.Count(c.Sessions), humanize.PercentOrDash(c.RowsResolved, c.Rows, 0))
	return lw.err
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
