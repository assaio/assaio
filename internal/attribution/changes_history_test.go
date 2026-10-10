package attribution

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/layer"
)

func historyPR() *PullRequests {
	prs := reviewedPR(9)
	p := prs.Requests[0].Payload.(event.PullRequest)
	p.HistoryState, p.CheckState, p.Checks, p.ChecksListed = "read", "success", 1, 1
	prs.Requests[0].Payload = p
	l := listed(9, hashOf("a"))
	l.Payload = event.PullRequestCommit{Number: 9, Commit: hashOf("a"), Suites: &event.Population{Total: 3, Listed: 1}}
	prs.Listings = []event.Event{l}
	suite := event.CheckSuite{Number: 9, Commit: hashOf("a"), State: "completed", Conclusion: "failure", Runs: &event.Population{Total: 4, Listed: 1}}
	prs.Suites = []event.Event{forgeEvent(event.TypeCheckSuite, "CS_1", epoch.Add(hour), event.TimeStated, event.GrainSuite, suite)}
	check := event.Check{Number: 9, Kind: "run", State: "completed", Conclusion: "failure", Commit: hashOf("a"), Suite: "CS_1", StartedAt: epoch, CompletedAt: epoch.Add(hour)}
	prs.HistoricalChecks = []event.Event{forgeEvent(event.TypeCheck, "CR_1", epoch.Add(hour), event.TimeStated, event.GrainCheck, check)}
	prs.Checks = []event.Event{forgeEvent(event.TypeCheck, "CR_1", epoch.Add(hour), event.TimeStated, event.GrainCheck,
		event.Check{Number: 9, Kind: "run", State: "completed", Conclusion: "success"})}
	prs.RepositoryTotal, prs.HistoryRead, prs.HistoryBackTo, prs.ReadBackTo = pointerTo(int64(250)), 100, epoch.Add(day), epoch
	return prs
}

func TestHistoricalConnectionsRetainBoundsSourceAndEveryConnectionPopulation(t *testing.T) {
	prs := historyPR()
	c := ChangesOf(prs, namesPR(9), "gh")
	pr := c.Linked[0]
	if c.PullRequestsRead != 1 || c.HistoryRead != 100 || c.RepositoryTotal == nil || *c.RepositoryTotal != 250 ||
		!c.ReadBackTo.Equal(epoch) || !c.HistoryBackTo.Equal(epoch.Add(day)) {
		t.Fatalf("independent read populations = %+v", c)
	}
	h := pr.HistoricalChecks
	if h == nil || h.ReadState != "read" || h.Layer != layer.Outcome || !reflect.DeepEqual(h.Commits, &event.Population{Total: 1, Listed: 1}) || len(h.ByCommit) != 1 {
		t.Fatalf("history commit population = %+v", h)
	}
	commit := h.ByCommit[0]
	if commit.Commit != hashOf("a") || !reflect.DeepEqual(commit.Suites, &event.Population{Total: 3, Listed: 1}) || len(commit.SuiteObservations) != 1 || len(commit.CheckObservations) != 1 {
		t.Fatalf("historical suite coverage = %+v", commit)
	}
	suite, check := commit.SuiteObservations[0], commit.CheckObservations[0]
	if !reflect.DeepEqual(suite.Payload.Runs, &event.Population{Total: 4, Listed: 1}) || check.Source.Name != "gh" || check.Provenance != event.Parsed || check.Privacy != event.LocalOnly ||
		!check.Payload.StartedAt.Equal(epoch) || !check.Payload.CompletedAt.Equal(epoch.Add(hour)) {
		t.Fatalf("historical run source or coverage = %+v / %+v", suite, check)
	}
	if pr.HeadChecks.Total != 1 || len(pr.HeadChecks.States) != 1 || pr.HeadChecks.States[0] != (StateCount{State: "run:success", Count: 1}) {
		t.Fatalf("history contaminated latest head checks: %+v", pr.HeadChecks)
	}
	if c.Rates[1].Value != nil || c.Rates[1].Numerator != nil || c.Rates[1].Reason != "comparable-pr-pipeline-history-unavailable" {
		t.Fatalf("complete commit observations became pipeline evidence: %+v", c.Rates[1])
	}
}

