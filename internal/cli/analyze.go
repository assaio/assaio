package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/label"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

// analyzeRecentWindow is the recent-vs-prior window validators use for trend and
// staleness signals, matching the status dashboard's window.
const analyzeRecentWindow = 7 * 24 * time.Hour

func newAnalyzeCmd() *cobra.Command {
	var since, format string
	var list bool
	c := &cobra.Command{
		Use:   "analyze [name...]",
		Short: "Run metric validators and print each one's directional text report",
		Args: func(_ *cobra.Command, args []string) error {
			if list && len(args) > 0 {
				return fmt.Errorf("--list takes no positional arguments, got %v", args)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if list {
				return runAnalyzeList(cmd)
			}
			return runAnalyze(cmd, args, &since, format)
		},
	}
	c.Flags().StringVar(&since, "since", "30d", "time window, e.g. 7d")
	c.Flags().StringVar(&format, "format", "text", "output format: text|json")
	c.Flags().BoolVar(&list, "list", false, "list registered validators and exit")
	addLabelFlags(c)
	addDBFlag(c)
	return c
}

// runAnalyzeList prints every runnable metric: registered validators, then configured
// exec metric plugins -- listed from config only, never executed, so --list stays free
// of side effects.
func runAnalyzeList(cmd *cobra.Command) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	for _, v := range analyze.Validators() {
		cmd.Printf("%-14s %-32s %s\n", v.Name(), v.Title(), v.Describe())
	}
	for _, pc := range sortedPluginConfigs(cfg.Metrics) {
		cmd.Printf("%-14s %-32s %s\n", metricPluginPrefix+pc.Name, "(exec metric plugin)", pc.Command)
	}
	return nil
}

func runAnalyze(cmd *cobra.Command, names []string, since *string, format string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}
	if err := validateAnalysisNames(names, cfg.Metrics); err != nil {
		return err
	}
	// analyze has its own --format vocabulary (text|json), so only since comes from
	// config; the config-driven table|json|csv format default is report/effectiveness's.
	if !cmd.Flags().Changed("since") {
		*since = cfg.Since
	}
	start, err := parseSinceAt(*since, time.Now())
	if err != nil {
		return err
	}
	st, err := openReportStore(cmd)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	n, err := st.Count(cmd.Context())
	if err != nil {
		return err
	}
	if n == 0 && format == "text" {
		return emptyStatusHint(cmd)
	}
	filter, err := labelFilterFrom(cmd)
	if err != nil {
		return err
	}

	in, err := buildAnalyzeInputFiltered(cmd, st, start, filter)
	if err != nil {
		return err
	}
	in.PlanMonthlyCost = cfg.Pricing.MonthlySubscriptionCost
	validators := analyze.Validators()
	if !filter.Empty() {
		if validators, err = narrowAnalysis(cmd, st, start, filter, &in, names, format); err != nil {
			return err
		}
	}
	results, err := collectAnalysisResults(cmd, names, cfg.Metrics, &in, validators)
	if err != nil {
		return err
	}
	return renderAnalyzeResults(cmd, results, format)
}

// narrowAnalysis states what a label filter selected and returns the validators that can
// honestly answer for a subset of the window.
func narrowAnalysis(cmd *cobra.Command, st *store.Store, start time.Time, filter store.LabelFilter,
	in *analyze.Input, names []string, format string,
) ([]analyze.Validator, error) {
	kept, skipped := narrowableValidators()
	for _, name := range names {
		if v, ok := analyze.Get(name); ok && !analyze.ProjectScoped(v) {
			return nil, fmt.Errorf("%s describes the whole window and cannot be read per label; run it without --%s/--%s/--%s",
				name, label.Task, label.Outcome, label.Difficulty)
		}
	}
	if format != "text" {
		return kept, nil
	}
	all, err := st.Sessions(cmd.Context(), start)
	if err != nil {
		return nil, err
	}
	return kept, renderNarrowing(cmd, filter, len(in.Sessions), len(all), skipped)
}

func buildAnalyzeInput(cmd *cobra.Command, st *store.Store, start time.Time) (analyze.Input, error) {
	return buildAnalyzeInputFiltered(cmd, st, start, store.LabelFilter{})
}

// buildAnalyzeInputFiltered is buildAnalyzeInput restricted to the sessions filter selects.
// Every one of the five queries below takes the filter: narrowing only the usage rows would
// state one subset's figures beside the whole window's delegation, attribution and turn mix.
func buildAnalyzeInputFiltered(cmd *cobra.Command, st *store.Store, start time.Time, filter store.LabelFilter) (analyze.Input, error) {
	usageRows, err := st.UsageFiltered(cmd.Context(), start, filter)
	if err != nil {
		return analyze.Input{}, err
	}
	sessionRows, err := st.SessionsFiltered(cmd.Context(), start, filter)
	if err != nil {
		return analyze.Input{}, err
	}
	sub, total, err := st.DelegationFiltered(cmd.Context(), start, filter)
	if err != nil {
		return analyze.Input{}, err
	}
	table, err := pricing.Load()
	if err != nil {
		return analyze.Input{}, err
	}
	turns, err := st.TurnSizingFiltered(cmd.Context(), start, analyze.RightSizeSmallOutput, filter)
	if err != nil {
		return analyze.Input{}, err
	}
	skills, agents, err := st.AttributionFiltered(cmd.Context(), start, filter)
	if err != nil {
		return analyze.Input{}, err
	}
	misses, err := st.CacheMissesFiltered(cmd.Context(), start, filter)
	if err != nil {
		return analyze.Input{}, err
	}
	in := analyze.BuildInput(usageRows, sessionRows, table, time.Now(), analyzeRecentWindow, analyze.Delegation{Sub: sub, Total: total})
	in.TurnSizing = turns
	in.Skills, in.Agents = skills, agents
	in.CacheMisses = misses
	// Provenance is diagnostic, not a figure: a store that cannot answer leaves it unknown
	// rather than failing an analysis that is otherwise complete.
	in.Ingested, in.ParsedBy, _ = st.Provenance(cmd.Context())
	return in, nil
}

func renderAnalyzeResults(cmd *cobra.Command, results []analyze.Result, format string) error {
	switch format {
	case "text":
		return renderAnalyzeText(cmd, results)
	case "json":
		return renderAnalyzeJSON(cmd, results)
	default:
		return fmt.Errorf("unknown format %q (want text|json)", format)
	}
}

func renderAnalyzeText(cmd *cobra.Command, results []analyze.Result) error {
	for i := range results {
		if err := analyze.RenderResultText(cmd.OutOrStdout(), results[i]); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), report.CostEstimateDisclosure)
	return err
}

// renderAnalyzeJSON writes the Results as a JSON array. Each Result already carries its
// own Name, so no wrapping map is needed to identify which entry is which.
func renderAnalyzeJSON(cmd *cobra.Command, results []analyze.Result) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(results)
}
