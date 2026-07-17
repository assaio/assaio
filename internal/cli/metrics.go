package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/plugin"
)

func newMetricsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "metrics",
		Short: "Inspect and validate configured exec metric plugins",
	}
	c.AddCommand(newMetricsListCmd(), newMetricsVerifyCmd())
	return c
}

func newMetricsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured metric plugins and their resolved command paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			if len(cfg.Metrics) == 0 {
				cmd.Println("no metric plugins configured")
				return nil
			}
			for _, pc := range sortedMetricConfigs(cfg.Metrics) {
				printPluginListing(cmd, pc)
			}
			return nil
		},
	}
}

func newMetricsVerifyCmd() *cobra.Command {
	var since string
	c := &cobra.Command{
		Use:   "verify <name>",
		Short: "Run a configured metric plugin on real store data and report protocol conformance, without storing anything",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMetricsVerify(cmd, args[0], since)
		},
	}
	c.Flags().StringVar(&since, "since", "30d", "time window, e.g. 7d")
	addDBFlag(c)
	return c
}

func runMetricsVerify(cmd *cobra.Command, name, since string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	pc, ok := findMetricConfig(cfg.Metrics, name)
	if !ok {
		return fmt.Errorf("no metric plugin named %q in config", name)
	}
	resolved, err := plugin.Resolve(pc)
	if err != nil {
		return err
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
	in, err := buildAnalyzeInput(cmd, st, start)
	if err != nil {
		return err
	}

	res, violations, err := plugin.VerifyMetric(cmd.Context(), resolved, &in)
	if err != nil {
		printMetricVerifyFailure(cmd, name, violations, err)
		return err
	}
	cmd.Printf("%s: handshake OK\n", name)
	cmd.Println("result: VALID")
	return analyze.RenderResultText(cmd.OutOrStdout(), res)
}

// printMetricVerifyFailure reports what broke before the error propagates: the contract
// violations when the result document failed validation, or the run error alone for
// upstream failures (handshake, timeout, non-zero exit).
func printMetricVerifyFailure(cmd *cobra.Command, name string, violations []string, err error) {
	cmd.Printf("%s: FAILED %v\n", name, err)
	if len(violations) == 0 {
		return
	}
	cmd.Println("violations:")
	for _, v := range violations {
		cmd.Printf("  - %s\n", v)
	}
}
