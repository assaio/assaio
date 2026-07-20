// Package cli wires the assaio-agent command-line interface.
package cli

import (
	"fmt"
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
	path, explicit, err := configPath(cmd)
	if err != nil {
		return config.Config{}, err
	}
	// A missing default path is fine (built-in defaults apply); an explicitly passed
	// --config that does not exist is a user error, not something to silently ignore.
	if explicit {
		if _, err := os.Stat(path); err != nil {
			return config.Config{}, fmt.Errorf("config file %s: %w", path, err)
		}
	}
	return config.Load(path)
}

// configPath returns the config path in effect and whether it was set explicitly via
// --config. An explicit path is taken verbatim; otherwise the default location is used.
func configPath(cmd *cobra.Command) (path string, explicit bool, err error) {
	if p, _ := cmd.Flags().GetString("config"); p != "" {
		return p, true, nil
	}
	p, err := paths.ConfigPath()
	if err != nil {
		return "", false, err
	}
	return p, false, nil
}