func TestUnavailableHistoricalConnectionsRemainNullAndKnownEmptyRemainsZero(t *testing.T) {
	for _, tt := range []struct {
		name     string
		adjust   func(*PullRequests)
		state    string
		fragment string
	}{
		{"outside read", setHistoryState("outside-history-read"), "outside-history-read", `"commits":null`},
		{"changed list", setHistoryState("commit-list-changed"), "commit-list-changed", `"byCommit":null`},
		{"missing state", setHistoryState(""), "unavailable", `"commits":null`},
		{"missing suites", func(prs *PullRequests) {
			p := prs.Listings[0].Payload.(event.PullRequestCommit)
			p.Suites = nil
			prs.Listings[0].Payload = p
		}, "read", `"suites":null,"suiteObservations":null,"checkObservations":null`},
		{"source-empty suites", func(prs *PullRequests) {
			p := prs.Listings[0].Payload.(event.PullRequestCommit)
			p.Suites = &event.Population{}
			prs.Listings[0].Payload = p
			prs.Suites, prs.HistoricalChecks = nil, nil
		}, "read", `"suites":{"total":0,"listed":0},"suiteObservations":[],"checkObservations":[]`},
		{"missing runs", func(prs *PullRequests) {
			p := prs.Suites[0].Payload.(event.CheckSuite)
			p.Runs = nil
			prs.Suites[0].Payload = p
			prs.HistoricalChecks = nil
		}, "read", `"runs":null`},
		{"source-empty runs", func(prs *PullRequests) {
			p := prs.Suites[0].Payload.(event.CheckSuite)
			p.Runs = &event.Population{}
			prs.Suites[0].Payload = p
			prs.HistoricalChecks = nil
		}, "read", `"runs":{"total":0,"listed":0}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prs := historyPR()
			tt.adjust(prs)
			h := ChangesOf(prs, namesPR(9), "gh").Linked[0].HistoricalChecks
			body, err := json.Marshal(h)
			if err != nil {
				t.Fatal(err)
			}
			if h.ReadState != tt.state || !strings.Contains(string(body), tt.fragment) {
				t.Fatalf("history lost absence or known zero: %s, want %s", body, tt.fragment)
			}
			if tt.name == "missing runs" && h.ByCommit[0].CheckObservations != nil {
				t.Fatal("missing run connections became an empty run population")
			}
		})
	}
}

func setHistoryState(state string) func(*PullRequests) {
	return func(prs *PullRequests) {
		p := prs.Requests[0].Payload.(event.PullRequest)
		p.HistoryState = state
		prs.Requests[0].Payload = p
	}
}

func TestHistoricalFactsAreOrderedWithoutChangingSourceConnections(t *testing.T) {
	prs := historyPR()
	other := listed(9, hashOf("b"))
	other.Payload = event.PullRequestCommit{Number: 9, Commit: hashOf("b"), Suites: nil}
	prs.Listings = append(prs.Listings, other)
	p := prs.Requests[0].Payload.(event.PullRequest)
	p.Commits, p.Listed = 120, 2
	prs.Requests[0].Payload = p
	first := ChangesOf(prs, namesPR(9), "gh")
	slices.Reverse(prs.Listings)
	slices.Reverse(prs.Suites)
	slices.Reverse(prs.HistoricalChecks)
	second := ChangesOf(prs, namesPR(9), "gh")
	if !reflect.DeepEqual(first, second) || first.Linked[0].HistoricalChecks.Commits.Total != 120 || first.Linked[0].HistoricalChecks.Commits.Listed != 2 {
		t.Fatal("input order or missing suites erased source commit coverage")
	}
}
