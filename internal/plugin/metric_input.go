package plugin

import (
	"encoding/json"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// metricInputVersion is the stdin envelope's protocol version, carried in the
// assaio_metric_input key.
const metricInputVersion = 1

// metricInput is the wire envelope a metric plugin reads on stdin: analyze.Input mapped
// onto explicit wire types so a core refactor never silently changes the protocol -- the
// same decoupling wireRecord gives the parser protocol (ADR 0003/0004). Field names are
// camelCase to mirror the public `analyze --format json` Result shape; only the version
// keys stay snake_case, matching assaio_plugin.
type metricInput struct {
	Version    int                    `json:"assaio_metric_input"`
	Now        time.Time              `json:"now"`
	RecentDays int                    `json:"recentDays"`
	Usage      []metricUsageRow       `json:"usage"`
	Sessions   []metricSessionRow     `json:"sessions"`
	Delegation metricDelegation       `json:"delegation"`
	ByModel    []metricModelStat      `json:"byModel"`
	ByProject  []metricProjectStat    `json:"byProject"`
	Totals     metricTotals           `json:"totals"`
	Prices     map[string]metricPrice `json:"prices"`
}

type metricUsageRow struct {
	Day          string `json:"day"`
	Tool         string `json:"tool"`
	Model        string `json:"model"`
	Project      string `json:"project"`
	Entrypoint   string `json:"entrypoint"`
	Member       string `json:"member"`
	In           int64  `json:"in"`
	Out          int64  `json:"out"`
	CacheRead    int64  `json:"cacheRead"`
	CacheWrite   int64  `json:"cacheWrite"`
	Reasoning    int64  `json:"reasoning"`
	LinesAdded   int64  `json:"linesAdded"`
	LinesRemoved int64  `json:"linesRemoved"`
	Edits        int64  `json:"edits"`
	ToolCalls    int64  `json:"toolCalls"`
	Rejected     int64  `json:"rejected"`
	Compactions  int64  `json:"compactions"`
	ReworkLines  int64  `json:"reworkLines"`
}

type metricSessionRow struct {
	SessionID         string    `json:"sessionId"`
	Project           string    `json:"project"`
	Tool              string    `json:"tool"`
	Model             string    `json:"model"`
	Member            string    `json:"member"`
	FirstTs           time.Time `json:"firstTs"`
	LastTs            time.Time `json:"lastTs"`
	Turns             int64     `json:"turns"`
	OutputTokens      int64     `json:"outputTokens"`
	PeakContextTokens int64     `json:"peakContextTokens"`
	Edits             int64     `json:"edits"`
	Compactions       int64     `json:"compactions"`
	ActiveMinutes     float64   `json:"activeMinutes"`
}

type metricDelegation struct {
	Sub   int64 `json:"sub"`
	Total int64 `json:"total"`
}

type metricModelStat struct {
	Model      string   `json:"model"`
	Tier       string   `json:"tier"`
	Tokens     int64    `json:"tokens"`
	Input      int64    `json:"input"`
	Output     int64    `json:"output"`
	CacheRead  int64    `json:"cacheRead"`
	CacheWrite int64    `json:"cacheWrite"`
	Lines      int64    `json:"lines"`
	Cost       *float64 `json:"cost"`
	Priced     bool     `json:"priced"`
	TokenShare float64  `json:"tokenShare"`
}

type metricProjectStat struct {
	Project    string   `json:"project"`
	Lines      int64    `json:"lines"`
	Cost       *float64 `json:"cost"`
	Priced     bool     `json:"priced"`
	TokenShare float64  `json:"tokenShare"`
}

type metricTotals struct {
	Tokens          int64    `json:"tokens"`
	Input           int64    `json:"input"`
	Output          int64    `json:"output"`
	CacheRead       int64    `json:"cacheRead"`
	CacheWrite      int64    `json:"cacheWrite"`
	Lines           int64    `json:"lines"`
	Cost            *float64 `json:"cost"`
	Priced          bool     `json:"priced"`
	CacheEfficiency float64  `json:"cacheEfficiency"`
}

type metricPrice struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

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
		Prices: pricesWire(in.Usage, in.Prices),
	}
}

func (mi *metricInput) marshal() ([]byte, error) { return json.Marshal(mi) }

func usageWire(rows []store.UsageRow) []metricUsageRow {
	out := make([]metricUsageRow, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		out = append(out, metricUsageRow{
			Day: r.Day, Tool: r.Tool, Model: r.Model, Project: r.Project,
			Entrypoint: r.Entrypoint, Member: r.Member,
			In: r.In, Out: r.Out, CacheRead: r.CacheRead, CacheWrite: r.CacheWrite,
			Reasoning: r.Reasoning, LinesAdded: r.LinesAdded, LinesRemoved: r.LinesRemoved,
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
		out[model] = metricPrice{Input: p.Input, Output: p.Output, CacheRead: p.CacheRead, CacheWrite: p.CacheWrite}
	}
	return out
}
