package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/digest"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

func newDigestCmd() *cobra.Command {
	var since string
	var weekly, dry bool
	c := &cobra.Command{
		Use:   "digest",
		Short: "Report in markdown what moved since the last digest",
		Long: `Write a short markdown summary of what CHANGED -- top movers, verdict changes, and
the reasons a comparison may not mean what it looks like -- rather than restating what the
other surfaces already show. It is meant for cron or launchd; delivery stays your own
script, so pipe it into mail, a Slack webhook, or a file.

Each run records what it reported so the next one has something to compare against. The
first run therefore has no comparison and says so, instead of reporting every figure as
new. Nothing about a session's content is recorded -- only totals, per-model and
per-project weights, and each validator's verdict.

A digest also states when the comparison itself is weak: two windows that overlap, windows
of different lengths, or a parser that changed between the runs and therefore corrected
history underneath the numbers.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDigest(cmd, since, weekly, dry)
		},
	}
	c.Flags().StringVar(&since, "since", "7d", "window to summarize, e.g. 30d")
	c.Flags().BoolVar(&weekly, "weekly", false, "shorthand for --since 7d, the cadence this is built for")
	c.Flags().BoolVar(&dry, "dry-run", false, "print the digest without recording this run as a comparison basis")
	addDBFlag(c)
	return c
}

func runDigest(cmd *cobra.Command, since string, weekly, dry bool) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	if weekly {
		since = "7d"
	} else if !cmd.Flags().Changed("since") {
		since = cfg.Since
	}
	start, err := parseSinceAt(since, time.Now())
	if err != nil {
		return err
	}
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
		return emptyStatusHint(cmd)
	}
	in, err := buildAnalyzeInput(cmd, st, start)
	if err != nil {
		return err
	}
	in.PlanMonthlyCost = cfg.Pricing.MonthlySubscriptionCost

	results := runValidatorResults(analyze.Validators(), &in)
	// A metric plugin that failed leaves a hole in the verdict set, and a hole compared
	// against last week's verdict renders as a metric that vanished. `check` already refuses
	// to pass on an incomplete set for the same reason; the digest names what did not run.
	plugins, unevaluated := runMetricPlugins(cmd.Context(), cfg.Metrics, &in, cmd.ErrOrStderr())
	results = append(results, plugins...)
	// The digest leads with the same findings analyze leads with, which is what MarkLead
	// decides -- and it can only be asked of a run that computed the whole window, which
	// this one always does.
	analyze.MarkLead(results)

	note, share := unpricedDisclosure(&in)
	now := digest.Take(&in, results, digest.Options{
		Window: since, UnpricedNote: note, UnpricedShare: share, At: time.Now(),
	})
	previous, err := previousSnapshot(cmd, st, now.TakenAt, since)
	if err != nil {
		return err
	}
	result := digest.Compare(&now, previous, unevaluated)
	cmd.Print(result.WithPseudonym(projectPseudonym(cfg.Privacy.Anonymize)).Markdown())
	if dry {
		return nil
	}
	return saveSnapshot(cmd, st, &now, since)
}

// unpricedDisclosure returns the same sentence the cost tables print and the share it is
// about, so the digest states the size of the gap in its `$` figure rather than only that
// estimates are estimates.
func unpricedDisclosure(in *analyze.Input) (note string, share float64) {
	inv := report.BuildInventory(in.Usage, in.Prices)
	return report.UnpricedDisclosure(&inv.Unpriced, "this window's tokens"), inv.Unpriced.Share()
}

// projectPseudonym returns the renamer a digest's project names go through, or nil when the
// person turned anonymization off. A digest is written to be sent somewhere, which is the
// case privacy.anonymize exists for; the dashboard already treats its own shared file this way.
func projectPseudonym(anonymize bool) func(string) string {
	if !anonymize {
		return nil
	}
	return func(project string) string { return report.Pseudonym(analyze.PseudonymProject, project) }
}

// previousSnapshot reads the last run's basis for the same window. A payload this build
// cannot parse is treated as no basis at all: reporting movement against a shape we may be
// misreading would be worse than reporting a first run.
func previousSnapshot(cmd *cobra.Command, st *store.Store, at time.Time, window string) (*digest.Snapshot, error) {
	stored, ok, err := st.PreviousDigestSnapshot(cmd.Context(), at, window)
	if err != nil || !ok {
		return nil, err
	}
	snap, ok := digest.Parse(stored.Payload)
	if !ok {
		return nil, nil
	}
	// The row's columns are what ordered the snapshots and selected this one, so they are
	// the authority on when it was taken and which build read it. The payload carries its
	// own copies for self-description; letting those win would compare against a timestamp
	// that did not choose this row.
	snap.TakenAt, snap.ParsedBy = stored.TakenAt, stored.ParsedBy
	return &snap, nil
}

func saveSnapshot(cmd *cobra.Command, st *store.Store, snap *digest.Snapshot, window string) error {
	payload, err := snap.Marshal()
	if err != nil {
		return err
	}
	return st.SaveDigestSnapshot(cmd.Context(), &store.DigestSnapshot{
		TakenAt: snap.TakenAt, ParsedBy: snap.ParsedBy, Window: window, Payload: payload,
	})
}
