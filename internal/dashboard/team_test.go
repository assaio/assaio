package dashboard

import (
	"bytes"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/store"
)

// fixtureInputWithMembers extends fixtureInput with two members' usage and sessions --
// alice more active (2 sessions) than bob (1 session) -- for exercising the Team section.
func fixtureInputWithMembers() analyze.Input {
	in := fixtureInput()
	in.Usage = append(
		in.Usage,
		store.UsageRow{
			Day: "2026-07-12", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", Member: "alice",
			In: 1000, Out: 500, LinesAdded: 40,
		},
		store.UsageRow{
			Day: "2026-07-12", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", Member: "bob",
			In: 200, Out: 100, LinesAdded: 5,
		},
	)
	in.Sessions = append(
		in.Sessions,
		store.SessionRow{
			SessionID: "s3", Project: "web", Tool: "claude-code", Model: "claude-sonnet-4-5", Member: "alice",
			FirstTs: fixtureNow, LastTs: fixtureNow,
		},
		store.SessionRow{
			SessionID: "s4", Project: "web", Tool: "claude-code", Model: "claude-sonnet-4-5", Member: "alice",
			FirstTs: fixtureNow, LastTs: fixtureNow,
		},
		store.SessionRow{
			SessionID: "s5", Project: "web", Tool: "claude-code", Model: "claude-sonnet-4-5", Member: "bob",
			FirstTs: fixtureNow, LastTs: fixtureNow,
		},
	)
	return analyze.BuildInput(in.Usage, in.Sessions, in.Prices, in.Now, in.Recent, in.Delegation)
}

func TestBuildTeamAbsentWithoutMemberData(t *testing.T) {
	d := Build(fixtureInput(), "last 30 days", true, nil, nil)
	if d.Team != nil {
		t.Fatalf("Team = %+v, want nil for a purely local store with no member data", d.Team)
	}
}

// twoMemberInput is the alice-vs-bob comparison on its own: bob out-spends and
// out-produces alice by two orders of magnitude, alice engages twice as often.
// Deliberately self-contained (not fixtureInputWithMembers, which also carries the base
// fixture's un-tagged local rows) so that contrast is the only variable.
func twoMemberInput() analyze.Input {
	usage := []store.UsageRow{
		{
			Day: "2026-07-12", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", Member: "alice",
			In: 1000, Out: 500, LinesAdded: 40,
		},
		{
			Day: "2026-07-12", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", Member: "bob",
			In: 100000, Out: 50000, LinesAdded: 5000,
		},
	}
	sessions := []store.SessionRow{
		{SessionID: "s1", Project: "web", Tool: "claude-code", Model: "claude-sonnet-4-5", Member: "alice", FirstTs: fixtureNow, LastTs: fixtureNow},
		{SessionID: "s2", Project: "web", Tool: "claude-code", Model: "claude-sonnet-4-5", Member: "alice", FirstTs: fixtureNow, LastTs: fixtureNow},
		{SessionID: "s3", Project: "web", Tool: "claude-code", Model: "claude-sonnet-4-5", Member: "bob", FirstTs: fixtureNow, LastTs: fixtureNow},
	}
	return analyze.BuildInput(usage, sessions, fixturePrices(), fixtureNow, 7*24*time.Hour, analyze.Delegation{})
}

// TestBuildTeamBarScalesBySessionsNotCostOrLines locks in what the bar means: engagement
// frequency, never a cost or lines-added scoreboard. A bar drawn from output would run bob's
// to full width and leave alice's a sliver; drawn from sessions it is alice's that fills and
// bob's that reaches half.
func TestBuildTeamBarScalesBySessionsNotCostOrLines(t *testing.T) {
	d := Build(twoMemberInput(), "last 30 days", false, nil, nil)
	if d.Team == nil || len(d.Team.Stats) != 2 {
		t.Fatalf("Team = %+v, want 2 member stats", d.Team)
	}
	byLabel := map[string]TeamStat{}
	for _, s := range d.Team.Stats {
		byLabel[s.Member] = s
	}
	busiest, quietest := byLabel[memberLabel("alice")], byLabel[memberLabel("bob")]
	if busiest.Sessions != 2 || busiest.Frac != 1 {
		t.Fatalf("2-session member = %+v, want Frac 1 -- scaled against the list's own max", busiest)
	}
	if quietest.Sessions != 1 || quietest.Frac != 0.5 {
		t.Fatalf("1-session member = %+v, want Frac 0.5 despite far higher cost and lines", quietest)
	}
	if d.Team.LinesAdded != 5040 {
		t.Fatalf("Team.LinesAdded = %d, want 5040 -- the team's total, not any member's", d.Team.LinesAdded)
	}
}

// TestBuildTeamOrderedAlphabeticallyNotByMagnitude: the panel is a list of people, and a
// list of people ordered by a number is a league table however the caption reads. The
// order is the rendered label's, so the busiest member has no reserved position.
func TestBuildTeamOrderedAlphabeticallyNotByMagnitude(t *testing.T) {
	for _, anonymize := range []bool{true, false} {
		d := Build(fixtureInputWithMembers(), "last 30 days", anonymize, nil, nil)
		if d.Team == nil || len(d.Team.Stats) < 2 {
			t.Fatalf("anonymize=%v: Team = %+v, want at least 2 member stats", anonymize, d.Team)
		}
		labels := make([]string, len(d.Team.Stats))
		for i, s := range d.Team.Stats {
			labels[i] = s.Member
		}
		if !sort.StringsAreSorted(labels) {
			t.Errorf("anonymize=%v: Team.Stats must be ordered alphabetically by label, got %q", anonymize, labels)
		}
	}
}

