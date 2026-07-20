package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/dashboard"
	"github.com/assaio/assaio/internal/store"
)

const demoIntro = "assaio-agent demo -- the reports you'd get on your own data, shown on bundled sample usage.\n" +
	"No logs were read; the sample lives in a throwaway store, discarded on exit."

const demoOutro = "That's the demo. On your machine: 'assaio-agent backfill' imports your local session logs,\n" +
	"then 'assaio-agent status', 'assaio-agent analyze', and 'assaio-agent dashboard' show the real thing."

func newDemoCmd() *cobra.Command {
	var dash bool
	c := &cobra.Command{
		Use:   "demo",
		Short: "Show assaio's reports on bundled sample data, no local logs needed",
		Long: `Seed a throwaway store with realistic sample usage and print the same reports you'd
get on your own data -- so you can see the full value before backfilling your own logs.
The sample store lives in a temp dir and is removed on exit; your real data is untouched.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDemo(cmd, dash)
		},
	}
	c.Flags().BoolVar(&dash, "dashboard", false, "also write the sample HTML Assay dashboard and print its path")
	return c
}

func runDemo(cmd *cobra.Command, dash bool) error {
	dir, err := os.MkdirTemp("", "assaio-demo")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	st, err := store.Open(filepath.Join(dir, "demo.db"))
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	now := time.Now()
	if _, err := st.Insert(cmd.Context(), demoRecords(now)); err != nil {
		return err
	}
	start := now.AddDate(0, 0, -demoWindowDays)

	lw := &lineWriter{w: cmd.OutOrStdout()}
	lw.println(demoIntro)
	lw.println("")
	if err := runDemoReports(cmd, st, start, lw); err != nil {
		return err
	}
	if dash {
		if err := writeDemoDashboard(cmd, st, start); err != nil {
			return err
		}
	}
	lw.println(demoOutro)
	return lw.err
}

// runDemoReports prints the three headline surfaces against the sample store, reusing the
// same build/render helpers the real report/effectiveness/analyze commands use.
func runDemoReports(cmd *cobra.Command, st *store.Store, start time.Time, lw *lineWriter) error {
	demoHeader(lw, "report -- cost & tokens by project")
	built, err := buildReport(cmd, st, start, "project")
	if err != nil {
		return err
	}
	if err := renderReport(cmd, built, "table", "project"); err != nil {
		return err
	}
	lw.println("")

	demoHeader(lw, "effectiveness -- AI lines vs cost")
	eff, err := buildEffectiveness(cmd, st, start, "project")
	if err != nil {
		return err
	}
	if err := renderEffectiveness(cmd, eff, "table", "project"); err != nil {
		return err
	}
	lw.println("")

	demoHeader(lw, "analyze -- the five-dimension litmus")
	in, err := buildAnalyzeInput(cmd, st, start)
	if err != nil {
		return err
	}
	return renderAnalyzeResults(cmd, runValidatorResults(analyze.Validators(), &in), "text")
}

func demoHeader(lw *lineWriter, title string) {
	lw.printf("-- %s --\n", title)
}

// writeDemoDashboard renders the sample Assay HTML to a stable temp path (outside the
// discarded sample-store dir) so the user can open it after the command exits.
func writeDemoDashboard(cmd *cobra.Command, st *store.Store, start time.Time) error {
	in, err := buildAnalyzeInput(cmd, st, start)
	if err != nil {
		return err
	}
	subpaths, err := loadDrillSubpaths(cmd, st, &in, start)
	if err != nil {
		return err
	}
	// No exec metric plugins on demo: the sample walkthrough stays deterministic.
	data := dashboard.Build(in, "last 30 days (sample)", true, subpaths, nil)
	// A private per-invocation temp dir, not a fixed name in the shared temp root: the
	// latter lets another local user pre-plant or symlink-hijack the path (CWE-377). Kept
	// (not cleaned up) so the user can open it after the command exits.
	dir, err := os.MkdirTemp("", "assaio-demo-dashboard")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "assaio-demo-dashboard.html")
	if err := writeDashboardFile(path, &data); err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "\nWrote sample dashboard to %s -- open it in a browser.\n", path)
	return err
}
