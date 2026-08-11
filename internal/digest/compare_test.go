package digest

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
)

func at(day int) time.Time { return time.Date(2026, 8, day, 12, 0, 0, 0, time.UTC) }

func TestMoversRankByTheSizeOfTheMovement(t *testing.T) {
	got := movers(
		map[string]int64{"small": 100, "big": 1000, "gone": 500, "still": 42},
		map[string]int64{"small": 150, "big": 100, "new": 700, "still": 42},
	)
	want := []string{"big", "new", "gone", "small", "still"}
	if len(got) != len(want) {
		t.Fatalf("got %d movers, want %d: %+v", len(got), len(want), got)
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Errorf("position %d = %q, want %q (full: %+v)", i, got[i].Name, name, got)
		}
	}
}

func TestAppearingAndVanishingAreNotTheSameAsZero(t *testing.T) {
	got := movers(map[string]int64{"gone": 5}, map[string]int64{"new": 5, "flat": 0})
	byName := map[string]Mover{}
	for _, m := range got {
		byName[m.Name] = m
	}
	if !byName["new"].Appeared || byName["new"].Vanished {
		t.Errorf("new = %+v, want Appeared", byName["new"])
	}
	if !byName["gone"].Vanished || byName["gone"].Now != 0 {
		t.Errorf("gone = %+v, want Vanished with Now 0", byName["gone"])
	}
	// A name that was absent and is now zero appeared at zero: it is not movement, and the
	// render drops it -- but the fact that it appeared must not be lost here.
	if !byName["flat"].Appeared {
		t.Errorf("flat = %+v, want Appeared", byName["flat"])
	}
}

func TestVerdictChangesIncludeMetricsThatStoppedReporting(t *testing.T) {
	got := verdictChanges(
		map[string]string{"rework": "good", "cache": "watch", "dropped": "good"},
		map[string]string{"rework": "watch", "cache": "watch", "added": "good"},
		map[string]string{"rework": "high"},
		nil,
	)
	if len(got) != 2 {
		t.Fatalf("got %+v, want the changed one and the dropped one", got)
	}
	if got[0].Name != "dropped" || got[0].To != "—" {
		t.Errorf("got[0] = %+v, want dropped → —", got[0])
	}
	if got[1].Name != "rework" || got[1].From != "good" || got[1].To != "watch" {
		t.Errorf("got[1] = %+v, want rework good → watch", got[1])
	}
}

func TestAChangedReaderIsDeclaredRatherThanReported(t *testing.T) {
	now := Snapshot{Window: "7d", TakenAt: at(11), ParsedBy: "v0.17.0", Priced: true}
	prev := Snapshot{Window: "7d", TakenAt: at(4), ParsedBy: "v0.16.0", Priced: true}
	d := Compare(&now, &prev, nil)
	if !hasCaveat(&d, "reader changed") {
		t.Errorf("caveats = %v, want one naming the changed reader", d.Caveats)
	}
}

func TestTheIntendedWeeklyCadenceHasNoOverlapCaveat(t *testing.T) {
	now := Snapshot{Window: "7d", TakenAt: at(11), ParsedBy: "same", Priced: true}
	prev := Snapshot{Window: "7d", TakenAt: at(4), ParsedBy: "same", Priced: true}
	if d := Compare(&now, &prev, nil); len(d.Caveats) != 0 {
		t.Errorf("a 7d window run 7 days apart produced %v, want no caveats", d.Caveats)
	}
}

func TestRunningTooSoonDeclaresTheOverlap(t *testing.T) {
	now := Snapshot{Window: "7d", TakenAt: at(11), ParsedBy: "same", Priced: true}
	prev := Snapshot{Window: "7d", TakenAt: at(9), ParsedBy: "same", Priced: true}
	d := Compare(&now, &prev, nil)
	if !hasCaveat(&d, "overlap") {
		t.Fatalf("caveats = %v, want one naming the overlap", d.Caveats)
	}
	if !hasCaveat(&d, "5 day") {
		t.Errorf("caveats = %v, want the overlap measured as 5 days", d.Caveats)
	}
}

func TestDifferentWindowsAreNotARateOfChange(t *testing.T) {
	now := Snapshot{Window: "30d", TakenAt: at(11), ParsedBy: "same", Priced: true}
	prev := Snapshot{Window: "7d", TakenAt: at(4), ParsedBy: "same", Priced: true}
	if d := Compare(&now, &prev, nil); !hasCaveat(&d, "windows differ") {
		t.Errorf("caveats = %v, want one naming the differing windows", d.Caveats)
	}
}

