package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

func newReportCmd() *cobra.Command {
	var since, format, by string
	c := &cobra.Command{
		Use:   "report",
		Short: "Print a token/cost report from stored usage",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runReport(cmd, &since, &format, by)
		},
	}
	c.Flags().StringVar(&since, "since", "30d", "time window, e.g. 7d")
	c.Flags().StringVar(&format, "format", "table", "output format: table|json|csv")
	c.Flags().StringVar(&by, "by", "day", "group by: day|project|tool|model|entrypoint|member")
	c.Flags().Bool("compare", false, "show period-over-period top movers vs the previous equal window (renders a movers table, not --format)")
	addDBFlag(c)
	return c
}

func runReport(cmd *cobra.Command, since, format *string, by string) error {
	if err := resolveReportFlags(cmd, since, format); err != nil {
		return err
	}
	start, err := parseSinceAt(*since, time.Now())
	if err != nil {
		return err
	}
	st, err := openReportStore(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()
	if compare, _ := cmd.Flags().GetBool("compare"); compare {
		return runCompare(cmd, st, *since, by)
	}
	built, err := buildReport(cmd, st, start, by)
	if err != nil {
		return err
	}
	return renderReport(cmd, built, *format, by)
}

// resolveReportFlags fills since and format from config when the caller did not
// override them on the command line.
func resolveReportFlags(cmd *cobra.Command, since, format *string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	if !cmd.Flags().Changed("since") {
		*since = cfg.Since
	}
	if !cmd.Flags().Changed("format") {
		*format = cfg.Format
	}
	return nil
}

// resolveSince is the since-only sibling of resolveReportFlags, for commands whose
// --format vocabulary, if any, is not config's table|json|csv (status, dashboard,
// check).
func resolveSince(cmd *cobra.Command, since *string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	if !cmd.Flags().Changed("since") {
		*since = cfg.Since
	}
	return nil
}

// addDBFlag registers the --db override every read command (report, effectiveness,
// analyze, status, dashboard) shares: a team operator points it at a central
// `assaio-agent serve` store instead of this machine's own local one. Unset (the
// default) keeps opening the local store, e.g. `assaio-agent sync` -- which shares
// openReportStore/resolveDBPath but never registers this flag -- always does.
func addDBFlag(c *cobra.Command) {
	c.Flags().String("db", "", "override the store path, e.g. point at a central team-server store")
}

// resolveDBPath returns cmd's --db value when the command defines that flag and the
// caller set it explicitly, else the default local store path (internal/paths.DBPath).
// Every command reaching this helper only reads a store, so an explicit --db must already
// exist: silently creating an empty database at a typo'd path would look identical to "no
// usage yet" instead of surfacing the wrong-path mistake it actually is. Reading the flag
// via Lookup rather than GetString means a command that never calls addDBFlag (sync,
// backfill, clear, doctor, ...) falls through to the default instead of erroring on an
// undefined flag.
func resolveDBPath(cmd *cobra.Command) (string, error) {
	f := cmd.Flags().Lookup("db")
	if f == nil || !f.Changed || f.Value.String() == "" {
		return paths.DBPath()
	}
	dbPath := f.Value.String()
	if _, err := os.Stat(dbPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("--db path %s does not exist", dbPath)
		}
		return "", err
	}
	return dbPath, nil
}

// emptyStoreHint is the message shown when a report-family command's store has no
// matching records. The backfill suggestion only makes sense for the default local
// store: backfill has no --db flag and always writes to paths.DBPath(), so repeating it
// while --db points elsewhere would send the user to populate the wrong database.
func emptyStoreHint(cmd *cobra.Command, prefix string) string {
	if cmd.Flags().Changed("db") {
		return prefix + " This store has no usage records."
	}
	return prefix + " Run 'assaio-agent backfill' to import your local session logs."
}

func openReportStore(cmd *cobra.Command) (*store.Store, error) {
	dbPath, err := resolveDBPath(cmd)
	if err != nil {
		return nil, err
	}
	if err := ensureParent(dbPath); err != nil {
		return nil, err
	}
	return store.Open(dbPath)
}

func buildReport(cmd *cobra.Command, st *store.Store, start time.Time, by string) ([]report.Row, error) {
	rows, err := st.Usage(cmd.Context(), start)
	if err != nil {
		return nil, err
	}
	table, err := pricing.Load()
	if err != nil {
		return nil, err
	}
	built := report.Build(rows, table)
	return report.Aggregate(built, by)
}

func renderReport(cmd *cobra.Command, built []report.Row, format, by string) error {
	switch format {
	case "table":
		if len(built) == 0 {
			cmd.Println(emptyStoreHint(cmd, "No usage found."))
			return nil
		}
		return report.RenderTable(cmd.OutOrStdout(), built, by)
	case "json":
		return report.RenderJSON(cmd.OutOrStdout(), built)
	case "csv":
		return report.RenderCSV(cmd.OutOrStdout(), built)
	default:
		return fmt.Errorf("unknown format %q (want table|json|csv)", format)
	}
}

// parseSinceAt turns "7d" into a start time relative to now. Only a day window is
// supported in the MVP; the suffix must be 'd'.
func parseSinceAt(window string, now time.Time) (time.Time, error) {
	if !strings.HasSuffix(window, "d") {
		return time.Time{}, fmt.Errorf("invalid window %q (want e.g. 7d)", window)
	}
	days, err := strconv.Atoi(strings.TrimSuffix(window, "d"))
	if err != nil || days < 0 {
		return time.Time{}, fmt.Errorf("invalid window %q", window)
	}
	return now.AddDate(0, 0, -days), nil
}
