package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/i18n"
	"github.com/assaio/assaio/internal/parser/claude"
	"github.com/assaio/assaio/internal/parser/cline"
	"github.com/assaio/assaio/internal/parser/codex"
	"github.com/assaio/assaio/internal/parser/gemini"
	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose detected AI tools, log locations, and store health",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			home, err := paths.Home()
			if err != nil {
				return err
			}
			cfg, err := loadConfigLenient(cmd)
			if err != nil {
				return err
			}

			cmd.Print(doctorClaudeLine(cfg.Sources.Claude, paths.ClaudeRoot(home)))
			cmd.Print(doctorSourceLine("codex", "file", cfg.Sources.Codex, codex.Discover, paths.CodexRoots(home)...))
			cmd.Print(doctorSourceLine("gemini-cli", "file", cfg.Sources.Gemini, gemini.Discover, paths.GeminiRoot(home)))
			cmd.Print(doctorSourceLine("cline", "task", cfg.Sources.Cline, cline.Discover, paths.ClineRoots(home)...))

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
			cmd.Printf("inventory:    %s\n", doctorInventoryLabel(cmd, st, n))
			cmd.Printf("freshness:    %s\n", doctorFreshnessLabel(cmd, st))

			models, snapshotDate := pricing.Info()
			cmd.Printf("pricing:      %d models, snapshot %s (refresh ships with releases)\n", models, snapshotDate)
			cmd.Println("activity:     Claude Code and Codex turns carry edit/line signals; Gemini and Cline are token-only.")

			cmd.Println("\ncaveats:")
			cmd.Println("  - Claude input_tokens can be a streaming placeholder; totals may diverge from the Console.")
			cmd.Println("  - Codex reasoning tokens are reported but assumed included in output for cost.")
			cmd.Println("  - Gemini tool-use tokens are folded into output tokens; ~/.gemini may be shared with other tools.")
			cmd.Println("  - Cline stores its own request cost; assaio recomputes cost from tokens for cross-tool consistency.")
			cmd.Println("  - The price table is flat per model; long-context (e.g. [1m]) and 1h-cache premiums are not")
			cmd.Println("    modeled yet, so cost for very long-context or heavy-caching sessions is an under-estimate.")
			cmd.Println("  - Days and week-over-week windows are bucketed in UTC; late local-evening work may land on the")
			cmd.Println("    next UTC day.")
			cmd.Println("  - Activity counts (lines/edits/rework), not tokens or cost, can be off if you ingest a session")
			cmd.Println("    while it is still being written; re-running backfill after it ends does not restate that turn.")
			cmd.Println("  - All on-disk log formats are internal and may change between tool versions.")
			return nil
		},
	}
}

// pluginCountLabel renders the doctor summary line for configured exec plugins.
func pluginCountLabel(plugins []config.PluginConfig) string {
	if len(plugins) == 0 {
		return "none"
	}
	return fmt.Sprintf("%d configured", len(plugins))
}

// toolActivityLabel renders a tool's detected count, so a zero count reads
// unambiguously as "not detected" rather than a bare, ambiguous zero.
func toolActivityLabel(n int, noun string) string {
	if n == 0 {
		return fmt.Sprintf("0 %s(s) — not detected", noun)
	}
	return fmt.Sprintf("%d %s(s)", n, noun)
}

// doctorSourceLine renders one tool's discovery line: activity count, the roots
// actually in effect (configured, or the internal/paths default), and whether those
// roots are the default or config-overridden.
func doctorSourceLine(tool, noun string, configured []string, discover func(string) ([]string, error), defaults ...string) string {
	roots := paths.Resolve(configured, defaults...)
	var files []string
	for _, root := range roots {
		found, _ := discover(root)
		files = append(files, found...)
	}
	return doctorLine(tool, toolActivityLabel(len(files), noun), configured, roots)
}

// doctorClaudeLine reports Claude's two kinds of transcript separately. ingest reads
// top-level sessions and the sub-agent transcripts beneath them, so a line counting only
// the former hides the whole sub-agent surface -- thousands of files on a real machine,
// and exactly the usage v0.3.0 made exact.
func doctorClaudeLine(configured []string, defaults ...string) string {
	roots := paths.Resolve(configured, defaults...)
	var main, sub int
	for _, root := range roots {
		m, _ := claude.Discover(root)
		s, _ := claude.DiscoverSubagents(root)
		main += len(m)
		sub += len(s)
	}
	activity := toolActivityLabel(main, "file")
	if sub > 0 {
		activity += fmt.Sprintf(" + %d sub-agent transcript(s)", sub)
	}
	return doctorLine("claude-code", activity, configured, roots)
}

// doctorLine renders a source line's shared shape. A configured root that doesn't exist
// on disk gets a hint line — a missing default root does not, since the tool may simply
// not be installed here.
func doctorLine(tool, activity string, configured, roots []string) string {
	origin := "default"
	if len(configured) > 0 {
		origin = "config-overridden"
	}
	line := fmt.Sprintf("%-14s%s under %v (%s)\n", tool+":", activity, roots, origin)
	if len(configured) > 0 {
		if missing := paths.Missing(roots); len(missing) > 0 {
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
