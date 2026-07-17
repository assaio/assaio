package report

import (
	"fmt"
	"strings"

	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// EffRow is one usage group's effectiveness signals under a single --by dimension:
// AI-line output, edit/rejection activity, and cost efficiency.
type EffRow struct {
	// Group is this row's label for the chosen --by dimension, e.g. a project name.
	Group string `json:"group"`
	// LinesAdded is AI-added code lines, the primary effect proxy.
	LinesAdded int64 `json:"lines_added"`
	// LinesRemoved is AI-removed code lines.
	LinesRemoved int64 `json:"lines_removed"`
	// Edits is the count of productive edit tool-calls (Edit/Write/NotebookEdit/MultiEdit).
	Edits int64 `json:"edits"`
	// ToolCalls is the count of all tool-use calls in the group, including edits.
	ToolCalls int64 `json:"tool_calls"`
	// Rejected is tool-use proposals the human declined: a friction signal.
	Rejected int64 `json:"rejected"`
	// TokensTotal sums input, output, cache-read, cache-write, and reasoning tokens.
	TokensTotal int64 `json:"tokens_total"`
	// Cost is USD cost summed from priced usage only; nil when nothing in the group priced.
	Cost *float64 `json:"cost"`
	// HasUnpriced reports whether this group excludes some usage's cost because its
	// model has no known price.
	HasUnpriced bool `json:"has_unpriced"`
	// CostPer100Lines is Cost per 100 AI lines; nil when LinesAdded is zero or Cost is
	// unknown -- never a divide-by-zero substitute for a legitimate zero-line group.
	CostPer100Lines *float64 `json:"cost_per_100_lines"`
}

// BuildEffectiveness groups rows by the chosen --by dimension (day, project, tool,
// model, or entrypoint) and computes each group's AI-line output against its cost. An
// unknown dimension returns an error listing the valid ones.
func BuildEffectiveness(rows []store.UsageRow, t pricing.Table, by string) ([]EffRow, error) {
	if !isValidDim(by) {
		return nil, fmt.Errorf("unknown dimension %q (want one of %s)", by, strings.Join(validDims, ", "))
	}

	out := groupBy(
		len(rows),
		func(i int) string { return usageDimValue(&rows[i], by) },
		func(key string) EffRow { return EffRow{Group: key} },
		func(g *EffRow, i int) { accumulateEff(g, &rows[i], t) },
	)
	for i := range out {
		out[i].CostPer100Lines = costPer100Lines(out[i].Cost, out[i].LinesAdded)
	}
	return out, nil
}

// usageDimValue returns u's value for one of validDims.
func usageDimValue(u *store.UsageRow, by string) string {
	switch by {
	case "day":
		return u.Day
	case "project":
		return u.Project
	case "tool":
		return u.Tool
	case "model":
		return u.Model
	case "entrypoint":
		return u.Entrypoint
	case "member":
		return u.Member
	default:
		return ""
	}
}

// accumulateEff folds u's activity counts and, when u.Model is priced, its cost into g.
func accumulateEff(g *EffRow, u *store.UsageRow, t pricing.Table) {
	g.LinesAdded += u.LinesAdded
	g.LinesRemoved += u.LinesRemoved
	g.Edits += u.Edits
	g.ToolCalls += u.ToolCalls
	g.Rejected += u.Rejected
	g.TokensTotal += u.In + u.Out + u.CacheRead + u.CacheWrite + u.Reasoning

	cost, ok := t.CostTokens(u.Model, u.In, u.Out, u.CacheWrite, u.CacheRead)
	if !ok {
		g.HasUnpriced = true
		return
	}
	if g.Cost == nil {
		zero := 0.0
		g.Cost = &zero
	}
	*g.Cost += cost
}

// costPer100Lines is cost per 100 AI lines; nil when linesAdded is zero or cost is
// unknown, so a group is shown as having no defined ratio rather than a fake one.
func costPer100Lines(cost *float64, linesAdded int64) *float64 {
	if linesAdded == 0 || cost == nil {
		return nil
	}
	ratio := *cost / (float64(linesAdded) / 100)
	return &ratio
}
