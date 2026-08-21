package cli

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// The Input every analysis surface is built from, kept apart from the command that made it
// first. Six other commands call this builder -- analyze, dashboard, check, digest, share and
// recommend -- and hiding a shared constructor inside one command's file is what produced the
// `metrics verify` plan-price bug v0.22 fixed: the second caller got a different Input from
// the first and nothing said so.

// buildAnalyzeInputFiltered is buildAnalyzeInput restricted to the sessions filter selects.
// Every one of the five queries below takes the filter: narrowing only the usage rows would
// state one subset's figures beside the whole window's delegation, attribution and turn mix.
//
// The plan price is set here rather than by each caller: a field every caller has to remember is
// a field one of them will forget.
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
	in.WindowStart = start
	in.TurnSizing = turns
	in.Skills, in.Agents = skills, agents
	in.CacheMisses = misses
	if err := readSequenceFacts(cmd, st, start, &in, sessionRows); err != nil {
		return analyze.Input{}, err
	}
	cfg, err := loadConfig(cmd)
	if err != nil {
		return analyze.Input{}, err
	}
	in.PlanMonthlyCost = cfg.Pricing.MonthlySubscriptionCost
	// Provenance is diagnostic, not a figure: a store that cannot answer leaves it unknown
	// rather than failing an analysis that is otherwise complete.
	in.Ingested, in.ParsedBy, _ = st.Provenance(cmd.Context())
	return in, nil
}
