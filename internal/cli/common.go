package cli

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

// The helpers every store-reading command shares: where the store is, how a window is
// parsed, and what to say when there is nothing in it. They live here rather than beside
// any one command because report, effectiveness, analyze, status, dashboard, check, mark
// and sync all reach for them.

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

// compareFormatConflict errors when --compare is combined with an explicit machine format:
// --compare only renders a human movers table, so a scripted `--format json|csv` consumer
// would otherwise get a table with no error. A format inherited from config (not set on
// the command line) is left alone, since --compare then just renders its table.
func compareFormatConflict(cmd *cobra.Command) error {
	if !cmd.Flags().Changed("format") {
		return nil
	}
	if f, _ := cmd.Flags().GetString("format"); f == "json" || f == "csv" {
		return errors.New("--compare renders a movers table and cannot be combined with --format json|csv")
	}
	return nil
}

// emptyStatusHint writes the store-empty message for status/analyze, db-aware like
// emptyStoreHint: the rich backfill suggestion for the local store, or a terse "this store
// has no records" when --db points at a central store backfill cannot populate.
func emptyStatusHint(cmd *cobra.Command) error {
	if cmd.Flags().Changed("db") {
		cmd.Println(emptyStoreHint(cmd, "No usage found."))
		return nil
	}
	return report.RenderEmptyStatusHint(cmd.OutOrStdout())
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

// usageForDim reads the window's usage the way dimension by needs it: grouped by session
// annotation when by is one of them, plain otherwise. Every other command keeps calling
// store.Usage, so the annotation join runs only where it was actually asked for and no
// existing figure can move because someone labeled a session.
func usageForDim(cmd *cobra.Command, st *store.Store, start time.Time, by string) ([]store.UsageRow, error) {
	if report.IsLabelDim(by) {
		return st.UsageByLabel(cmd.Context(), start)
	}
	return st.Usage(cmd.Context(), start)
}

// parseSinceAt turns "7d" into the store-query floor N*24h back from now. A duration, not a
// bucket boundary: consecutive runs of a recurring command must cover contiguous time, and the
// helpers that compare day-buckets align on their own. Only a day window; the suffix must be 'd'.
func parseSinceAt(window string, now time.Time) (time.Time, error) {
	days, err := windowDays(window)
	if err != nil {
		return time.Time{}, err
	}
	return now.AddDate(0, 0, -days), nil
}

// startOfUTCDay is the midnight opening t's UTC day -- where a stored day bucket begins.
func startOfUTCDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}
