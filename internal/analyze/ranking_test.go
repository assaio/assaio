package analyze

import (
	"bytes"
	"strings"
	"testing"
)

// finding builds a Result in the shape Evaluate leaves one, so a ranking test exercises the
// ordering rather than any validator's logic.
func finding(name, readKey, confidence string, samples int, signal float64) Result {
	r := Result{Name: name, Title: strings.ToUpper(name[:1]) + name[1:], Read: Read{Key: readKey}}
	r.Confidence = Confidence{
		Label: confidence, Samples: samples, Unit: "sessions",
		Activity: 1, Priced: 1, Turn: 1, Signal: &signal,
	}
	return r
}

// The whole point of an ordering is that a reader acts on something. A window with nothing
// worth acting on has to say so: promoting the least weak finding teaches the reader that the
// top of the list means nothing, which costs more than the empty list ever would.
func TestRankPromotesNothingWhenEveryFindingIsWeak(t *testing.T) {
	tests := []struct {
		name    string
		results []Result
	}{
		{"every read is favorable", []Result{
			finding("adoption", "good", ConfidenceHigh, 40, 1),
			finding("throughput", "good", ConfidenceHigh, 40, 1),
		}},
		{"every read is withheld", []Result{
			finding("adoption", "neutral", ConfidenceHigh, 40, 1),
		}},
		{"the flagged ones rest on too little", []Result{
			finding("adoption", "watch", ConfidenceLow, 2, 1),
			finding("friction", "watch", ConfidenceInsufficient, 0, 0),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Rank(tt.results); len(got) != 0 {
				t.Fatalf("Rank promoted %d findings, want none: %+v", len(got), got)
			}
		})
	}
}

// Stronger evidence leads, then wider reach. Both are already on every Confidence, which is
// the whole claim: this orders by what is knowable, never by a guess at impact.
func TestRankOrdersByEvidenceThenReach(t *testing.T) {
	results := []Result{
		finding("narrow", "watch", ConfidenceHigh, 40, 0.2),
		finding("weaker", "watch", ConfidenceMedium, 40, 1),
		finding("broad", "watch", ConfidenceHigh, 40, 0.9),
	}
	got := Rank(results)
	want := []string{"broad", "narrow", "weaker"}
	if len(got) != len(want) {
		t.Fatalf("Rank returned %d findings, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Name != want[i] {
			t.Fatalf("order = %v, want %v", names(got), want)
		}
	}
}

// A window with more findings than a week has room for leads with a few, and the rest stay
// exactly where they were -- one screen down, not dropped.
func TestRankCapsWhatLeadsWithoutHidingTheRest(t *testing.T) {
	var results []Result
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		results = append(results, finding(name, "watch", ConfidenceHigh, 40, 1))
	}
	got := Rank(results)
	if len(got) != RankedMax {
		t.Fatalf("Rank returned %d findings, want the %d that lead", len(got), RankedMax)
	}
	var buf bytes.Buffer
	if err := RenderRankingText(&buf, got, len(results)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "of 5 reads") {
		t.Fatalf("the lead must say what it was chosen from: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "Every read is below") {
		t.Fatalf("the lead must say the rest are still there: %q", buf.String())
	}
}

// Every promoted finding shows the reasons that put it there, and they have to be the ones
// the ordering actually weighed -- an order a reader cannot audit is a score in disguise.
func TestRankShowsTheReasonsItOrderedBy(t *testing.T) {
	got := Rank([]Result{finding("friction", "watch", ConfidenceHigh, 328772, 0.42)})
	if len(got) != 1 {
		t.Fatalf("Rank returned %d findings, want 1", len(got))
	}
	joined := strings.Join(got[0].Reasons, " · ")
	for _, want := range []string{"flagged for a closer look", "high confidence", "328,772 sessions", "covers 42% of the window"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("reasons = %q, want them to say %q", joined, want)
		}
	}
}

// A window that promotes nothing says so in words, rather than printing an empty heading a
// reader would take for a rendering bug.
func TestRenderRankingTextNamesTheEmptyCase(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderRankingText(&buf, nil, 19); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "nothing here") || !strings.Contains(buf.String(), "All 19 reads") {
		t.Fatalf("empty ranking must explain itself: %q", buf.String())
	}
}

// MarkLead is the machine-readable half, and it must mark exactly what leads -- a window that
// promotes nothing leaves every Result unmarked, so a script reading the field learns the
// same "act on nothing this week" the terminal shows.
func TestMarkLeadStampsOnlyWhatLeads(t *testing.T) {
	results := []Result{
		finding("acted", "watch", ConfidenceHigh, 40, 1),
		finding("fine", "good", ConfidenceHigh, 40, 1),
	}
	MarkLead(results)
	if results[0].Lead == nil || results[0].Lead.Rank != 1 {
		t.Fatalf("Lead = %+v, want the flagged read stamped first", results[0].Lead)
	}
	if results[1].Lead != nil {
		t.Fatalf("Lead = %+v on a favorable read, want none", results[1].Lead)
	}

	weak := []Result{finding("thin", "watch", ConfidenceLow, 2, 1)}
	MarkLead(weak)
	if weak[0].Lead != nil {
		t.Fatalf("Lead = %+v on a window that promotes nothing, want none", weak[0].Lead)
	}
}

func names(ranked []Ranked) []string {
	out := make([]string, len(ranked))
	for i := range ranked {
		out[i] = ranked[i].Name
	}
	return out
}
