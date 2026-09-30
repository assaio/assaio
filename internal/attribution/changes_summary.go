package attribution

import (
	"slices"
	"time"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/layer"
)

// Changes is what pull-request data added to a document: when it was read, what bounded the read,
// how many candidates it added, every count behind an absence, and the pull requests a result
// names. A pull request no result names is not listed and not counted apart: the document is about
// sessions, not a team's pull requests.
type Changes struct {
	Source     string    `json:"source"`
	ObservedAt time.Time `json:"observedAt"`
	ReadBackTo time.Time `json:"readBackTo"`
	// StateLayer is the measurement layer of the states in Linked: an outcome of each pull request,
	// never of a session linked to it.
	StateLayer       layer.Layer `json:"stateLayer"`
	PullRequestsRead int         `json:"pullRequestsRead"`
	CommitListsCut   int         `json:"commitListsCut"`
	// ListedMadeHere are the listed commits made here that session-commit/v2 does not see: they enlarge
	// the candidate set, so coverage is comparable only with another session-commit/v3 document.
	ListedMadeHere     int `json:"listedMadeHere"`
	ListedNotInReflog  int `json:"listedNotInReflog"`
	ListedNotLocal     int `json:"listedNotLocal"`
	ListedUnreadable   int `json:"listedUnreadable"`
	ListedBeforeWindow int `json:"listedBeforeWindow"`
	// SessionsInOneChange counts the results whose candidates share exactly one pull request, and
	// SessionsAcrossChanges those whose candidates span two or more.
	SessionsInOneChange   int                 `json:"sessionsInOneChange"`
	SessionsAcrossChanges int                 `json:"sessionsAcrossChanges"`
	Linked                []LinkedPullRequest `json:"linked"`
}

// LinkedPullRequest is a pull request's state as the forge reported it at Changes.ObservedAt, and
// whether a candidate names it or only an alternative does.
type LinkedPullRequest struct {
	Number   int64     `json:"number"`
	State    string    `json:"state"`
	MergedAt time.Time `json:"mergedAt,omitzero"`
	NamedBy  string    `json:"namedBy"`
}

const (
	namedByCandidate   = "candidate"
	namedByAlternative = "alternative"
)

// ChangesOf summarizes what prs added to results.
func ChangesOf(prs *PullRequests, results []Result, source string) *Changes {
	c := &Changes{
		Source: source, ObservedAt: prs.ObservedAt, ReadBackTo: prs.ReadBackTo, StateLayer: layer.Outcome,
		PullRequestsRead: len(prs.Requests), ListedMadeHere: prs.MadeHere, ListedNotInReflog: len(prs.NotInReflog),
		ListedNotLocal: len(prs.NotLocal), ListedUnreadable: len(prs.Unreadable), ListedBeforeWindow: len(prs.BeforeWindow),
		Linked: []LinkedPullRequest{},
	}
	named := namedBy(results)
	for i := range prs.Requests {
		pr, ok := prs.Requests[i].Payload.(event.PullRequest)
		if !ok {
			continue
		}
		if pr.Listed < pr.Commits {
			c.CommitListsCut++
		}
		if by, ok := named[pr.Number]; ok {
			c.Linked = append(c.Linked, LinkedPullRequest{Number: pr.Number, State: pr.State, MergedAt: pr.MergedAt, NamedBy: by})
		}
	}
	slices.SortFunc(c.Linked, func(a, b LinkedPullRequest) int { return int(a.Number - b.Number) })
	for i := range results {
		switch {
		case results[i].Change != 0:
			c.SessionsInOneChange++
		case len(candidateChanges(results[i].Candidates)) > 1:
			c.SessionsAcrossChanges++
		}
	}
	return c
}

// namedBy maps each pull request a result names to whether a candidate names it; an alternative
// alone is the weaker tie, since a squash landing near a session is one for every nearby session.
func namedBy(results []Result) map[int64]string {
	out := map[int64]string{}
	for i := range results {
		for _, n := range candidateChanges(results[i].Alternatives) {
			if _, ok := out[n]; !ok {
				out[n] = namedByAlternative
			}
		}
		for _, n := range candidateChanges(results[i].Candidates) {
			out[n] = namedByCandidate
		}
	}
	return out
}

func candidateChanges(set []Candidate) []int64 {
	var out []int64
	for i := range set {
		for _, n := range changeNumbers(set[i].Changes) {
			if !slices.Contains(out, n) {
				out = append(out, n)
			}
		}
	}
	return out
}
