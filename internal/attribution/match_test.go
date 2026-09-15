package attribution

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
)

func TestMatchPassesEveryConformanceScenario(t *testing.T) {
	for _, scenario := range Corpus() {
		t.Run(scenario.Name, func(t *testing.T) {
			fixture := buildOrSkip(t, &scenario)
			results := Match(fixtureSessions(&fixture), fixture.Commits, fixture.Confirmed, epoch.Add(7*day))
			if violations := Check(&scenario, &fixture, linksFrom(results)); len(violations) != 0 {
				t.Fatalf("engine broke the conformance corpus:\n%s", Report(violations))
			}
		})
	}
}

func TestMatchIsStableAcrossInputOrderAndReplay(t *testing.T) {
	session := Session{ID: "s1", Project: "repo", Tool: "codex", StartedAt: epoch, EndedAt: epoch.Add(hour)}
	commits := []event.Event{
		commitEvent("b", epoch.Add(hour+time.Minute)),
		commitEvent("a", epoch.Add(hour+time.Minute)),
	}
	want := Match([]Session{session}, commits, nil, epoch.Add(day))
	slices.Reverse(commits)
	for i := range 2 {
		got := Match([]Session{session}, commits, nil, epoch.Add(day))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("replay %d changed result:\n got %#v\nwant %#v", i, got, want)
		}
	}
}

func TestMatchIncludesTheFollowingWindowBoundary(t *testing.T) {
	session := Session{ID: "s1", Project: "repo", StartedAt: epoch, EndedAt: epoch.Add(hour)}
	boundary := session.EndedAt.Add(DefaultMaxGap)
	results := Match([]Session{session}, []event.Event{
		commitEvent("inside", boundary),
		commitEvent("outside", boundary.Add(time.Second)),
	}, nil, boundary.Add(time.Hour))
	if got := candidateIDs(results[0].Candidates); !slices.Equal(got, []string{"inside"}) {
		t.Fatalf("boundary candidates = %v, want [inside]", got)
	}
}

func TestMatchKeepsAlternativesBehindAConfirmation(t *testing.T) {
	session := Session{ID: "s1", Project: "repo", StartedAt: epoch, EndedAt: epoch.Add(hour)}
	commits := []event.Event{
		commitEvent("a", epoch.Add(hour+time.Minute)),
		commitEvent("b", epoch.Add(hour+2*time.Minute)),
	}
	result := Match([]Session{session}, commits, map[string]string{"s1": "b"}, epoch.Add(day))[0]
	if result.Method != methodConfirmed || result.Provenance != event.Manual || result.Ambiguous {
		t.Fatalf("confirmed result = %#v", result)
	}
	if got := candidateIDs(result.Candidates); !slices.Equal(got, []string{"b"}) {
		t.Fatalf("confirmed candidates = %v, want [b]", got)
	}
	if got := candidateIDs(result.Alternatives); !slices.Equal(got, []string{"a"}) {
		t.Fatalf("confirmed alternatives = %v, want [a]", got)
	}
}

func TestMatchAbstainsWithSpecificReasons(t *testing.T) {
	now := epoch.Add(10 * day)
	tests := []struct {
		name    string
		session Session
		reason  string
	}{
		{"unknown project", Session{ID: "s1", StartedAt: epoch, EndedAt: epoch.Add(hour)}, "project-or-session-unavailable"},
		{"closed empty window", Session{ID: "s1", Project: "repo", StartedAt: epoch, EndedAt: epoch.Add(hour)}, "no-commit-candidate"},
		{"invalid window", Session{ID: "s1", Project: "repo", StartedAt: epoch.Add(hour), EndedAt: epoch}, "invalid-session-window"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Match([]Session{test.session}, nil, nil, now)[0]
			if result.Status != statusUnmatched || result.Confidence != confidenceInsufficient || result.Reason != test.reason {
				t.Fatalf("result = %#v, want unmatched reason %q", result, test.reason)
			}
		})
	}
}

func TestResultJSONHasNoContentOrPersonSurface(t *testing.T) {
	session := Session{ID: "s1", Project: "repo", Tool: "claude-code", StartedAt: epoch, EndedAt: epoch.Add(hour)}
	data, err := json.Marshal(Match([]Session{session}, []event.Event{
		commitEvent("abc", epoch.Add(time.Minute)),
	}, nil, epoch.Add(day)))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{`"prompt"`, `"response"`, `"code"`, `"diff"`, `"message"`, `"branch"`, `"member"`, `"author"`, `"email"`, `"rank"`, `"score"`} {
		if strings.Contains(strings.ToLower(string(data)), forbidden) {
			t.Fatalf("serialized result exposes forbidden surface %q: %s", forbidden, data)
		}
	}
}

func fixtureSessions(fixture *Fixture) []Session {
	byID := map[string]Session{}
	for i := range fixture.Sessions {
		record := &fixture.Sessions[i]
		session, found := byID[record.SessionID]
		if !found || record.Timestamp.Before(session.StartedAt) {
			session.StartedAt = record.Timestamp
		}
		if !found || record.Timestamp.After(session.EndedAt) {
			session.EndedAt = record.Timestamp
		}
		session.ID, session.Project, session.Tool = record.SessionID, record.Project, record.Tool
		byID[record.SessionID] = session
	}
	out := make([]Session, 0, len(byID))
	for _, session := range byID {
		out = append(out, session)
	}
	return out
}

func linksFrom(results []Result) Links {
	links := Links{}
	for i := range results {
		if len(results[i].Candidates) > 0 {
			links[results[i].Session.ID] = Link{Commits: candidateIDs(results[i].Candidates), Ambiguous: results[i].Ambiguous}
		}
	}
	return links
}

func candidateIDs(candidates []Candidate) []string {
	ids := make([]string, 0, len(candidates))
	for i := range candidates {
		ids = append(ids, candidates[i].CommitID)
	}
	return ids
}

func commitEvent(id string, at time.Time) event.Event {
	return event.Event{
		SpecVersion: event.SpecVersion, Type: event.TypeCommit, ID: id,
		Source: event.Source{Name: "git", Build: "test"}, OccurredAt: at, ObservedAt: at.Add(time.Hour),
		TimeSource: event.TimeStated, Grain: event.GrainCommit, Privacy: event.LocalOnly,
		Provenance: event.Parsed, Subject: event.Subject{Project: "repo"}, Payload: event.Commit{},
	}
}
