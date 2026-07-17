package dashboard

import (
	"bytes"
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

// TestBuildTeamRankedBySessionsNotCostOrLines locks in the honesty framing: adoption
// spread (session/engagement frequency), never a cost or lines-added scoreboard. bob out-
// spends and out-produces alice here, yet alice -- more sessions -- still ranks first.
// Deliberately self-contained (not fixtureInputWithMembers, which also carries the base
// fixture's un-tagged local rows) so the alice-vs-bob comparison is the only variable.
func TestBuildTeamRankedBySessionsNotCostOrLines(t *testing.T) {
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
	in := analyze.BuildInput(usage, sessions, fixturePrices(), fixtureNow, 7*24*time.Hour, analyze.Delegation{})
	d := Build(in, "last 30 days", false, nil, nil)
	if d.Team == nil || len(d.Team.Stats) != 2 {
		t.Fatalf("Team = %+v, want 2 member stats", d.Team)
	}
	if d.Team.Stats[0].Member != "alice" || d.Team.Stats[0].Sessions != 2 {
		t.Fatalf("Team.Stats[0] = %+v, want alice ranked first on 2 sessions despite bob's higher cost/lines", d.Team.Stats[0])
	}
	if d.Team.Stats[1].Member != "bob" || d.Team.Stats[1].Sessions != 1 {
		t.Fatalf("Team.Stats[1] = %+v, want bob ranked second", d.Team.Stats[1])
	}
	if d.Team.Stats[0].Frac != 1 {
		t.Fatalf("top-ranked member's Frac = %v, want 1 (scaled against the list's own max)", d.Team.Stats[0].Frac)
	}
	if d.Team.Stats[0].LinesAdded != 40 || d.Team.Stats[1].LinesAdded != 5000 {
		t.Fatalf("Stats = %+v, want alice=40 lines and bob=5000 lines preserved as data, just not the rank key", d.Team.Stats)
	}
}

func TestBuildTeamPseudonymizedByDefault(t *testing.T) {
	d := Build(fixtureInputWithMembers(), "last 30 days", true, nil, nil)
	if d.Team == nil {
		t.Fatal("Team = nil, want a breakdown when usage carries member data")
	}
	for _, s := range d.Team.Stats {
		if s.Member == "alice" || s.Member == "bob" {
			t.Fatalf("Team member labels must be pseudonymized by default: %+v", d.Team.Stats)
		}
		if s.Member == "(local)" {
			continue // the fixture's inherited un-tagged rows form their own local group.
		}
		if !strings.HasPrefix(s.Member, "member-") {
			t.Fatalf("Team member label %q must look like a member pseudonym", s.Member)
		}
	}
}

func TestBuildTeamRealNamesWhenNotAnonymized(t *testing.T) {
	d := Build(fixtureInputWithMembers(), "last 30 days", false, nil, nil)
	names := map[string]bool{}
	for _, s := range d.Team.Stats {
		names[s.Member] = true
	}
	if !names["alice"] || !names["bob"] {
		t.Fatalf("Team member labels must show real names when anonymize=false: %+v", d.Team.Stats)
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
	var buf bytes.Buffer
	if err := RenderHTML(&buf, Build(fixtureInputWithMembers(), "last 30 days", true, nil, nil)); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	if !strings.Contains(html, `class="team"`) {
		t.Fatalf("dashboard HTML must include the Team section when usage carries member data: %s", html)
	}
	if !strings.Contains(html, "aggregated, pseudonymized by default") {
		t.Fatalf("Team section must carry its honesty caption: %s", html)
	}
	if !strings.Contains(html, "member-") {
		t.Fatalf("Team section must show pseudonymized member labels by default: %s", html)
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
