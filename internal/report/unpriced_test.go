package report

import (
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/store"
)

// usageRows is a window in the shape a real store holds one: priced work, work on a model the
// table has no row for, and the zero-token "<synthetic>" rows Claude writes for its own
// locally-generated turns.
func usageRows(t *testing.T) []store.UsageRow {
	t.Helper()
	return []store.UsageRow{
		{Day: "2026-07-01", Tool: "claude-code", Model: "claude-opus-4-5", Project: "web", In: 400, Out: 100, CacheRead: 500},
		{Day: "2026-07-01", Tool: "claude-code", Model: "unknown-model", Project: "web", In: 300, Out: 200},
		{Day: "2026-07-02", Tool: "claude-code", Model: "<synthetic>", Project: "api"},
		{Day: "2026-07-02", Tool: "claude-code", Model: "claude-opus-4-5", Project: "api", In: 500, Out: 500},
	}
}

// A window's unpriced share is the error bar on its cost, and it has to survive aggregation:
// the same rows grouped by project must report the same share as the day/tool/model rows they
// came from, or a reader learns a different number depending on which --by they typed.
func TestUnpricedShareSurvivesAggregation(t *testing.T) {
	rows := Build(usageRows(t), table())
	base := BuildUnpriced(rows)
	if base.Total == 0 {
		t.Fatal("fixture carries no tokens")
	}
	grouped, err := Aggregate(rows, "project")
	if err != nil {
		t.Fatal(err)
	}
	got := BuildUnpriced(grouped)
	if got.Tokens != base.Tokens || got.Total != base.Total {
		t.Fatalf("grouped = %d/%d, want the base rows' %d/%d", got.Tokens, got.Total, base.Tokens, base.Total)
	}
}

// The disclosure has to separate a cost that is short from a cost that is whole. An unpriced
// row carrying no token -- Claude writes locally-generated turns as "<synthetic>" with an
// all-zero usage block, 1,108 of them on the maintainer's own store -- leaves the total
// complete, and telling that reader their cost excludes 0.0% of anything is noise dressed as
// a warning.
func TestUnpricedDisclosureSeparatesShortFromComplete(t *testing.T) {
	tests := []struct {
		name string
		u    Unpriced
		want string
	}{
		{"nothing unpriced", Unpriced{Total: 1000}, ""},
		{"rows but no tokens", Unpriced{Total: 1000, Rows: 3}, "they carry no tokens, so the total above is complete"},
		{"a share worth acting on", Unpriced{Tokens: 455, Total: 1000, Rows: 2}, "cost excludes 45.5% of the tokens in view"},
		{"a share too small to act on", Unpriced{Tokens: 1, Total: 100_000, Rows: 1}, "cost excludes <0.1% of the tokens in view"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UnpricedDisclosure(&tt.u, "the tokens in view")
			if tt.want == "" {
				if got != "" {
					t.Fatalf("disclosure = %q, want none", got)
				}
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Fatalf("disclosure = %q, want it to say %q", got, tt.want)
			}
		})
	}
}

// UnpricedModels answers what a refresh has to cover, heaviest first, and never names a model
// whose rows carry no tokens: that model costs nothing to leave unpriced.
func TestUnpricedModelsRankByWhatIsMissing(t *testing.T) {
	got := UnpricedModels(usageRows(t), table())
	for _, model := range got {
		if model == "<synthetic>" {
			t.Fatalf("models = %v, want no zero-token model named", got)
		}
	}
	if len(got) > 1 {
		t.Fatalf("models = %v, want only the one unpriced model carrying tokens", got)
	}
	if len(got) == 1 && got[0] != "unknown-model" {
		t.Fatalf("models = %v, want the unpriced model that carries tokens", got)
	}
}
