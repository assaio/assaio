package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
)

func newCheckCmd() *cobra.Command {
	var since string
	var maxTokens int64
	var maxCost float64
	c := &cobra.Command{
		Use:   "check",
		Short: "Exit non-zero when usage exceeds a budget or a rule plugin raises an error (CI/pre-push gate)",
		Long: `Roll the window's usage up and compare it against optional budgets, exiting non-zero
when one is exceeded -- a CI gate or pre-push hook. Token budgets are the honest default:
tokens are physical and plan-independent. A --max-cost budget is allowed but gates on the
API-equivalent estimate, not your actual spend (subscriptions bill a flat rate).

Configured rule plugins (rules: in config.yaml) run here too: each reads this window's
validator verdicts and emits alerts. An "error" alert fails the gate, and so does a rule
that could not be evaluated -- a gate that did not run is not a gate that passed.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCheck(cmd, &since, budget{MaxTokens: maxTokens, MaxCost: maxCost})
		},
	}
	c.Flags().StringVar(&since, "since", "30d", "time window, e.g. 7d")
	c.Flags().Int64Var(&maxTokens, "max-tokens", 0, "fail if total tokens exceed this budget (0 = unset)")
	c.Flags().Float64Var(&maxCost, "max-cost", 0, "fail if API-equivalent cost exceeds this budget in dollars (0 = unset)")
	addDBFlag(c)
	return c
}

func runCheck(cmd *cobra.Command, since *string, b budget) error {
	if err := resolveSince(cmd, since); err != nil {
		return err
	}
	start, err := parseSinceAt(*since, time.Now())
	if err != nil {
		return err
	}
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	st, err := openReportStore(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	rows, err := st.Usage(cmd.Context(), start)
	if err != nil {
		return err
	}
	table, err := pricing.Load()
	if err != nil {
		return err
	}
	totals := sumCheckTotals(report.Build(rows, table))
	breaches := evaluateBudget(totals, b)
	if err := renderCheck(cmd, *since, totals, b, cfg.Pricing); err != nil {
		return err
	}
	ruleFailures, err := gateOnRules(cmd, &cfg, st, start)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), report.CostEstimateDisclosure); err != nil {
		return err
	}
	if len(breaches) > 0 {
		return fmt.Errorf("budget exceeded: %s", strings.Join(breaches, "; "))
	}
	if len(ruleFailures) > 0 {
		return fmt.Errorf("rule gate failed: %s", strings.Join(ruleFailures, "; "))
	}
	return nil
}
