package reprice

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// The tier split is a price threshold, not a name: "big" is premium because its output rate
// is at or above the floor internal/analyze classifies on, and "small" is not.
var table = pricing.Table{
	"big":   {Input: 5e-6, Output: 25e-6, CacheWrite: 6.25e-6, CacheRead: 5e-7},
	"small": {Input: 1e-6, Output: 5e-6, CacheWrite: 1.25e-6, CacheRead: 1e-7},
}

var (
	windowStart = time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	windowEnd   = time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
)

func row(model string, in, out, cacheRead, cacheWrite int64) store.UsageRow {
	return store.UsageRow{
		Day: "2026-07-01", Tool: "claude-code", Model: model,
		In: in, Out: out, CacheRead: cacheRead, CacheWrite: cacheWrite,
	}
}

// input builds the window every test reads, spanning exactly the 30 days a monthly projection
// divides by, so a projected figure equals the window figure and no assertion has to restate
// the projection arithmetic to check the re-pricing.
func input(rows []store.UsageRow, plan float64) analyze.Input {
	in := analyze.BuildInput(rows, nil, table, windowEnd, 0, analyze.Delegation{})
	in.WindowStart = windowStart
	in.PlanMonthlyCost = plan
	return in
}

func near(got, want float64) bool { return math.Abs(got-want) < 1e-9 }

// TestRouteMovesOnlyThePremiumSlice is the arithmetic the whole command rests on: the window
// re-priced is everything outside the premium slice exactly as observed, plus that slice at
// the target's rates. A route that re-priced the whole window would be answering a different
// question -- "what if all of this ran on one model" -- while reading as a mix.
func TestRouteMovesOnlyThePremiumSlice(t *testing.T) {
	in := input([]store.UsageRow{
		row("big", 1_000_000, 100_000, 10_000_000, 1_000_000),
		row("small", 100_000, 10_000, 0, 0),
	}, 0)
	w := Compute(&in, Options{})

	for _, want := range []struct {
		label string
		got   float64
		want  float64
	}{
		{"observed cost", w.Basis.Cost, 18.90},
		{"premium slice", w.Basis.Premium.Cost, 18.75},
	} {
		if !near(want.got, want.want) {
			t.Errorf("%s = %v, want %v", want.label, want.got, want.want)
		}
	}
	if len(w.Routes) != 1 || w.Routes[0].Target != "small" {
		t.Fatalf("routes = %+v, want the one cheaper model this window ran", w.Routes)
	}
	r := w.Routes[0]
	if !near(r.Premium, 3.75) {
		t.Errorf("premium slice at small's rates = %v, want 3.75", r.Premium)
	}
	// The cheaper model's own 0.15 stays exactly as observed: 18.90 - 18.75 + 3.75.
	if !near(r.Window, 3.90) || !near(r.Delta, 15.00) {
		t.Errorf("route = %+v, want window 3.90 and delta 15.00", r)
	}
}

// TestUnpricedTokensStayInTheMargin: a re-priced total that silently excluded unpriced usage
// would be the failure this project already fixed once. The tokens are outside both sides and
// the window says so, in the share and in the assumption.
func TestUnpricedTokensStayInTheMargin(t *testing.T) {
	in := input([]store.UsageRow{
		row("big", 1_000_000, 100_000, 10_000_000, 1_000_000),
		row("nobody-priced-this", 40_000_000, 1_000_000, 0, 0),
	}, 0)
	w := Compute(&in, Options{})

	if w.Basis.Unpriced.Tokens != 41_000_000 || w.Basis.Unpriced.Total != 53_100_000 {
		t.Fatalf("unpriced = %+v, want 41M of 53.1M tokens", w.Basis.Unpriced)
	}
	if w.Basis.Trustworthy() {
		t.Errorf("share %v is above the ceiling and must not be acted on", w.Basis.Unpriced.Share)
	}
	if !strings.Contains(strings.Join(w.Assumptions, " "), "excluded from both sides") {
		t.Errorf("assumptions = %q, want the exclusion stated", w.Assumptions)
	}
	if got := strings.Join(w.Basis.Unpriced.Models, ","); got != "nobody-priced-this" {
		t.Errorf("unpriced models = %q, want the model a price refresh has to cover", got)
	}
}