func TestAFirstRunComparesAgainstNothing(t *testing.T) {
	d := Compare(&Snapshot{Window: "7d", Tokens: 5}, nil, nil)
	if len(d.Models) != 0 || len(d.Verdicts) != 0 {
		t.Errorf("a first run reported movement: %+v", d)
	}
	md := d.Markdown()
	if !strings.Contains(md, "First digest") {
		t.Errorf("markdown does not say it is the first run:\n%s", md)
	}
	if strings.Contains(md, "What moved") {
		t.Errorf("a first run rendered a movement section:\n%s", md)
	}
}

func TestSnapshotRoundTripsAndRejectsAnotherVersion(t *testing.T) {
	s := Snapshot{Version: SnapshotVersion, Window: "7d", Tokens: 12, Verdicts: map[string]string{"a": "good"}}
	payload, err := s.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	back, ok := Parse(payload)
	if !ok || back.Tokens != 12 || back.Verdicts["a"] != "good" {
		t.Fatalf("Parse = %+v, %v", back, ok)
	}
	if _, ok := Parse([]byte(`{"version":99}`)); ok {
		t.Error("a snapshot from another version was accepted")
	}
	if _, ok := Parse([]byte(`not json`)); ok {
		t.Error("a corrupt payload was accepted")
	}
}

func TestTheDigestLeadsWithWhatAnalyzeLeadsWith(t *testing.T) {
	s := Snapshot{Leads: []string{"cache", "rework"}}
	got := s.LeadsFirst([]VerdictChange{{Name: "adoption"}, {Name: "rework"}, {Name: "cache"}})
	want := []string{"cache", "rework", "adoption"}
	for i, name := range want {
		if got[i].Name != name {
			t.Errorf("position %d = %q, want %q (full: %+v)", i, got[i].Name, name, got)
		}
	}
}

