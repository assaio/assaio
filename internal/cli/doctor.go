package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/i18n"
	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

func newDoctorCmd() *cobra.Command {
	var strict bool
	c := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose detected AI tools, log locations, and store health",
		Long: `Report what assaio can see: each tool's log roots and how many inputs they hold,
the store's health and freshness, and whether any format-drift canary fired.

--strict turns the diagnosis into a gate, exiting non-zero when a canary fired, when a
configured source finds no inputs at all, when the store itself cannot be read, or when too
much of the reported window carries no model price for cost to mean anything -- so a cron or
CI job alerts on vendor format drift and on a price table that has fallen behind, instead of a
human eventually noticing the numbers shrank. The price ceiling is pricing.max_unpriced_share
(default 5% of the window's tokens; 0 turns that half of the gate off), read over the same
window 'since' gives a report, because a share of a whole backfilled history could never
reach it.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			home, err := paths.Home()
			if err != nil {
				return err
			}
			cfg, err := loadConfigLenient(cmd)
			if err != nil {
				return err
			}

			scans := scanSources(home, &cfg.Sources)
			for i := range scans {
				cmd.Print(doctorLine(&scans[i]))
			}

			cmd.Printf("plugins:      %s\n", pluginCountLabel(cfg.Plugins))

			dbPath, err := paths.DBPath()
			if err != nil {
				return err
			}
			if err := ensureParent(dbPath); err != nil {
				return err
			}
			// The price gate reads the same window a report does by default, because that is
			// the window its ceiling was calibrated against.
			since, err := parseSinceAt(cfg.Since, time.Now())
			if err != nil {
				return err
			}
			warnings, storeFailures := doctorStore(cmd, home, dbPath, since, cfg.Since, cfg.Pricing.UnpricedCeiling(), cfg.Trace.HorizonDays)

			models, snapshotDate := pricing.Info()
			cmd.Printf("pricing:      %d models, snapshot %s (refresh ships with releases)\n", models, snapshotDate)
			cmd.Print(doctorDepthSection(scans))

			cmd.Println("\ncaveats:")
			cmd.Println("  - Claude input_tokens can be a streaming placeholder; totals may diverge from the Console.")
			cmd.Println("  - Codex reasoning tokens are reported but assumed included in output for cost.")
			cmd.Println("  - The price table is flat per model; long-context (e.g. [1m]) and 1h-cache premiums are not")
			cmd.Println("    modeled yet, so cost for very long-context or heavy-caching sessions is an under-estimate.")
			cmd.Println("  - Days and week-over-week windows are bucketed in UTC; late local-evening work may land on the")
			cmd.Println("    next UTC day.")
			cmd.Println("  - Activity counts (lines/edits/rework), not tokens or cost, read low on a session ingested")
			cmd.Println("    while it was still being written; the next backfill restates them upward, never downward,")
			cmd.Println("    so a count that first came out too high stays.")
			cmd.Println("  - All on-disk log formats are internal and may change between tool versions.")
			failures := append(strictFailures(warnings, scans), storeFailures...)
			if strict && len(failures) > 0 {
				return fmt.Errorf("strict check failed: %s", strings.Join(failures, "; "))
			}
			return nil
		},
	}
	c.Flags().BoolVar(&strict, "strict", false, "exit non-zero on suspected format drift or a configured source with no inputs")
	return c
}

// pluginCountLabel renders the doctor summary line for configured exec plugins.
func pluginCountLabel(plugins []config.PluginConfig) string {
	if len(plugins) == 0 {
		return "none"
	}
	return fmt.Sprintf("%d configured", len(plugins))
}

// doctorLine renders one scanned source: what was found, the roots actually in effect,
// and whether those roots are the default or config-overridden. A configured root that
// doesn't exist on disk gets a hint line — a missing default root does not, since the tool
// may simply not be installed here.
func doctorLine(sc *sourceScan) string {
	origin := "default"
	if len(sc.configured) > 0 {
		origin = "config-overridden"
	}
	line := fmt.Sprintf("%-14s%s under %v (%s)\n", sc.tool+":", sc.activity, sc.roots, origin)
	if sc.err != nil {
		line += fmt.Sprintf("  warning: discovery failed, so the count above is not what is there: %v\n", sc.err)
	}
	if len(sc.configured) > 0 {
		if missing := paths.Missing(sc.roots); len(missing) > 0 {
			line += fmt.Sprintf("  hint: configured path(s) not found: %v\n", missing)
		}
	}
	return line
}

// doctorFreshnessLabel reports when each source was last ingested: how current the stored
// figures are, which no other line answers.
func doctorFreshnessLabel(cmd *cobra.Command, st *store.Store) string {
	freshness, err := st.IngestFreshness(cmd.Context())
	if err != nil || len(freshness) == 0 {
		return "no ingest recorded yet — run 'assaio-agent backfill'"
	}
	tools := make([]string, 0, len(freshness))
	for tool := range freshness {
		tools = append(tools, tool)
	}
	sort.Strings(tools)
	now := time.Now()
	l := i18n.For("").CLI
	parts := make([]string, 0, len(tools))
	for _, tool := range tools {
		parts = append(parts, tool+" "+agoLabel(now.Sub(freshness[tool]), &l))
	}
	return strings.Join(parts, " · ")
}

// storeContents is what doctor reads out of the usage rows: the inventory line, which
// describes the whole store, and the price coverage, which describes a window.
type storeContents struct {
	Inventory string
	Unpriced  report.Unpriced
	Models    []string
	Window    string
}

// doctorStoreContents answers both questions doctor asks of the stored rows from one read,
// since the store can hold hundreds of thousands of them.
//
// The two questions have different scopes on purpose. What is in the store is a fact about
// all of it. What the price table cannot cost is a fact about a *window*: the ceiling
// --strict gates on was calibrated against how fast a newly adopted model takes over a 7- or
// 30-day window, and applying it to a store holding years of history buries that same model
// under a denominator it can never move. A gate that cannot fire in the case it was built for
// is not a gate.
//
// An unreadable store is returned as an error rather than degraded to zero counts: the price
// line would otherwise report a clean bill of health for rows nobody read.
func doctorStoreContents(cmd *cobra.Command, st *store.Store, n int64, since time.Time, window string) (storeContents, error) {
	rows, err := st.Usage(cmd.Context(), time.Time{})
	if err != nil {
		return storeContents{}, err
	}
	table, err := pricing.Load()
	if err != nil {
		return storeContents{}, err
	}
	inv := report.BuildInventory(rows, table)
	windowed := rowsSince(rows, since)
	return storeContents{
		Inventory: fmt.Sprintf("%d projects · %d models · %d tools across %d record(s)",
			inv.Projects, inv.Models, inv.Tools, n),
		Unpriced: report.BuildInventory(windowed, table).Unpriced,
		Models:   report.UnpricedModels(windowed, table),
		Window:   window,
	}, nil
}

// rowsSince keeps the rows on or after since. Days are the store's own UTC day strings, so
// the comparison is the same one every windowed query makes.
func rowsSince(rows []store.UsageRow, since time.Time) []store.UsageRow {
	day := since.UTC().Format(time.DateOnly)
	out := make([]store.UsageRow, 0, len(rows))
	for i := range rows {
		if rows[i].Day >= day {
			out = append(out, rows[i])
		}
	}
	return out
}
