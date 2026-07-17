package report

import "testing"

func TestMoversDiffsAndSortsByCostDelta(t *testing.T) {
	c := func(f float64) *float64 { return &f }
	recent := []EffRow{
		{Group: "web", Cost: c(10), LinesAdded: 100},
		{Group: "api", Cost: c(5), LinesAdded: 50},
		{Group: "new", Cost: c(8), LinesAdded: 30}, // only in recent
	}
	prior := []EffRow{
		{Group: "web", Cost: c(3), LinesAdded: 40},
		{Group: "api", Cost: c(6), LinesAdded: 55},
		{Group: "gone", Cost: c(2), LinesAdded: 10}, // only in prior
	}
	got := Movers(recent, prior)
	if len(got) != 4 {
		t.Fatalf("want 4 mover rows (union of both windows), got %d", len(got))
	}
	// |Δcost|: new(8), web(7), gone(2), api(1) -> sorted largest first.
	if got[0].Group != "new" || got[0].DeltaCost != 8 {
		t.Fatalf("first mover = %+v, want new +8", got[0])
	}
	if got[1].Group != "web" || got[1].DeltaCost != 7 || got[1].DeltaLines != 60 {
		t.Fatalf("second mover = %+v, want web +7 cost, +60 lines", got[1])
	}
	var gone *MoverRow
	for i := range got {
		if got[i].Group == "gone" {
			gone = &got[i]
		}
	}
	if gone == nil || gone.DeltaCost != -2 || gone.CostNow != 0 {
		t.Fatalf("dropped group 'gone' = %+v, want Δ-2, now 0", gone)
	}
}
