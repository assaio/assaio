package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/config"
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
			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}

			cmd.Print(doctorSourceLine("claude-code", "file", cfg.Sources.Claude, claude.Discover, paths.ClaudeRoot(home)))
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
			n, _ := st.Count(cmd.Context())
			cmd.Printf("store:        ok, %d record(s) at %s\n", n, dbPath)
			cmd.Printf("inventory:    %s\n", doctorInventoryLabel(cmd, st, n))

			models, snapshotDate := pricing.Info()
			cmd.Printf("pricing:      %d models, snapshot %s (refresh ships with releases)\n", models, snapshotDate)
			cmd.Println("activity:     Claude Code and Codex turns carry edit/line signals; Gemini and Cline are token-only.")

			cmd.Println("\ncaveats:")
			cmd.Println("  - Claude input_tokens can be a streaming placeholder; totals may diverge from the Console.")
			cmd.Println("  - Codex reasoning tokens are reported but assumed included in output for cost.")
			cmd.Println("  - Gemini tool-use tokens are folded into output tokens; ~/.gemini may be shared with other tools.")
			cmd.Println("  - Cline stores its own request cost; assaio recomputes cost from tokens for cross-tool consistency.")
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
// roots are the default or config-overridden. A configured root that doesn't exist on
// disk gets a hint line — a missing default root does not, since the tool may simply
// not be installed here.
func doctorSourceLine(tool, noun string, configured []string, discover func(string) ([]string, error), defaults ...string) string {
	roots := paths.Resolve(configured, defaults...)
	var files []string
	for _, root := range roots {
		found, _ := discover(root)
		files = append(files, found...)
	}
	overridden := len(configured) > 0
	origin := "default"
	if overridden {
		origin = "config-overridden"
	}
	line := fmt.Sprintf("%-14s%s under %v (%s)\n", tool+":", toolActivityLabel(len(files), noun), roots, origin)
	if overridden {
		if missing := paths.Missing(roots); len(missing) > 0 {
			line += fmt.Sprintf("  hint: configured path(s) not found: %v\n", missing)
		}
	}
	return line
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
