package plugin

// The mapping from core types onto the wire envelope, kept apart from the envelope's own
// shape so the protocol's documented surface reads as one file.

import (
	"encoding/json"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

func (mi *metricInput) marshal() ([]byte, error) { return json.Marshal(mi) }

func buildMetricInput(in *analyze.Input) metricInput {
	return metricInput{
		Version:    metricInputVersion,
		Now:        in.Now,
		RecentDays: int(in.Recent / (24 * time.Hour)),
		Usage:      usageWire(in.Usage),
		Sessions:   sessionWire(in.Sessions),
		Delegation: metricDelegation{Sub: in.Delegation.Sub, Total: in.Delegation.Total},
		ByModel:    modelWire(in.ByModel),
		ByProject:  projectWire(in.ByProject),
		Totals: metricTotals{
			Tokens: in.Totals.Tokens, Input: in.Totals.Input, Output: in.Totals.Output,
			CacheRead: in.Totals.CacheRead, CacheWrite: in.Totals.CacheWrite,
			Lines: in.Totals.Lines, Cost: in.Totals.Cost, Priced: in.Totals.Priced,
			CacheEfficiency: in.Totals.CacheEfficiency,
		},
		Prices:          pricesWire(in.Usage, in.Prices),
		Answers:         answersWire(in.Usage, in.Sessions),
		WindowStart:     in.WindowStart,
		PlanMonthlyCost: in.PlanMonthlyCost,
		Skills:          attributionWire(in.Skills),
		Agents:          attributionWire(in.Agents),
		TurnSizing:      turnSizingWire(in.TurnSizing),
		CacheMisses:     cacheMissWire(in.CacheMisses),
	}
}

// answersWire declares what every tool in this window can record. Only the tools actually
// present are sent: a plugin needs the capability of the data it was handed, and shipping
// the whole matrix would make the envelope a second publication of internal/parser.
func answersWire(usage []store.UsageRow, sessions []store.SessionRow) map[string][]string {
	out := make(map[string][]string)
	for i := range usage {
		out[usage[i].Tool] = nil
	}
	for i := range sessions {
		out[sessions[i].Tool] = nil
	}
	for tool := range out {
		out[tool] = parser.SignalsAnsweredBy(tool)
	}
	return out
}

func usageWire(rows []store.UsageRow) []metricUsageRow {
	out := make([]metricUsageRow, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		out = append(out, metricUsageRow{
			Day: r.Day, Tool: r.Tool, Model: r.Model, Project: r.Project,
			Entrypoint: r.Entrypoint, Member: r.Member, Granularity: r.Granularity,
			In: r.In, Out: r.Out, CacheRead: r.CacheRead, CacheWrite: r.CacheWrite,
			CacheWrite1h: r.CacheWrite1h,
			Reasoning:    r.Reasoning, LinesAdded: r.LinesAdded, LinesRemoved: r.LinesRemoved,
			Edits: r.Edits, ToolCalls: r.ToolCalls, Rejected: r.Rejected,
			Compactions: r.Compactions, ReworkLines: r.ReworkLines,
		})
	}
	return out
}

func sessionWire(rows []store.SessionRow) []metricSessionRow {
	out := make([]metricSessionRow, 0, len(rows))
	for i := range rows {
		s := &rows[i]
		out = append(out, metricSessionRow{
			SessionID: s.SessionID, Project: s.Project, Tool: s.Tool, Model: s.Model,
			Member: s.Member, FirstTs: s.FirstTs, LastTs: s.LastTs, Turns: s.Turns,
			OutputTokens: s.OutputTokens, PeakContextTokens: s.PeakContextTokens,
			Edits: s.Edits, Compactions: s.Compactions, ActiveMinutes: s.ActiveMinutes,
		})
	}
	return out
}

func modelWire(stats []analyze.ModelStat) []metricModelStat {
	out := make([]metricModelStat, 0, len(stats))
	for _, m := range stats {
		out = append(out, metricModelStat{
			Model: m.Model, Tier: m.Tier, Tokens: m.Tokens, Input: m.Input,
			Output: m.Output, CacheRead: m.CacheRead, CacheWrite: m.CacheWrite,
			Lines: m.Lines, Cost: m.Cost, Priced: m.Priced, TokenShare: m.TokenShare,
		})
	}
	return out
}

func projectWire(stats []analyze.ProjectStat) []metricProjectStat {
	out := make([]metricProjectStat, 0, len(stats))
	for _, p := range stats {
		out = append(out, metricProjectStat{
			Project: p.Project, Lines: p.Lines, Cost: p.Cost, Priced: p.Priced,
			TokenShare: p.TokenShare,
		})
	}
	return out
}

// pricesWire resolves a price for every distinct model present in the window's usage,
// exact name first then pricing.NormalizeModel -- the same lookup CostTokens applies --
// and emits it under the raw model name. An unpriced model is absent, never a zero
// price.
func pricesWire(rows []store.UsageRow, prices pricing.Table) map[string]metricPrice {
	out := make(map[string]metricPrice)
	for i := range rows {
		model := rows[i].Model
		if _, done := out[model]; done {
			continue
		}
		p, ok := prices[model]
		if !ok {
			p, ok = prices[pricing.NormalizeModel(model)]
		}
		if !ok {
			continue
		}
		out[model] = metricPrice{
			Input: p.Input, Output: p.Output, CacheRead: p.CacheRead,
			CacheWrite: p.CacheWrite, CacheWrite1h: p.CacheWrite1h,
		}
	}
	return out
}
