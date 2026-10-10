package attribution

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/layer"
)

func hashOf(tag string) string { return strings.Repeat(tag, 40/len(tag))[:40] }

func pullRequest(number int64, state string, updated time.Time, merge string, commits, listed int64) event.Event {
	pr := event.PullRequest{Number: number, State: state, MergeCommit: merge, Commits: commits, Listed: listed}
	if state == event.ChangeMerged {
		pr.MergedAt = updated
	}
	return forgeEvent(event.TypePullRequest, "PR_"+strconv.FormatInt(number, 10), updated, event.TimeStated, event.GrainChange, pr)
}

func listed(number int64, hash string) event.Event {
	return forgeEvent(event.TypePullRequestCommit, hash, epoch.Add(10*day), event.TimeIngestTime, event.GrainCommit,
		event.PullRequestCommit{Number: number, Commit: hash})
}

// TestAnUnreadableListedCommitExplainsAnEmptySession: a session with no candidate while a pull
// request updated after it started lists a commit this clone cannot read may have its work there;
// one updated before it started cannot.
func TestAnUnreadableListedCommitExplainsAnEmptySession(t *testing.T) {
	session := Session{ID: "s1", Project: "repo", StartedAt: epoch, EndedAt: epoch.Add(hour)}
	gone := hashOf("d")
	landing := []event.Event{forgeCommit("sq", epoch.Add(3*day), epoch.Add(3*day), 1)}
	for _, tt := range []struct {
		name    string
		updated time.Time
		prs     PullRequests
		commits []event.Event
		want    string
	}{
		{"absent here", epoch.Add(2 * hour), PullRequests{NotLocal: []string{gone}}, nil, ReasonListedUnreadable},
		{"unreadable here", epoch.Add(2 * hour), PullRequests{Unreadable: []string{gone}}, nil, ReasonListedUnreadable},
		{"updated before the session", epoch.Add(-hour), PullRequests{NotLocal: []string{gone}}, nil, reasonNoCandidate},
		{"only a teammate's, fetched", epoch.Add(2 * hour), PullRequests{NotInReflog: []string{gone}}, nil, reasonNoCandidate},
		{"made here before the window", epoch.Add(2 * hour), PullRequests{BeforeWindow: []string{gone}}, nil, reasonNoCandidate},
		{"a later landing keeps its own reason", epoch.Add(2 * hour), PullRequests{NotLocal: []string{gone}}, landing, ReasonLaterLanding},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prs := tt.prs
			prs.Requests = []event.Event{pullRequest(1, event.ChangeOpen, tt.updated, "", 1, 1)}
			prs.Listings = []event.Event{listed(1, gone)}
			results := Match([]Session{session}, tt.commits, nil, epoch.Add(10*day))
			LinkChanges(results, tt.commits, &prs)
			if results[0].Reason != tt.want || results[0].Status != statusUnmatched {
				t.Fatalf("result = %s reason %q, want unmatched with %q", results[0].Status, results[0].Reason, tt.want)
			}
		})
	}
}

// TestOneChangeNeedsEveryCandidateInIt: the order candidates arrive in must not decide whether an
// unlinked one is noticed, and two shared pull requests name neither.
func TestOneChangeNeedsEveryCandidateInIt(t *testing.T) {
	in := func(numbers ...int64) Candidate {
		c := Candidate{Changes: []ChangeLink{}}
		for _, n := range numbers {
			c.Changes = append(c.Changes, ChangeLink{Number: n, Via: viaListed})
		}
		return c
	}
	for _, tt := range []struct {
		name       string
		candidates []Candidate
		want       int64
	}{
		{"one pull request", []Candidate{in(1), in(1)}, 1},
		{"an unlinked candidate last", []Candidate{in(1), in()}, 0},
		{"an unlinked candidate first", []Candidate{in(), in(1)}, 0},
		{"a stack shared by both", []Candidate{in(1, 2), in(1, 2)}, 0},
		{"a stack narrowed to one", []Candidate{in(1, 2), in(2)}, 2},
		{"no candidate", nil, 0},
	} {
		if got := oneChange(tt.candidates); got != tt.want {
			t.Errorf("%s: oneChange = %d, want %d", tt.name, got, tt.want)
		}
	}
}

