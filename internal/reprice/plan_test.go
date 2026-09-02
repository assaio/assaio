package reprice

import (
	"testing"

	"github.com/assaio/assaio/internal/store"
)

func TestParsePlan(t *testing.T) {
	for _, tc := range []struct {
		name    string
		in      string
		want    Plan
		wantErr bool
	}{
		{name: "name and price", in: "Max 20x=200", want: Plan{Name: "Max 20x", Monthly: 200, Source: SourceFlag}},
		{name: "spaces and a dollar sign", in: " Pro = $20 ", want: Plan{Name: "Pro", Monthly: 20, Source: SourceFlag}},
		{name: "a name may contain the separator", in: "a=b=30", want: Plan{Name: "a=b", Monthly: 30, Source: SourceFlag}},
		{name: "no separator", in: "Max 20x", wantErr: true},
		{name: "no name", in: "=200", wantErr: true},
		{name: "not a price", in: "Max=free", wantErr: true},
		{name: "free is not a plan", in: "Max=0", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParsePlan(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParsePlan(%q) = %+v, want an error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePlan(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ParsePlan(%q) = %+v, want %+v", tc.in, got, tc.want)
			}
		})
	}
}

// TestPlansPutTheConfiguredOneFirst: the plan a user is actually on is the control the
// candidates are read against, so it leads the list rather than sorting into it.
func TestPlansPutTheConfiguredOneFirst(t *testing.T) {
	in := input([]store.UsageRow{row("big", 1_000_000, 100_000, 10_000_000, 1_000_000)}, 5)
	supplied, err := ParsePlan("cheaper tier=25")
	if err != nil {
		t.Fatal(err)
	}
	w := Compute(&in, Options{Plans: []Plan{supplied}})

	if len(w.Plans) != 2 {
		t.Fatalf("plans = %+v, want the configured one and the candidate", w.Plans)
	}
	if w.Plans[0].Source != SourceConfig || w.Plans[1].Source != SourceFlag {
		t.Fatalf("plans = %+v, want the configured plan first", w.Plans)
	}
	// The window spans exactly the 30 days a monthly projection divides by, so the multiple is
	// the observed 18.75 over each plan price with no projection arithmetic in between.
	if w.Plans[0].Multiple == nil || w.Plans[1].Multiple == nil {
		t.Fatalf("plans = %+v, want a multiple on a priced window", w.Plans)
	}
	if !near(*w.Plans[0].Multiple, 18.75/5) || !near(*w.Plans[1].Multiple, 18.75/25) {
		t.Errorf("multiples = %v and %v, want 3.75 and 0.75", *w.Plans[0].Multiple, *w.Plans[1].Multiple)
	}
	// Below one, the plan costs more than the window's API-equivalent -- the direction the
	// figure alone does not carry.
	if got := multipleNote(w.Plans[1].Multiple); got != "the plan costs more than this window's API-equivalent at this volume" {
		t.Errorf("note for a multiple below one = %q", got)
	}
}

// TestAnEmptyWindowJudgesNoPlan: absence is never zero. A window with nothing priced in it has
// no side to compare a plan against, and a multiple of zero there would read as a plan
// returning nothing -- a verdict on the plan drawn from the absence of evidence about it.
func TestAnEmptyWindowJudgesNoPlan(t *testing.T) {
	in := input(nil, 195)
	w := Compute(&in, Options{})

	if len(w.Plans) != 1 || w.Plans[0].Multiple != nil {
		t.Fatalf("plans = %+v, want the plan listed with no multiple", w.Plans)
	}
	if got := multipleNote(w.Plans[0].Multiple); got != "no priced usage this window to compare against the plan" {
		t.Errorf("note = %q, want the abstention", got)
	}
	if w.Basis.Days != 0 || windowSpan(&w.Basis) != "—" {
		t.Errorf("days = %v, want no span claimed behind figures that have none", w.Basis.Days)
	}
	if w.Routes == nil {
		t.Error("routes must encode as [], never null")
	}
}
