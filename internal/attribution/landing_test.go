package attribution

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
)

// forgeCommit is a commit the forge wrote: committed at committed, authored at authored, with
// parents parents.
func forgeCommit(id string, authored, committed time.Time, parents int64) event.Event {
	e := commitEvent(id, committed)
	e.Payload = event.Commit{Parents: parents, AuthoredAt: authored, CommittedByForge: true}
	return e
}

// amended is a commit written at authored and rewritten locally at committed.
func amended(id string, authored, committed time.Time) event.Event {
	e := commitEvent(id, committed)
	e.Payload = event.Commit{Parents: 1, AuthoredAt: authored}
	return e
}

func TestLandingRules(t *testing.T) {
	session := Session{ID: "s1", Project: "repo", StartedAt: epoch, EndedAt: epoch.Add(hour)}
	tests := []struct {
		name               string
		commit             event.Event
		method, confidence string
		relation           string
		evidenceAt         time.Time
		gap                int64
	}{
		{
			"a squash landing while the session runs is low, never an overlap",
			forgeCommit("sq", epoch.Add(30*time.Minute), epoch.Add(30*time.Minute), 1),
			methodLanded, confidenceLow, relationLanded, epoch.Add(30 * time.Minute), 0,
		},
		{
			"a squash landing after the session is low",
			forgeCommit("sq", epoch.Add(2*hour), epoch.Add(2*hour), 1),
			methodLanded, confidenceLow, relationLanded, epoch.Add(2 * hour), 3600,
		},
		{
			"a forge rebase is judged at its author time",
			forgeCommit("rb", epoch.Add(30*time.Minute), epoch.Add(3*day), 1),
			methodOverlap, confidenceMedium, relationOverlap, epoch.Add(30 * time.Minute), 0,
		},
		{
			"a forge rebase written after the session is measured from when it was written",
			forgeCommit("rb", epoch.Add(2*hour), epoch.Add(3*day), 1),
			methodFollowing, confidenceLow, relationFollowing, epoch.Add(2 * hour), 3600,
		},
		{
			"an amend is measured from its first write",
			amended("c1", epoch.Add(2*hour), epoch.Add(3*day)),
			methodFollowing, confidenceLow, relationFollowing, epoch.Add(2 * hour), 3600,
		},
		{
			"a commit whose clocks disagree is measured from the earlier one",
			amended("c1", epoch.Add(3*hour), epoch.Add(2*hour)),
			methodFollowing, confidenceLow, relationFollowing, epoch.Add(2 * hour), 3600,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Match([]Session{session}, []event.Event{tt.commit}, nil, epoch.Add(10*day))[0]
			if r.Status != statusMatched || r.Method != tt.method || r.Confidence != tt.confidence {
				t.Fatalf("result = %s/%s/%s, want matched/%s/%s", r.Status, r.Method, r.Confidence, tt.method, tt.confidence)
			}
			c := r.Candidates[0]
			if c.Relation != tt.relation || !c.EvidenceAt.Equal(tt.evidenceAt) || !c.OccurredAt.Equal(tt.commit.OccurredAt) ||
				c.GapSeconds != tt.gap {
				t.Fatalf("candidate = %s judged at %s (committed %s) gap %ds, want %s judged at %s gap %ds",
					c.Relation, c.EvidenceAt, c.OccurredAt, c.GapSeconds, tt.relation, tt.evidenceAt, tt.gap)
			}
		})
	}
}

// TestWhyALandingLeavesASessionUnmatched: only a commit the forge landed, in the session's own
// repository, after its following window says the session's work may have shipped later.
func TestWhyALandingLeavesASessionUnmatched(t *testing.T) {
	session := Session{ID: "s1", Project: "repo", StartedAt: epoch, EndedAt: epoch.Add(hour)}
	elsewhere := forgeCommit("sq", epoch.Add(3*day), epoch.Add(3*day), 1)
	elsewhere.Subject.Project = "other"
	tests := []struct {
		name   string
		commit event.Event
		reason string
	}{
		{
			"a forge merge commit is no candidate",
			forgeCommit("mg", epoch.Add(30*time.Minute), epoch.Add(30*time.Minute), 2), "no-commit-candidate",
		},
		{
			"a squash beyond the window leaves a reason, not an empty verdict",
			forgeCommit("sq", epoch.Add(3*day), epoch.Add(3*day), 1), ReasonLaterLanding,
		},
		{
			"an ordinary commit beyond the window is no landing",
			commitEvent("c1", epoch.Add(3*day)), "no-commit-candidate",
		},
		{
			"a forge rebase beyond the window is no landing",
			forgeCommit("rb", epoch.Add(3*day-hour), epoch.Add(3*day), 1), "no-commit-candidate",
		},
		{"a squash landing in another project is no landing for this one", elsewhere, "no-commit-candidate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Match([]Session{session}, []event.Event{tt.commit}, nil, epoch.Add(10*day))[0]
			if r.Status != statusUnmatched || r.Confidence != confidenceInsufficient || r.Reason != tt.reason {
				t.Fatalf("result = %s/%s reason %q, want unmatched/insufficient reason %q",
					r.Status, r.Confidence, r.Reason, tt.reason)
			}
		})
	}
}

