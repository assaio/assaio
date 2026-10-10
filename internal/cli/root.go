// Package cli wires the assaio-agent command-line interface.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/paths"
	"github.com/assaio/assaio/internal/version"
)

// NewRootCmd builds the assaio-agent root command with all subcommands attached.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "assaio-agent",
		Short: "AI adoption, estimated costs, and local delivery evidence for engineering teams",
		Long: `assaio-agent reads the local session logs of AI coding tools (` + strings.Join(parser.Tools(), ", ") + `),
stores normalized usage in embedded SQLite, and reports observed sessions, active days,
tool/project breadth, API-equivalent estimated costs, and recorded output. Antigravity supplies
activity only and is excluded from token/cost figures. Reports and diagnostics include a
self-contained HTML dashboard. Local analysis runs offline with no telemetry; prompt and
response content is never extracted. The optional self-hosted team usage MVP (serve/sync)
is opt-in. evidence --github explicitly uses your own gh for bounded PR, review, latest-head
check, and separate historical suite/run observations. Delivery evidence stays local and
in memory for one command, outside sync and dashboards. It does not establish causal AI
session outcomes, productivity, or employee-wide adoption percentages.`,
		Example: `  assaio-agent demo            # the full reports on bundled sample data
  assaio-agent backfill        # import all historical local logs
  assaio-agent report --since 7d
  assaio-agent doctor          # what tools were detected`,
		Version:       version.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String("config", "", "config file path")
	addCommands(root, newVersionCmd(), newDemoCmd(), newReportCmd(), newEffectivenessCmd(), newAnalyzeCmd(), newCheckCmd(),
		newInitCmd(), newDashboardCmd(), newBackfillCmd(), newDoctorCmd(), newStatusCmd(), newClearCmd(), newCompactCmd(),
		newConfigCmd(), newPluginsCmd(), newMetricsCmd(), newServeCmd(), newSyncCmd(), newSurvivalCmd(),
		newStatuslineCmd(), newExplainCmd(), newMarkCmd(), newSignalsCmd(), newReconcileCmd(),
		newDigestCmd(), newDocsCmd(), newShareCmd(), newRecommendCmd(), newRepriceCmd(),
		newEvidenceCmd(), newReposCmd())
	return root
}

func ensureParent(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o750)
}

// loadConfigRaw resolves the config path and loads the merged config WITHOUT validating it.
func loadConfigRaw(cmd *cobra.Command) (config.Config, error) {
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

// loadConfig loads the merged config and REJECTS an invalid one, so a typo in an
// honesty-relevant setting (a misspelled pricing.mode, a duplicate plugin name) can't
// silently apply to a report. Used by every reporting and serving command.
func loadConfig(cmd *cobra.Command) (config.Config, error) {
	cfg, err := loadConfigRaw(cmd)
	if err != nil {
		return config.Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

// loadConfigLenient loads the merged config but only WARNS on an invalid one, returning it
// anyway. The diagnostic and import commands (doctor, backfill) use it: they do not consume
// the validated report settings and must keep working on a broken config so the user can
// diagnose it and still import data. The config command loads raw and warns itself.
func loadConfigLenient(cmd *cobra.Command) (config.Config, error) {
	cfg, err := loadConfigRaw(cmd)
	if err != nil {
		return config.Config{}, err
	}
	if verr := cfg.Validate(); verr != nil {
		cmd.PrintErrf("warning: %v (ignored for this command)\n", verr)
	}
	return cfg, nil
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
