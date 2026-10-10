package attribution

import (
	"slices"
	"time"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/layer"
)

// Changes is what pull-request data added to a document: when it was read, what bounded the read,
// how many candidates it added, every count behind an absence, and the pull requests a result
// names. A pull request no result names is counted as unmatched in the read population; its state,
// reviews and checks are not listed because the document starts from sessions.
type Changes struct {
	Source     string    `json:"source"`
	ObservedAt time.Time `json:"observedAt"`
	ReadBackTo time.Time `json:"readBackTo"`
	// StateLayer is the measurement layer of the states in Linked: an outcome of each pull request,
	// never of a session linked to it.
	StateLayer            layer.Layer `json:"stateLayer"`
	PullRequestsRead      int         `json:"pullRequestsRead"`
	PullRequestsNamed     int         `json:"pullRequestsNamed"`
	PullRequestsUnmatched int         `json:"pullRequestsUnmatched"`
	RepositoryTotal       *int64      `json:"repositoryTotal"`
	HistoryRead           int         `json:"historyRead"`
	HistoryBackTo         time.Time   `json:"historyBackTo,omitzero"`
	CommitListsCut        int         `json:"commitListsCut"`
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
	Rates                 []PRRate            `json:"rates"`
}

// LinkedPullRequest is a pull request's state as the forge reported it at Changes.ObservedAt, and
// whether a candidate names it or only an alternative does.
type LinkedPullRequest struct {
	Number                    int64               `json:"number"`
	State                     string              `json:"state"`
	MergedAt                  time.Time           `json:"mergedAt,omitzero"`
	NamedBy                   string              `json:"namedBy"`
	Reviews                   *ObservedStates     `json:"reviews"`
	HeadChecks                *ObservedStates     `json:"headChecks"`
	ReviewObservations        []ReviewObservation `json:"reviewObservations"`
	RequestedChangesRevisions *DerivedCount       `json:"requestedChangesRevisions"`
	ReviewRounds              *DerivedCount       `json:"reviewRounds"`
	MergeMethod               *ObservedMethod     `json:"mergeMethod"`
	HistoricalChecks          *PRCheckHistory     `json:"historicalChecks"`
}

// DerivedCount distinguishes a known zero from an unsupported count. Confidence applies only
// to the declared observation population, never to a session's contribution.
type DerivedCount struct {
	Layer      layer.Layer `json:"layer"`
	Value      *int        `json:"value"`
	Provenance string      `json:"provenance"`
	Confidence string      `json:"confidence"`
	Reason     string      `json:"reason,omitempty"`
}

// ObservedMethod identifies only merge topology that the source parent count establishes.
type ObservedMethod struct {
	Layer      layer.Layer  `json:"layer"`
	Value      *string      `json:"value"`
	Source     event.Source `json:"source"`
	Provenance string       `json:"provenance"`
	Confidence string       `json:"confidence"`
	Reason     string       `json:"reason,omitempty"`
}

// ObservedStates reports a bounded forge connection. Total and Listed show how much of that
// connection the read reached; State is the head check rollup, absent for reviews.
type ObservedStates struct {
	Layer  layer.Layer  `json:"layer"`
	Total  int64        `json:"total"`
	Listed int64        `json:"listed"`
	State  string       `json:"state,omitempty"`
	States []StateCount `json:"states"`
}

// StateCount is the number of observations in one closed forge state.
type StateCount struct {
	State string `json:"state"`
	Count int    `json:"count"`
}

const (
	namedByCandidate   = "candidate"
	namedByAlternative = "alternative"
)

// ChangesOf summarizes what prs added to results.
func ChangesOf(prs *PullRequests, results []Result, source string) *Changes {
	c := &Changes{
		Source: source, ObservedAt: prs.ObservedAt, ReadBackTo: prs.ReadBackTo, StateLayer: layer.Outcome,
		RepositoryTotal: prs.RepositoryTotal, HistoryRead: prs.HistoryRead, HistoryBackTo: prs.HistoryBackTo,
		PullRequestsRead: len(prs.Requests), ListedMadeHere: prs.MadeHere, ListedNotInReflog: len(prs.NotInReflog),
		ListedNotLocal: len(prs.NotLocal), ListedUnreadable: len(prs.Unreadable), ListedBeforeWindow: len(prs.BeforeWindow),
		Linked: []LinkedPullRequest{},
	}
	named := namedBy(results)
	delivery := deliveryByPR(prs)
	reviews := reviewsByPR(prs)
	history := historyByPR(prs)
	for i := range prs.Requests {
		pr, ok := prs.Requests[i].Payload.(event.PullRequest)
		if !ok {
			continue
		}
		if pr.Listed < pr.Commits {
			c.CommitListsCut++
		}
		if by, ok := named[pr.Number]; ok {
			c.PullRequestsNamed++
			linked := LinkedPullRequest{Number: pr.Number, State: pr.State, MergedAt: pr.MergedAt, NamedBy: by}
			if states := delivery[pr.Number]; states != nil {
				linked.Reviews, linked.HeadChecks = states.reviews, states.checks
			}
			if pr.ReviewsAvailable {
				linked.ReviewObservations = reviews[pr.Number]
			}
			linked.RequestedChangesRevisions = requestedChangesRevisions(&pr, linked.ReviewObservations)
			linked.ReviewRounds = withheldCount("review-states-do-not-define-round-boundaries")
			linked.MergeMethod = mergeMethod(&pr, prs.Requests[i].Source)
			linked.HistoricalChecks = history[pr.Number]
			c.Linked = append(c.Linked, linked)
		}
	}
	c.PullRequestsUnmatched = c.PullRequestsRead - c.PullRequestsNamed
	slices.SortFunc(c.Linked, func(a, b LinkedPullRequest) int { return int(a.Number - b.Number) })
	c.Rates = deliveryRates(c.Linked)
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
