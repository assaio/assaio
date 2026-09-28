package analyze

import (
	"sort"
	"time"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/report"
	"github.com/assaio/assaio/internal/store"
)

// BuildInput assembles a validator-ready Input from raw store data. ByModel, ByProject,
// and Totals are computed once here, so every construction site (CLI, server, the
// dashboard's project drill-down, tests) must call this rather than building Input by
// hand -- otherwise the prepared views are left empty or, worse, stale against a
// hand-edited Usage.
func BuildInput(usage []store.UsageRow, sessions []store.SessionRow, prices pricing.Table, now time.Time, recent time.Duration, delegation Delegation) Input {
	credit := report.NewLineCredit(usage, prices)
	totals := buildTotals(usage, prices, credit)
	return Input{
		Usage: usage, Sessions: sessions, Prices: prices, Now: now, Recent: recent, Delegation: delegation,
		ByModel:   buildModelStats(usage, prices, totals.Tokens, credit),
		ByProject: buildProjectStats(usage, prices, totals.Tokens, credit),
		Totals:    totals,
	}
}

// recordedLines is r's added lines when its source records them, and nothing otherwise: a count a
// source cannot keep contributes nothing to a figure, even when a row carries one (ADR 0011).
func recordedLines(r *store.UsageRow) int64 {
	if !parser.RecordsLines(r.Tool) {
		return 0
	}
	return r.LinesAdded
}

// usageTokens is one row's billable counts in the shape pricing takes, so the three build
// passes cannot drift apart on which of them a cost is computed from.
func usageTokens(r *store.UsageRow) pricing.Tokens {
	return pricing.Tokens{
		In: r.In, Out: r.Out, CacheWrite: r.CacheWrite, CacheRead: r.CacheRead,
		CacheWrite1h: r.CacheWrite1h,
	}
}

// buildTotals sums rows into Totals in one pass, folding priced cost only from usage
// whose model resolves in prices and flagging Priced false the moment one does not.
// Priced starts false on zero rows (an honest "nothing to price" rather than a vacuous
// true), true as soon as there is at least one row, then back to false the moment any
// row's model is unpriced.
func buildTotals(rows []store.UsageRow, prices pricing.Table, credit report.LineCredit) Totals {
	var t Totals
	priced := len(rows) > 0
	for i := range rows {
		r := &rows[i]
		t.Input += r.In
		t.Output += r.Out
		t.CacheRead += r.CacheRead
		t.CacheWrite += r.CacheWrite
		t.Lines += recordedLines(r)
		cost, ok := prices.CostTokens(r.Model, usageTokens(r))
		credit.Fold(&t.LineRate, r, cost, ok, false)
		if ok {
			if t.Cost == nil {
				zero := 0.0
				t.Cost = &zero
			}
			*t.Cost += cost
		} else {
			priced = false
		}
	}
	// ReasoningTokens is a subset of OutputTokens (usage.Record); adding it double-counts.
	t.Tokens = t.Input + t.Output + t.CacheRead + t.CacheWrite
	t.Priced = priced
	t.CacheEfficiency = fracOf(t.CacheRead, t.CacheRead+t.Input)
	return t
}

// buildModelStats groups rows by model, classifies each model's tier once via modelTier,
// and prices each group's summed tokens once via prices -- a single model's price status
// is uniform across every row bearing its name, so pricing after summing (rather than
// per row) is exact. totalTokens is Totals.Tokens, the shared TokenShare denominator.
func buildModelStats(rows []store.UsageRow, prices pricing.Table, totalTokens int64, credit report.LineCredit) []ModelStat {
	byModel := make(map[string]*ModelStat)
	order := make([]string, 0)
	for i := range rows {
		r := &rows[i]
		m, ok := byModel[r.Model]
		if !ok {
			m = &ModelStat{Model: r.Model}
			byModel[r.Model] = m
			order = append(order, r.Model)
		}
		m.Input += r.In
		m.Output += r.Out
		m.CacheRead += r.CacheRead
		m.CacheWrite += r.CacheWrite
		m.CacheWrite1h += r.CacheWrite1h
		m.Lines += recordedLines(r)
		m.Tokens += r.In + r.Out + r.CacheRead + r.CacheWrite
		cost, ok := prices.CostTokens(r.Model, usageTokens(r))
		credit.Fold(&m.LineRate, r, cost, ok, true)
	}
	sort.Strings(order)

	stats := make([]ModelStat, len(order))
	for i, name := range order {
		m := byModel[name]
		m.Tier = modelTier(name, prices)
		cost, ok := prices.CostTokens(name, pricing.Tokens{In: m.Input, Out: m.Output, CacheWrite: m.CacheWrite, CacheRead: m.CacheRead, CacheWrite1h: m.CacheWrite1h})
		m.Priced = ok
		if ok {
			m.Cost = &cost
		}
		m.TokenShare = fracOf(m.Tokens, totalTokens)
		stats[i] = *m
	}
	sort.SliceStable(stats, func(i, j int) bool { return stats[i].Tokens > stats[j].Tokens })
	return stats
}

// projectAgg accumulates one project's ProjectStat plus the running token sum
// ProjectStat itself has no field for -- TokenShare's numerator, discarded once computed.
type projectAgg struct {
	stat     ProjectStat
	tokens   int64
	unpriced bool
}

// buildProjectStats groups rows by project, summing cost per row (unlike buildModelStats,
// a project's rows can mix priced and unpriced models, so pricing must fold row by row).
// totalTokens is Totals.Tokens, the shared TokenShare denominator.
func buildProjectStats(rows []store.UsageRow, prices pricing.Table, totalTokens int64, credit report.LineCredit) []ProjectStat {
	byProject := make(map[string]*projectAgg)
	order := make([]string, 0)
	for i := range rows {
		r := &rows[i]
		a, ok := byProject[r.Project]
		if !ok {
			a = &projectAgg{stat: ProjectStat{Project: r.Project}}
			byProject[r.Project] = a
			order = append(order, r.Project)
		}
		a.stat.Lines += recordedLines(r)
		a.tokens += r.In + r.Out + r.CacheRead + r.CacheWrite
		cost, ok := prices.CostTokens(r.Model, usageTokens(r))
		credit.Fold(&a.stat.LineRate, r, cost, ok, false)
		if ok {
			if a.stat.Cost == nil {
				zero := 0.0
				a.stat.Cost = &zero
			}
			*a.stat.Cost += cost
		} else {
			a.unpriced = true
		}
	}
	sort.Strings(order)

	stats := make([]ProjectStat, len(order))
	for i, name := range order {
		a := byProject[name]
		a.stat.Priced = !a.unpriced
		a.stat.TokenShare = fracOf(a.tokens, totalTokens)
		stats[i] = a.stat
	}
	sort.SliceStable(stats, func(i, j int) bool { return stats[i].Lines > stats[j].Lines })
	return stats
}
