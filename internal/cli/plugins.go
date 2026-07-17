package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/plugin"
)

func newPluginsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "plugins",
		Short: "Inspect and validate configured exec plugins",
	}
	c.AddCommand(newPluginsListCmd(), newPluginsVerifyCmd())
	return c
}

func newPluginsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured plugins and their resolved command paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			if len(cfg.Plugins) == 0 {
				cmd.Println("no plugins configured")
				return nil
			}
			for _, pc := range cfg.Plugins {
				printPluginListing(cmd, pc)
			}
			return nil
		},
	}
}

func printPluginListing(cmd *cobra.Command, pc config.PluginConfig) {
	resolved, err := plugin.Resolve(pc)
	if err != nil {
		cmd.Printf("%-16s  ERROR %v\n", pc.Name, err)
		return
	}
	cmd.Printf("%-16s  %s  (timeout %s)\n", pc.Name, resolved.Command, resolved.Timeout)
}

func newPluginsVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify <name>",
		Short: "Run a configured plugin and report protocol conformance, without storing results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginsVerify(cmd, args[0])
		},
	}
}

func runPluginsVerify(cmd *cobra.Command, name string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	pc, err := findPluginConfig(cfg.Plugins, name)
	if err != nil {
		return err
	}
	resolved, err := plugin.Resolve(pc)
	if err != nil {
		return err
	}
	_, violations, stats, err := plugin.Verify(cmd.Context(), resolved)
	if err != nil {
		cmd.Printf("%s: FAILED %v\n", name, err)
		return err
	}
	printVerifyReport(cmd, name, violations, stats)
	return nil
}

func findPluginConfig(plugins []config.PluginConfig, name string) (config.PluginConfig, error) {
	for _, pc := range plugins {
		if pc.Name == name {
			return pc, nil
		}
	}
	return config.PluginConfig{}, fmt.Errorf("no plugin named %q in config", name)
}

func printVerifyReport(cmd *cobra.Command, name string, violations []plugin.Violation, stats plugin.Stats) {
	cmd.Printf("%s: handshake OK\n", name)
	cmd.Printf("records ok: %d\n", stats.Records)
	cmd.Printf("skipped:    %d\n", stats.Skipped)
	if len(violations) == 0 {
		return
	}
	cmd.Println("violations:")
	for _, v := range violations {
		cmd.Printf("  line %d: %s\n", v.Line, v.Reason)
	}
}