// TestNoPremiumSliceProposesNoRoute: the absence has four different causes and each sends a
// reader somewhere else, so a single "no routes" line would be the least useful of the four.
func TestNoPremiumSliceProposesNoRoute(t *testing.T) {
	for _, tc := range []struct {
		name    string
		rows    []store.UsageRow
		against []string
		want    string
	}{
		{name: "no priced usage", rows: []store.UsageRow{row("unknown", 1000, 100, 0, 0)}, want: "carries a known price"},
		{name: "nothing premium", rows: []store.UsageRow{row("small", 1000, 100, 0, 0)}, want: "premium tier"},
		{name: "nowhere to move it", rows: []store.UsageRow{row("big", 1000, 100, 0, 0)}, want: "--against"},
		{
			name: "the only target named has no price", rows: []store.UsageRow{row("big", 1000, 100, 0, 0)},
			against: []string{"no-such-model"}, want: "no rate for the target(s) named below",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := input(tc.rows, 0)
			w := Compute(&in, Options{Against: tc.against})
			if len(w.Routes) != 0 {
				t.Fatalf("routes = %+v, want none", w.Routes)
			}
			if got := routeAbsence(&w); !strings.Contains(got, tc.want) {
				t.Errorf("absence = %q, want it to name %q", got, tc.want)
			}
		})
	}
}

// TestAgainstNamesAModelTheWindowNeverRan is the competitor case: the arbitrage is only worth
// running if it can reach a price the user has not already paid. The target the table cannot
// cost is named rather than dropped: one priced target beside it would otherwise render as the
// complete answer to a question that asked about two.
func TestAgainstNamesAModelTheWindowNeverRan(t *testing.T) {
	in := input([]store.UsageRow{row("big", 1_000_000, 100_000, 10_000_000, 1_000_000)}, 0)
	w := Compute(&in, Options{Against: []string{"small", "no-such-model"}})

	if len(w.Routes) != 1 || w.Routes[0].Target != "small" {
		t.Fatalf("routes = %+v, want only the named model that has a price", w.Routes)
	}
	if got := strings.Join(w.Unpriceable, ","); got != "no-such-model" {
		t.Errorf("unpriceable = %q, want the named target the price table cannot cost", got)
	}
}

// TestFlatPlanIsAnAssumptionNotAPrerequisite: once a flat subscription is configured, this
// binary knows the reader's bill does not move with the delta it prints. The sentence that says
// so used to fire only inside a recommendation gated on three other conditions.
func TestFlatPlanIsAnAssumptionNotAPrerequisite(t *testing.T) {
	rows := []store.UsageRow{
		row("big", 1_000_000, 100_000, 10_000_000, 1_000_000),
		row("small", 100_000, 10_000, 0, 0),
	}
	for _, tc := range []struct {
		name   string
		plan   float64
		stated bool
	}{
		{name: "no plan configured", plan: 0, stated: false},
		{name: "a flat plan configured", plan: 200, stated: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := input(rows, tc.plan)
			w := Compute(&in, Options{})
			joined := strings.Join(w.Assumptions, " ")
			if got := strings.Contains(joined, "flat subscription of $200/mo is configured"); got != tc.stated {
				t.Errorf("flat-plan assumption stated = %v, want %v, in %q", got, tc.stated, joined)
			}
			if !strings.Contains(joined, "Cost is an estimate at public pay-as-you-go API prices") {
				t.Errorf("assumptions = %q, want the same cost-estimate disclosure every other cost surface renders", joined)
			}
		})
	}
}
