package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/dashboard"
	"github.com/assaio/assaio/internal/store"
)

// dashboardDefaultOutput is where the dashboard writes when --output is not given.
const dashboardDefaultOutput = "assaio-dashboard.html"

func newDashboardCmd() *cobra.Command {
	var since, output string
	c := &cobra.Command{
		Use:   "dashboard",
		Short: "Write a self-contained offline HTML Assay report of local AI usage",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDashboard(cmd, &since, &output)
		},
	}
	c.Flags().StringVar(&since, "since", "30d", "time window, e.g. 7d")
	c.Flags().StringVar(&output, "output", dashboardDefaultOutput, "HTML output path")
	// anonymize/no-anonymize are read via cmd.Flags().Changed in resolveAnonymize, not
	// through these locals, so an unset flag can fall back to config.
	c.Flags().Bool("anonymize", true, "pseudonymize project and member names (default: from config privacy.anonymize)")
	c.Flags().Bool("no-anonymize", false, "show real project and member names, overriding --anonymize")
	addDBFlag(c)
	return c
}

func runDashboard(cmd *cobra.Command, since, output *string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	if !cmd.Flags().Changed("since") {
		*since = cfg.Since
	}
	start, err := parseSinceAt(*since, time.Now())
	if err != nil {
		return err
	}
	doAnonymize := resolveAnonymize(cmd, cfg.Privacy.Anonymize)

	st, err := openReportStore(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	n, err := st.Count(cmd.Context())
	if err != nil {
		return err
	}
	if n == 0 {
		cmd.Println(emptyStoreHint(cmd, "No usage yet; writing an empty dashboard."))
	}

	in, err := buildAnalyzeInput(cmd, st, start)
	if err != nil {
		return err
	}
	in.PlanMonthlyCost = cfg.Pricing.MonthlySubscriptionCost
	subpaths, err := loadDrillSubpaths(cmd, st, &in, start)
	if err != nil {
		return err
	}
	extras := runMetricPlugins(cmd.Context(), cfg.Metrics, &in, cmd.ErrOrStderr())
	data := dashboard.Build(in, windowLabel(*since), doAnonymize, subpaths, extras)

	if err := writeDashboardFile(*output, &data); err != nil {
		return err
	}
	cmd.Println(dashboardWroteLine(*output, *since, doAnonymize))
	return nil
}

// loadDrillSubpaths fetches the subpath breakdown for in's top project by AI lines --
// the one extra store query the report's drill-down needs beyond what buildAnalyzeInput
// already loaded, since subpath isn't part of store.UsageRow's day/tool/model/project
// grouping. Returns nil when no project has any usage in the window, matching
// dashboard.Build's own "no project to drill into" case.
func loadDrillSubpaths(cmd *cobra.Command, st *store.Store, in *analyze.Input, start time.Time) ([]store.SubpathRow, error) {
	top := dashboard.TopProject(in.Usage)
	if top == "" {
		return nil, nil
	}
	return st.Subpaths(cmd.Context(), top, start)
}

// resolveAnonymize applies --no-anonymize/--anonymize precedence over cfgDefault: an
// explicit --no-anonymize always wins, then an explicit --anonymize, then config. Both
// flags take an optional value (--no-anonymize=false, --anonymize=false), so precedence
// is decided by which flag was Changed, but the result must honor the VALUE the caller
// set -- not just assume the flag's bare/default meaning -- or "--no-anonymize=false"
// (asking to keep anonymization on) would perversely turn it off, and "--anonymize=false"
// (asking for real names) would perversely turn it on.
func resolveAnonymize(cmd *cobra.Command, cfgDefault bool) bool {
	switch {
	case cmd.Flags().Changed("no-anonymize"):
		noAnonymize, _ := cmd.Flags().GetBool("no-anonymize")
		return !noAnonymize
	case cmd.Flags().Changed("anonymize"):
		anonymize, _ := cmd.Flags().GetBool("anonymize")
		return anonymize
	default:
		return cfgDefault
	}
}

// writeDashboardFile creates (or truncates) path and renders data into it. The
// dashboard is a shareable artifact, so its perms are intentionally world-readable.
func writeDashboardFile(path string, data *dashboard.Data) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644) //nolint:gosec // shareable HTML artifact, deliberately world-readable
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	return dashboard.RenderHTML(f, *data)
}

// windowLabel turns a validated "Nd" --since value into a human label, e.g. "30d" ->
// "last 30 days".
func windowLabel(since string) string {
	n := strings.TrimSuffix(since, "d")
	unit := "days"
	if n == "1" {
		unit = "day"
	}
	return fmt.Sprintf("last %s %s", n, unit)
}

// dashboardWroteLine is the friendly confirmation printed after a successful write.
func dashboardWroteLine(path, since string, anonymized bool) string {
	if anonymized {
		return fmt.Sprintf("Wrote dashboard to %s (window: %s, project/member names pseudonymized).", path, windowLabel(since))
	}
	return fmt.Sprintf("Wrote dashboard to %s (window: %s).", path, windowLabel(since))
}
