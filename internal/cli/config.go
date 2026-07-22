package cli

import (
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show the effective configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfigRaw(cmd)
			if err != nil {
				return err
			}
			path, _, _ := configPath(cmd)
			cmd.Printf("config file: %s\nsince:  %s\nformat: %s\n", path, cfg.Since, cfg.Format)
			if cfg.Pricing.Configured() {
				cmd.Printf("pricing: mode=%q effective_per_token=%g monthly_subscription_cost=%g\n",
					cfg.Pricing.Mode, cfg.Pricing.EffectivePerToken, cfg.Pricing.MonthlySubscriptionCost)
			}
			if err := cfg.Validate(); err != nil {
				cmd.Printf("warning: %v\n", err)
			}
			return nil
		},
	}
}
