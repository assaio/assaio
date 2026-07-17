package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/config"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
)

func newCheckCmd() *cobra.Command {
	var since string
	var maxTokens int64
	var maxCost float64
	c := &cobra.Command{
		Use:   "check",
		Short: "Exit non-zero when usage exceeds a token or API-equivalent cost budget (CI/pre-push gate)",
		Long: `Roll the window's usage up and compare it against optional budgets, exiting non-zero
when one is exceeded -- a CI gate or pre-push hook. Token budgets are the honest default:
tokens are physical and plan-independent. A --max-cost budget is allowed but gates on the
API-equivalent estimate, not your actual spend (subscriptions bill a flat rate).`,
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

// budget holds the optional ceilings check gates on; zero means no budget for that axis.
type budget struct {
	MaxTokens int64
	MaxCost   float64
}

func (b budget) empty() bool { return b.MaxTokens <= 0 && b.MaxCost <= 0 }

// checkTotals rolls the window's usage up to the two axes a budget gates on.
type checkTotals struct {
	Tokens      int64
	Cost        float64
	HasUnpriced bool
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
	if len(breaches) > 0 {
		return fmt.Errorf("budget exceeded: %s", strings.Join(breaches, "; "))
	}
	return nil
}

// sumCheckTotals adds every priced row's cost and every row's tokens (all token types)
// into the window totals; HasUnpriced carries the usual "cost excludes unpriced usage".
func sumCheckTotals(rows []report.Row) checkTotals {
	var t checkTotals
	for i := range rows {
		r := &rows[i]
		t.Tokens += r.In + r.Out + r.CacheRead + r.CacheWrite + r.Reasoning
		if r.Priced {
			t.Cost += *r.Cost
		}
		if r.HasUnpriced {
			t.HasUnpriced = true
		}
	}
	return t
}

// evaluateBudget returns one breach message per exceeded axis; empty means within budget.
func evaluateBudget(t checkTotals, b budget) []string {
	var breaches []string
	if b.MaxTokens > 0 && t.Tokens > b.MaxTokens {
		breaches = append(breaches, fmt.Sprintf("tokens %d > %d", t.Tokens, b.MaxTokens))
	}
	if b.MaxCost > 0 && t.Cost > b.MaxCost {
		breaches = append(breaches, fmt.Sprintf("API-equivalent cost $%.2f > $%.2f", t.Cost, b.MaxCost))
	}
	return breaches
}

// lineWriter collects sequential write errors so renderCheck can emit many lines and
// report only the first failure, keeping the io.Writer error contract without a check
// after every line (Rob Pike's errWriter pattern).
type lineWriter struct {
	w   io.Writer
	err error
}

func (lw *lineWriter) printf(format string, a ...any) {
	if lw.err != nil {
		return
	}
	_, lw.err = fmt.Fprintf(lw.w, format, a...)
}

func (lw *lineWriter) println(s string) {
	if lw.err != nil {
		return
	}
	_, lw.err = fmt.Fprintln(lw.w, s)
}

func renderCheck(cmd *cobra.Command, since string, t checkTotals, b budget, p config.Pricing) error {
	lw := &lineWriter{w: cmd.OutOrStdout()}
	lw.printf("budget check · last %s\n", since)
	lw.printf("  total tokens: %d\n", t.Tokens)
	cost := fmt.Sprintf("$%.2f", t.Cost)
	if t.HasUnpriced {
		cost += "*"
	}
	lw.printf("  total cost:   %s (API-equivalent estimate)\n", cost)
	if t.HasUnpriced {
		lw.println("  * includes unpriced usage excluded from cost")
	}
	if line, ok := effectiveBasisLine(p, t.Tokens); ok {
		lw.printf("  %s\n", line)
	}
	lw.println("")
	writeBudgetAxis(lw, "token", b.MaxTokens > 0, t.Tokens <= b.MaxTokens, fmt.Sprintf("%d / %d", t.Tokens, b.MaxTokens))
	writeBudgetAxis(lw, "cost", b.MaxCost > 0, t.Cost <= b.MaxCost, fmt.Sprintf("$%.2f / $%.2f (API-equivalent)", t.Cost, b.MaxCost))
	if b.empty() {
		lw.println("  no budget set -- pass --max-tokens or --max-cost to gate.")
	}
	lw.println("")
	lw.println(report.CostEstimateDisclosure)
	return lw.err
}

// writeBudgetAxis prints one budget line, or nothing when that axis has no budget set.
func writeBudgetAxis(lw *lineWriter, label string, set, ok bool, detail string) {
	if !set {
		return
	}
	verdict := "OK"
	if !ok {
		verdict = "EXCEEDED"
	}
	lw.printf("  %s budget: %s  %s\n", label, detail, verdict)
}

// effectiveBasisLine renders the user's own cost basis when they configured one -- the
// honest counter to the API-equivalent estimate shown above it.
func effectiveBasisLine(p config.Pricing, tokens int64) (string, bool) {
	if !p.Configured() {
		return "", false
	}
	if p.EffectivePerToken > 0 {
		cost, _ := p.EffectiveWindowCost(tokens)
		return fmt.Sprintf("your basis: effective ~$%.2f this window at $%g/token (not the API figure above)", cost, p.EffectivePerToken), true
	}
	if p.MonthlySubscriptionCost > 0 {
		return fmt.Sprintf("your basis: ~$%.2f/mo flat subscription -- the $ above is an API-equivalent estimate, not this bill", p.MonthlySubscriptionCost), true
	}
	return "your basis: flat subscription -- the $ above is an API-equivalent estimate, not your spend", true
}