// TestSortMembersByLabelIgnoresSessionCount states the ordering contract on its own,
// without depending on which pseudonym this install happens to derive: a 99-session member
// sorts last when their label does.
func TestSortMembersByLabelIgnoresSessionCount(t *testing.T) {
	stats := []TeamStat{
		{Member: "member-zzzz", Sessions: 99},
		{Member: "member-aaaa", Sessions: 1},
	}
	sortMembersByLabel(stats)
	if stats[0].Member != "member-aaaa" || stats[1].Member != "member-zzzz" {
		t.Fatalf("sortMembersByLabel = %+v, want label order, not session order", stats)
	}
}

// TestTeamRowCarriesNoOutputFigure: a member's row shows how often they engaged and nothing
// else. Pseudonymous is not anonymous to a colleague who knows the roster, so lines and
// spend on a per-member row are a productivity comparison however the list is sorted (B141).
func TestTeamRowCarriesNoOutputFigure(t *testing.T) {
	d := Build(fixtureInputWithMembers(), "last 30 days", false, nil, nil)
	if d.Team == nil {
		t.Fatal("Team = nil, want a breakdown when usage carries member data")
	}
	fields := reflect.TypeOf(TeamStat{})
	for i := range fields.NumField() {
		switch name := fields.Field(i).Name; name {
		case "Member", "Sessions", "Frac":
		default:
			t.Errorf("TeamStat carries %q: a per-member row may show engagement only", name)
		}
	}
}

// TestBuildTeamAlwaysPseudonymized: no value of Build's anonymize argument prints a real
// member name here. That flag reveals project names; a roster of real names beside
// proportional bars is the per-named-individual ranking the Refusals rule out, so the
// panel has no real-name path at all (see buildTeam).
func TestBuildTeamAlwaysPseudonymized(t *testing.T) {
	for _, anonymize := range []bool{true, false} {
		d := Build(fixtureInputWithMembers(), "last 30 days", anonymize, nil, nil)
		if d.Team == nil {
			t.Fatalf("anonymize=%v: Team = nil, want a breakdown when usage carries member data", anonymize)
		}
		for _, s := range d.Team.Stats {
			if s.Member == "alice" || s.Member == "bob" {
				t.Fatalf("anonymize=%v: Team member labels must never be real names: %+v", anonymize, d.Team.Stats)
			}
			if s.Member == "(local)" {
				continue // the fixture's inherited un-tagged rows form their own local group.
			}
			if !strings.HasPrefix(s.Member, "member-") {
				t.Fatalf("anonymize=%v: Team member label %q must look like a member pseudonym", anonymize, s.Member)
			}
		}
	}
}

func TestBuildTeamLocalRowsLabeledLocalNeverPseudonymized(t *testing.T) {
	in := fixtureInputWithMembers()
	in.Usage = append(in.Usage, store.UsageRow{
		Day: "2026-07-12", Tool: "claude-code", Model: "claude-sonnet-4-5", Project: "web", Member: "",
		In: 10, LinesAdded: 1,
	})
	in = analyze.BuildInput(in.Usage, in.Sessions, in.Prices, in.Now, in.Recent, in.Delegation)
	d := Build(in, "last 30 days", true, nil, nil)
	found := false
	for _, s := range d.Team.Stats {
		if s.Member == "(local)" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Team.Stats must label the empty-member group \"(local)\", never a pseudonym: %+v", d.Team.Stats)
	}
}

func TestRenderHTMLTeamSectionPresentWithMemberData(t *testing.T) {
	for _, anonymize := range []bool{true, false} {
		var buf bytes.Buffer
		if err := RenderHTML(&buf, Build(fixtureInputWithMembers(), "last 30 days", anonymize, nil, nil)); err != nil {
			t.Fatal(err)
		}
		html := buf.String()
		if !strings.Contains(html, `class="team"`) {
			t.Fatalf("anonymize=%v: dashboard HTML must include the Team section when usage carries member data: %s", anonymize, html)
		}
		if !strings.Contains(html, "this panel is not a scoreboard") {
			t.Fatalf("anonymize=%v: Team section must carry its honesty caption: %s", anonymize, html)
		}
		if !strings.Contains(html, "member-") {
			t.Fatalf("anonymize=%v: Team section must show pseudonymized member labels: %s", anonymize, html)
		}
		// The rendered page, not just the built Data: the whole finding was that a
		// --no-anonymize render put real names in front of a reader.
		for _, name := range []string{">alice<", ">bob<"} {
			if strings.Contains(html, name) {
				t.Fatalf("anonymize=%v: rendered dashboard names a real member (%s): %s", anonymize, name, html)
			}
		}
	}
}

func TestRenderHTMLTeamSectionAbsentWithoutMemberData(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderHTML(&buf, Build(fixtureInput(), "last 30 days", true, fixtureSubpaths(), nil)); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), `class="team"`) {
		t.Fatal("dashboard HTML must omit the Team section entirely for a purely local store")
	}
}
