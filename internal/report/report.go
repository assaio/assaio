// Package report aggregates stored usage rows into priced report rows and renders
// them as a table, JSON, or CSV.
package report

import (
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// Row is one day/tool/model/project/entrypoint usage group with its computed cost.
type Row struct {
	// Day, Tool, Model identify the usage group.
	Day   string `json:"day"`
	Tool  string `json:"tool"`
	Model string `json:"model"`
	// Project, Entrypoint further narrow the usage group.
	Project    string `json:"project"`
	Entrypoint string `json:"entrypoint"`
	// Member is "" for purely local usage; non-empty only on a central store synced
	// from a team member.
	Member string `json:"member"`
	// In, Out, CacheRead, CacheWrite, Reasoning are summed token counts.
	In         int64 `json:"in"`
	Out        int64 `json:"out"`
	CacheRead  int64 `json:"cache_read"`
	CacheWrite int64 `json:"cache_write"`
	Reasoning  int64 `json:"reasoning"`
	// Cost is the USD cost computed from the pricing table; nil when the model is unpriced.
	Cost *float64 `json:"cost"`
	// Priced reports whether Cost was computed from a known model price.
	Priced bool `json:"priced"`
	// CacheEff is CacheRead / (In + CacheRead); nil when both are zero.
	CacheEff *float64 `json:"cache_eff"`
	// HasUnpriced reports whether this row aggregates any usage with an unknown model
	// price, meaning Cost excludes that usage.
	HasUnpriced bool `json:"has_unpriced"`
}

// cacheEff returns CacheRead / (in + CacheRead), or nil when both are zero.
func cacheEff(in, cacheRead int64) *float64 {
	if in+cacheRead == 0 {
		return nil
	}
	eff := float64(cacheRead) / float64(in+cacheRead)
	return &eff
}

// Build prices each usage row against t, matching the model exactly first and falling
// back to pricing.NormalizeModel. Unmatched models are returned with Priced=false and
// Cost=nil.
func Build(rows []store.UsageRow, t pricing.Table) []Row {
	out := make([]Row, 0, len(rows))
	for i := range rows {
		u := &rows[i]
		r := Row{
			Day: u.Day, Tool: u.Tool, Model: u.Model, Project: u.Project, Entrypoint: u.Entrypoint, Member: u.Member,
			In: u.In, Out: u.Out, CacheRead: u.CacheRead, CacheWrite: u.CacheWrite, Reasoning: u.Reasoning,
		}
		if cost, ok := t.CostTokens(u.Model, u.In, u.Out, u.CacheWrite, u.CacheRead); ok {
			r.Cost = &cost
			r.Priced = true
		} else {
			r.HasUnpriced = true
		}
		r.CacheEff = cacheEff(r.In, r.CacheRead)
		out = append(out, r)
	}
	return out
}

// validDims are the dimensions Aggregate accepts for --by, in canonical order.
var validDims = []string{"day", "project", "tool", "model", "entrypoint", "member"}

// dimValue returns r's value for one of validDims.
func dimValue(r *Row, by string) string {
	switch by {
	case "day":
		return r.Day
	case "project":
		return r.Project
	case "tool":
		return r.Tool
	case "model":
		return r.Model
	case "entrypoint":
		return r.Entrypoint
	case "member":
		return r.Member
	default:
		return ""
	}
}

// emptyDimLabel is the table placeholder for a grouped row whose --by dimension value is
// "": "(local)" for member, since an empty member is the expected default for
// never-synced local usage, not a missing value; "(unknown)" for every other dimension.
func emptyDimLabel(by string) string {
	if by == "member" {
		return "(local)"
	}
	return "(unknown)"
}