// TestAnAmendReachesBothSessions: the author time places a commit in the session that wrote it,
// the committer time in the one that amended it; each is a medium overlap for its own session.
func TestAnAmendReachesBothSessions(t *testing.T) {
	writer := Session{ID: "w", Project: "repo", StartedAt: epoch, EndedAt: epoch.Add(hour)}
	amender := Session{ID: "a", Project: "repo", StartedAt: epoch.Add(3 * day), EndedAt: epoch.Add(3*day + hour)}
	c := commitEvent("c1", epoch.Add(3*day+10*time.Minute))
	c.Payload = event.Commit{Parents: 1, AuthoredAt: epoch.Add(30 * time.Minute)}
	for _, r := range Match([]Session{writer, amender}, []event.Event{c}, nil, epoch.Add(10*day)) {
		if r.Status != statusMatched || r.Confidence != confidenceMedium || len(r.Candidates) != 1 {
			t.Errorf("session %s = %s/%s with %d candidate(s), want a medium match on c1",
				r.Session.ID, r.Status, r.Confidence, len(r.Candidates))
		}
	}
}

// TestTheForgeLineIsAlwaysPrinted: a reader must be able to tell "no forge commit" from "a forge
// this build does not recognise", so the scope prints every run, and the coverage caveat only when
// detection recognised something.
func TestTheForgeLineIsAlwaysPrinted(t *testing.T) {
	for _, tt := range []struct {
		name  string
		forge Forge
		want  []string
	}{
		{
			"nothing recognised",
			Forge{Detection: ForgeDetection},
			[]string{"forge (github.com-committer only): none recognised; merges by other forges are judged as local commits"},
		},
		{
			"squashes landed",
			Forge{Detection: ForgeDetection, LandedCommits: 3, LaterLanding: 1},
			[]string{"3 landed · 0 rebased · 0 merge commit(s) excluded · 1 session(s)", "coverage measures what commit timing can link"},
		},
		{
			"only rebases",
			Forge{Detection: ForgeDetection, RebasedCommits: 2},
			[]string{"0 landed · 2 rebased", "coverage measures what commit timing can link"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out strings.Builder
			if err := RenderText(&out, &Document{Algorithm: Algorithm, Forge: tt.forge}); err != nil {
				t.Fatal(err)
			}
			for _, want := range tt.want {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("output is missing %q:\n%s", want, out.String())
				}
			}
			if !tt.forge.Recognised() && strings.Contains(out.String(), "coverage measures") {
				t.Fatalf("the coverage caveat printed with nothing recognised:\n%s", out.String())
			}
		})
	}
}

// TestACandidateLineSaysWhatItsTimeMeans: a landing is never printed as "inside session", and a
// commit judged at the time it was written shows the time it reached its branch beside it.
func TestACandidateLineSaysWhatItsTimeMeans(t *testing.T) {
	for _, tt := range []struct {
		candidate Candidate
		want      string
	}{
		{Candidate{Relation: relationLanded, EvidenceAt: epoch, OccurredAt: epoch}, "landed by forge during session"},
		{Candidate{Relation: relationLanded, EvidenceAt: epoch, OccurredAt: epoch, GapSeconds: 90}, "landed by forge 90s after session"},
		{Candidate{Relation: relationFollowing, EvidenceAt: epoch, OccurredAt: epoch, GapSeconds: 60}, "60s after session"},
		{
			Candidate{Relation: relationOverlap, EvidenceAt: epoch, OccurredAt: epoch.Add(day)},
			epoch.Format(time.RFC3339) + " (written; committed " + epoch.Add(day).Format(time.RFC3339) + ") · inside session",
		},
	} {
		var out strings.Builder
		if err := renderCandidate(&out, "candidate", &tt.candidate); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), tt.want) {
			t.Errorf("line %q does not say %q", out.String(), tt.want)
		}
	}
}

func TestForgeOfCountsWhatDetectionFound(t *testing.T) {
	ordinary := commitEvent("o", epoch)
	commits := []event.Event{
		ordinary,
		forgeCommit("sq", epoch.Add(hour), epoch.Add(hour), 1),
		forgeCommit("rb", epoch, epoch.Add(hour), 1),
		forgeCommit("mg", epoch.Add(hour), epoch.Add(hour), 2),
	}
	results := []Result{{Reason: ReasonLaterLanding}, {Reason: "no-commit-candidate"}}
	want := Forge{Detection: ForgeDetection, LandedCommits: 1, RebasedCommits: 1, MergeCommits: 1, LaterLanding: 1}
	if got := ForgeOf(commits, results); got != want {
		t.Fatalf("ForgeOf = %+v, want %+v", got, want)
	}
}