// TestLinkingChangesNoVerdict: the pull request a session is placed in is beside its status, never
// a new status -- ambiguity and every summary fraction stay what the commits decided.
func TestLinkingChangesNoVerdict(t *testing.T) {
	session := Session{ID: "s1", Project: "repo", StartedAt: epoch, EndedAt: epoch.Add(hour)}
	commits := []event.Event{commitEvent(hashOf("a"), epoch.Add(2*hour)), commitEvent(hashOf("b"), epoch.Add(3*hour))}
	before := Match([]Session{session}, commits, nil, epoch.Add(10*day))
	after := Match([]Session{session}, commits, nil, epoch.Add(10*day))
	LinkChanges(after, commits, &PullRequests{
		Requests: []event.Event{pullRequest(7, event.ChangeOpen, epoch.Add(4*hour), "", 2, 2)},
		Listings: []event.Event{listed(7, hashOf("a")), listed(7, hashOf("b"))},
	})
	if after[0].Change != 7 {
		t.Fatalf("change = %d, want both candidates' one pull request #7", after[0].Change)
	}
	if after[0].Status != before[0].Status || after[0].Ambiguous != before[0].Ambiguous ||
		after[0].Confidence != before[0].Confidence || Summarize(after) != Summarize(before) {
		t.Fatalf("linking moved the verdict: %s/%v/%s -> %s/%v/%s", before[0].Status, before[0].Ambiguous,
			before[0].Confidence, after[0].Status, after[0].Ambiguous, after[0].Confidence)
	}
}

func TestTheChangesBlockCountsEveryAbsence(t *testing.T) {
	session := Session{ID: "s1", Project: "repo", StartedAt: epoch, EndedAt: epoch.Add(hour)}
	mine := commitEvent(hashOf("a"), epoch.Add(30*time.Minute))
	prs := &PullRequests{
		Requests: []event.Event{
			pullRequest(3, event.ChangeClosed, epoch.Add(2*hour), "", 1, 1),
			pullRequest(1, event.ChangeMerged, epoch.Add(2*hour), hashOf("f"), 250, 100),
			pullRequest(2, event.ChangeOpen, epoch.Add(2*hour), "", 1, 1),
		},
		Listings: []event.Event{listed(3, hashOf("a")), listed(1, hashOf("a"))},
		MadeHere: 4, NotInReflog: []string{hashOf("b")}, NotLocal: []string{hashOf("c"), hashOf("d")},
		Unreadable: []string{hashOf("e")}, BeforeWindow: []string{hashOf("g")},
		ObservedAt: epoch.Add(10 * day), ReadBackTo: epoch.Add(-day),
	}
	results := Match([]Session{session}, []event.Event{mine}, nil, epoch.Add(10*day))
	LinkChanges(results, []event.Event{mine}, prs)
	got := ChangesOf(prs, results, "gh")
	want := Changes{
		Source: "gh", ObservedAt: prs.ObservedAt, ReadBackTo: prs.ReadBackTo, StateLayer: layer.Outcome,
		PullRequestsRead: 3, PullRequestsNamed: 2, PullRequestsUnmatched: 1,
		CommitListsCut: 1, ListedMadeHere: 4, ListedNotInReflog: 1, ListedNotLocal: 2,
		ListedUnreadable: 1, ListedBeforeWindow: 1, SessionsAcrossChanges: 1,
	}
	gotLinked := got.Linked
	got.Linked = nil
	got.Rates = nil
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("changes = %+v, want %+v", *got, want)
	}
	want1 := LinkedPullRequest{Number: 1, State: event.ChangeMerged, MergedAt: epoch.Add(2 * hour), NamedBy: namedByCandidate}
	want3 := LinkedPullRequest{Number: 3, State: event.ChangeClosed, NamedBy: namedByCandidate}
	for i := range gotLinked {
		gotLinked[i].ReviewObservations, gotLinked[i].RequestedChangesRevisions, gotLinked[i].ReviewRounds = nil, nil, nil
		gotLinked[i].MergeMethod, gotLinked[i].HistoricalChecks = nil, nil
	}
	if !reflect.DeepEqual(gotLinked, []LinkedPullRequest{want1, want3}) {
		t.Fatalf("linked = %+v, want #1 merged and #3 closed, by number, and #2 not listed", gotLinked)
	}
}

func TestLinkedPullRequestReportsDeliveryCoverageWithoutAssigningItToASession(t *testing.T) {
	pr := pullRequest(9, event.ChangeMerged, epoch.Add(2*hour), hashOf("f"), 1, 1)
	p := pr.Payload.(event.PullRequest)
	p.ReviewsAvailable, p.Reviews, p.ReviewsListed = true, 101, 1
	p.CheckState, p.Checks, p.ChecksListed = "failure", 102, 1
	pr.Payload = p
	prs := &PullRequests{
		Requests:   []event.Event{pr},
		Reviews:    []event.Event{{Payload: event.Review{Number: 9, State: "changes_requested"}}},
		Checks:     []event.Event{{Payload: event.Check{Number: 9, Kind: "run", State: "completed", Conclusion: "failure"}}},
		ObservedAt: epoch.Add(10 * day),
	}
	results := []Result{{Candidates: []Candidate{{Changes: []ChangeLink{{Number: 9}}}}, Change: 9}}
	changes := ChangesOf(prs, results, "gh")
	if changes.PullRequestsNamed != 1 || changes.PullRequestsUnmatched != 0 || len(changes.Linked) != 1 {
		t.Fatalf("changes = %+v", changes)
	}
	linked := changes.Linked[0]
	if linked.Reviews == nil || linked.Reviews.Total != 101 || linked.Reviews.Listed != 1 ||
		linked.Reviews.States[0].State != "changes_requested" || linked.HeadChecks == nil ||
		linked.HeadChecks.Total != 102 || linked.HeadChecks.Listed != 1 || linked.HeadChecks.State != "failure" {
		t.Fatalf("linked delivery = %+v", linked)
	}
	var out strings.Builder
	if err := renderLinked(&out, changes); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"reviews: 1/101 read (incomplete", "changes_requested=1", "contexts: 1/102 read (incomplete", "run:failure=1"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("linked output lacks %q: %s", want, out.String())
		}
	}
}