func hasCaveat(d *Digest, substr string) bool {
	for _, c := range d.Caveats {
		if strings.Contains(strings.ToLower(c), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

func TestAnUnpricedWindowSaysTheCostMovementIsPartlyBlindness(t *testing.T) {
	now := Snapshot{
		Window: "7d", TakenAt: at(11), ParsedBy: "same", Priced: false, UnpricedShare: 0.4,
		UnpricedNote: "* cost excludes 40.0% of this window's tokens (2.0B of 5.0B tokens) on models the price table does not carry",
	}
	prev := Snapshot{Window: "7d", TakenAt: at(4), ParsedBy: "same", Priced: true}
	d := Compare(&now, &prev, nil)
	if !hasCaveat(&d, "40.0%") {
		t.Errorf("caveats = %v, want the unpriced share quantified", d.Caveats)
	}
	if !hasCaveat(&d, "could be priced") {
		t.Errorf("caveats = %v, want it to name the cost movement", d.Caveats)
	}
}

func TestAnUnknownPreviousReaderIsNotTreatedAsTheSameReader(t *testing.T) {
	now := Snapshot{Window: "7d", TakenAt: at(11), ParsedBy: "v0.17.0", Priced: true}
	prev := Snapshot{Window: "7d", TakenAt: at(4), ParsedBy: "", Priced: true}
	d := Compare(&now, &prev, nil)
	if !hasCaveat(&d, "could not be checked") {
		t.Errorf("caveats = %v, want the unknown reader declared rather than assumed unchanged", d.Caveats)
	}
}

func TestALongGapSaysTheseAreTwoSeparateWeeks(t *testing.T) {
	now := Snapshot{Window: "7d", TakenAt: at(28), ParsedBy: "same", Priced: true}
	prev := Snapshot{Window: "7d", TakenAt: at(1), ParsedBy: "same", Priced: true}
	d := Compare(&now, &prev, nil)
	if !hasCaveat(&d, "two separate weeks") {
		t.Errorf("caveats = %v, want the gap declared", d.Caveats)
	}
}

// A metric that did not run has no verdict; rendering that as "GOOD -> --" reports movement
// that never happened, twice: once when the plugin breaks and once when it is restored.
func TestAMetricThatDidNotRunIsNotAVanishedVerdict(t *testing.T) {
	now := Snapshot{
		Window: "7d", TakenAt: at(11), ParsedBy: "same", Priced: true,
		Verdicts: map[string]string{"rework": "good"},
	}
	prev := Snapshot{
		Window: "7d", TakenAt: at(4), ParsedBy: "same", Priced: true,
		Verdicts: map[string]string{"rework": "good", "plugin:cost-guard": "good"},
	}

	if d := Compare(&now, &prev, []string{"cost-guard"}); len(d.Verdicts) != 0 {
		t.Errorf("verdict changes = %+v, want none: the plugin did not run", d.Verdicts)
	}
	d := Compare(&now, &prev, []string{"cost-guard"})
	if md := d.Markdown(); !strings.Contains(md, "did not run") {
		t.Errorf("markdown does not name the metric that did not run:\n%s", md)
	}
	// Without that knowledge it must still be reported, so a genuinely dropped metric is seen.
	if d := Compare(&now, &prev, nil); len(d.Verdicts) != 1 {
		t.Errorf("verdict changes = %+v, want the dropped metric reported", d.Verdicts)
	}
}

func TestAVerdictChangeCarriesWhatItRestsOn(t *testing.T) {
	now := Snapshot{
		Window: "7d", TakenAt: at(11), ParsedBy: "same", Priced: true,
		Verdicts:   map[string]string{"thin": "watch", "solid": "watch"},
		Confidence: map[string]string{"thin": "insufficient", "solid": "high"},
	}
	prev := Snapshot{
		Window: "7d", TakenAt: at(4), ParsedBy: "same", Priced: true,
		Verdicts: map[string]string{"thin": "good", "solid": "good"},
	}
	d := Compare(&now, &prev, nil)
	md := d.Markdown()
	if !strings.Contains(md, "likely noise in a thin window") {
		t.Errorf("a change on insufficient confidence reads as a real one:\n%s", md)
	}
	if !strings.Contains(md, "*(high confidence)*") {
		t.Errorf("a change on high confidence is not marked as such:\n%s", md)
	}
}

// Take must store leads in the ranking MarkLead produced, not in the order results arrive:
// analyze.Validators() is name-sorted, so appending as they come would invent a second order.
func TestLeadsAreStoredInRankOrderNotResultOrder(t *testing.T) {
	results := []analyze.Result{
		{Name: "alphabetically-first", Lead: &analyze.Lead{Rank: 2}},
		{Name: "zzz-ranked-first", Lead: &analyze.Lead{Rank: 1}},
		{Name: "not-a-lead"},
	}
	s := Take(&analyze.Input{}, results, Options{Window: "7d", At: at(11)})
	want := []string{"zzz-ranked-first", "alphabetically-first"}
	if len(s.Leads) != len(want) {
		t.Fatalf("leads = %v, want %v", s.Leads, want)
	}
	for i, name := range want {
		if s.Leads[i] != name {
			t.Errorf("leads[%d] = %q, want %q (full: %v)", i, s.Leads[i], name, s.Leads)
		}
	}
}

// The pseudonym applies at render time, never to the stored key: a stored pseudonym would
// change the moment privacy.anonymize is toggled, and every project would then read as one
// that appeared beside one that vanished.
func TestTheStoredKeyIsTheRealNameAndTheRenderIsPseudonymized(t *testing.T) {
	in := analyze.Input{ByProject: []analyze.ProjectStat{{Project: "acme-web", Lines: 9}}}
	now := Take(&in, nil, Options{Window: "7d", At: at(11)})
	now.Priced, now.ParsedBy = true, "same"
	if now.Projects["acme-web"] != 9 {
		t.Fatalf("stored projects = %v, want the real name as the comparison key", now.Projects)
	}
	prev := Snapshot{
		Window: "7d", TakenAt: at(4), ParsedBy: "same", Priced: true,
		Projects: map[string]int64{"acme-web": 4},
	}

	d := Compare(&now, &prev, nil)
	md := d.WithPseudonym(func(string) string { return "project-abc123" }).Markdown()
	if strings.Contains(md, "acme-web") {
		t.Errorf("the real project name reached the rendered digest:\n%s", md)
	}
	if !strings.Contains(md, "project-abc123") {
		t.Errorf("the pseudonym is missing from the render:\n%s", md)
	}
}

// A row with no price and no tokens leaves the total complete, and its own disclosure says
// so. Treating that as an unpriced window produced a caveat contradicting its own quote.
func TestAPricelessRowCarryingNoTokensIsNotAnUnpricedWindow(t *testing.T) {
	now := Snapshot{
		Window: "7d", TakenAt: at(11), ParsedBy: "same", Priced: false, UnpricedShare: 0,
		UnpricedNote: "* marks rows on a model with no known price; they carry no tokens, so the total above is complete",
	}
	prev := Snapshot{Window: "7d", TakenAt: at(4), ParsedBy: "same", Priced: true}
	d := Compare(&now, &prev, nil)
	if hasCaveat(&d, "could be priced") {
		t.Errorf("caveats = %v, want no unpriced caveat: no tokens are missing a price", d.Caveats)
	}
}
