// Package cli wires the assaio-agent command-line interface.
package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/version"
)

// NewRootCmd builds the assaio-agent root command with all subcommands attached.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "assaio-agent",
		Short: "Offline reports, diagnostics, and dashboards for local AI-coding session logs",
		Long: `assaio-agent reads the local session logs of AI coding tools (Claude Code, Codex,
Gemini CLI, Cline), stores normalized usage in an embedded SQLite database, and turns it
into token/cost reports, effectiveness and analyze diagnostics, and a self-contained HTML
dashboard. The local analysis is fully offline: no telemetry, no network calls, prompts
are never read. The optional team server (serve/sync) is self-hosted and opt-in.`,
		Example: `  assaio-agent demo            # the full reports on bundled sample data
  assaio-agent backfill        # import all historical local logs
  assaio-agent report --since 7d
  assaio-agent doctor          # what tools were detected`,
		Version:       version.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String("config", "", "config file path")
	root.AddCommand(newVersionCmd(), newDemoCmd(), newReportCmd(), newEffectivenessCmd(), newAnalyzeCmd(), newCheckCmd(),
		newDashboardCmd(), newBackfillCmd(), newDoctorCmd(), newStatusCmd(), newClearCmd(), newConfigCmd(), newPluginsCmd(),
		newMetricsCmd(), newServeCmd(), newSyncCmd())
	return root
}

func ensureParent(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o750)
}

func loadConfig(cmd *cobra.Command) (config.Config, error) {
	path, _ := cmd.Flags().GetString("config")
	if path == "" {
		p, err := paths.ConfigPath()
		if err != nil {
			return config.Config{}, err
		}
		path = p
	}
	return config.Load(path)
}