// TestACandidateLineNeverCarriesAState: a state beside a session reads as the session's outcome, so
// the state is printed once, in the pull-request block, labelled as the forge's.
func TestACandidateLineNeverCarriesAState(t *testing.T) {
	session := Session{ID: "s1", Project: "repo", StartedAt: epoch, EndedAt: epoch.Add(hour)}
	commits := []event.Event{commitEvent(hashOf("a"), epoch.Add(30*time.Minute)), commitEvent(hashOf("b"), epoch.Add(40*time.Minute))}
	prs := &PullRequests{
		Requests:   []event.Event{pullRequest(98, event.ChangeMerged, epoch.Add(2*hour), hashOf("f"), 1, 1)},
		Listings:   []event.Event{listed(98, hashOf("a"))},
		ObservedAt: epoch.Add(10 * day),
	}
	results := Match([]Session{session}, commits, nil, epoch.Add(10*day))
	LinkChanges(results, commits, prs)
	var out strings.Builder
	doc := Document{Algorithm: AlgorithmChanges, Results: results, Changes: ChangesOf(prs, results, "gh")}
	if err := RenderText(&out, &doc); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{
		"pull request #98 listed (source=gh)",
		"in no pull request commit list read",
		"state as gh reported it at",
		"the pull request's outcome, not a session's",
		"    #98 merged",
		"compare it only with another session-commit/v3 document",
		"A merged pull request says the change landed, not that this session's lines are in it.",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("output is missing %q:\n%s", want, text)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "candidate ") && strings.Contains(line, "merged") {
			t.Errorf("a candidate line carries a state: %q", line)
		}
	}
}

func TestTheOfflineFooterSaysPullRequestsNeedTheFlag(t *testing.T) {
	var out strings.Builder
	if err := RenderText(&out, &Document{Algorithm: Algorithm}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Pull request, review and head-check observations are read only with --github") || strings.Contains(out.String(), "pull requests (") {
		t.Fatalf("offline output:\n%s", out.String())
	}
}

// TestAV3CandidateInNoListSaysSo: offline, a candidate has no changes field at all; with pull
// requests read, an empty list says the commit is in no commit list read.
func TestAV3CandidateInNoListSaysSo(t *testing.T) {
	session := Session{ID: "s1", Project: "repo", StartedAt: epoch, EndedAt: epoch.Add(hour)}
	commits := []event.Event{commitEvent(hashOf("a"), epoch.Add(30*time.Minute))}
	offline := Match([]Session{session}, commits, nil, epoch.Add(10*day))
	linked := Match([]Session{session}, commits, nil, epoch.Add(10*day))
	LinkChanges(linked, commits, &PullRequests{})
	for name, tt := range map[string]struct {
		results []Result
		want    bool
	}{"offline": {offline, false}, "with pull requests": {linked, true}} {
		b, err := json.Marshal(tt.results[0].Candidates[0])
		if err != nil {
			t.Fatal(err)
		}
		if got := strings.Contains(string(b), `"changes":[]`); got != tt.want {
			t.Errorf("%s: %s", name, b)
		}
	}
}

// TestTheReadBoundIsSaid: a read the page bound stopped says so, since the window was not covered.
func TestTheReadBoundIsSaid(t *testing.T) {
	for _, tt := range []struct {
		backTo time.Time
		want   bool
	}{{epoch, false}, {epoch.Add(day), true}} {
		var out strings.Builder
		doc := Document{Algorithm: AlgorithmChanges, Since: epoch, Changes: &Changes{ReadBackTo: tt.backTo, Linked: []LinkedPullRequest{}}}
		if err := RenderText(&out, &doc); err != nil {
			t.Fatal(err)
		}
		if got := strings.Contains(out.String(), "20-page bound"); got != tt.want {
			t.Errorf("read back to %s: bound said %v, want %v", tt.backTo, got, tt.want)
		}
	}
}
