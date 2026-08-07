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

--strict turns the diagnosis into a gate, exiting non-zero when a canary fired or a
configured source finds no inputs at all -- so a cron or CI job alerts on vendor format
drift instead of a human eventually noticing the numbers shrank.`,
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
			st, err := store.Open(dbPath)
			if err != nil {
				cmd.Printf("store:        ERROR %v\n", err)
				return nil
			}
			defer func() { _ = st.Close() }()
			n, err := st.Count(cmd.Context())
			if err != nil {
				cmd.Printf("store:        ERROR counting records: %v\n", err)
				return nil
			}
			cmd.Printf("store:        ok, %d record(s) at %s\n", n, dbPath)
			if size, sizeErr := st.Size(cmd.Context()); sizeErr == nil {
				cmd.Printf("size:         %s\n", storeSizeLine(size))
			}
			// The 0.12 upgrade moves the pre-response-grain Claude rows aside rather than
			// deleting them, so the space they hold has to be visible and droppable.
			if rows, archErr := st.LegacyArchiveRows(cmd.Context()); archErr == nil && rows > 0 {
				cmd.Printf("archived:     %d pre-0.12 Claude Code row(s) kept aside by migration 0008; no report reads them.\n", rows)
				cmd.Println("              Drop with: sqlite3 <db> 'DROP TABLE usage_record_pre_response_grain', then `assaio-agent compact`.")
			}
			cmd.Printf("inventory:    %s\n", doctorInventoryLabel(cmd, st, n))
			cmd.Printf("freshness:    %s\n", doctorFreshnessLabel(cmd, st))
			warnings, err := driftWarnings(cmd.Context(), st)
			if err != nil {
				return err
			}
			cmd.Print(doctorDriftSection(warnings))

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
			if failures := strictFailures(warnings, scans); strict && len(failures) > 0 {
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

// doctorInventoryLabel renders the doctor summary line for distinct projects, models,
// and tools seen across all stored records; pricing/store errors degrade to zero
// counts rather than failing doctor's diagnostic output.
func doctorInventoryLabel(cmd *cobra.Command, st *store.Store, n int64) string {
	rows, _ := st.Usage(cmd.Context(), time.Time{})
	table, _ := pricing.Load()
	inv := report.BuildInventory(rows, table)
	return fmt.Sprintf("%d projects · %d models · %d tools across %d record(s)", inv.Projects, inv.Models, inv.Tools, n)
}
