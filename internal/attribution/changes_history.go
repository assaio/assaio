package attribution

import (
	"slices"
	"time"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/layer"
)

// ForgeObservation retains the source envelope without a person subject or open payload.
type ForgeObservation struct {
	ID         string       `json:"id"`
	Source     event.Source `json:"source"`
	OccurredAt time.Time    `json:"occurredAt"`
	ObservedAt time.Time    `json:"observedAt"`
	TimeSource string       `json:"timeSource"`
	Privacy    string       `json:"privacy"`
	Provenance string       `json:"provenance"`
}

// ReviewObservation carries a source review snapshot, not a history of its state transitions.
type ReviewObservation struct {
	ForgeObservation
	Payload event.Review `json:"payload"`
}

// SuiteObservation describes checks on a listed commit, without proving a PR pipeline.
type SuiteObservation struct {
	ForgeObservation
	Payload event.CheckSuite `json:"payload"`
}

// CheckObservation retains one run's source times rather than substituting an occurrence time.
type CheckObservation struct {
	ForgeObservation
	Payload event.Check `json:"payload"`
}

// PRCheckHistory declares the independent read and its currently listed commit population.
// Unavailable populations stay nil; an empty source connection remains a present zero.
type PRCheckHistory struct {
	ReadState string               `json:"readState"`
	Layer     layer.Layer          `json:"layer"`
	Commits   *event.Population    `json:"commits"`
	ByCommit  []CommitCheckHistory `json:"byCommit"`
}

// CommitCheckHistory preserves coverage at every bounded connection, including unavailable runs.
type CommitCheckHistory struct {
	Commit            string             `json:"commit"`
	Suites            *event.Population  `json:"suites"`
	SuiteObservations []SuiteObservation `json:"suiteObservations"`
	CheckObservations []CheckObservation `json:"checkObservations"`
}

func forgeObservation(e *event.Event) ForgeObservation {
	return ForgeObservation{
		ID: e.ID, Source: e.Source, OccurredAt: e.OccurredAt, ObservedAt: e.ObservedAt,
		TimeSource: e.TimeSource, Privacy: e.Privacy, Provenance: e.Provenance,
	}
}

func reviewsByPR(prs *PullRequests) map[int64][]ReviewObservation {
	out := make(map[int64][]ReviewObservation, len(prs.Requests))
	for i := range prs.Requests {
		if pr, ok := prs.Requests[i].Payload.(event.PullRequest); ok && pr.ReviewsAvailable {
			out[pr.Number] = []ReviewObservation{}
		}
	}
	for i := range prs.Reviews {
		if r, ok := prs.Reviews[i].Payload.(event.Review); ok {
			if _, available := out[r.Number]; available {
				out[r.Number] = append(out[r.Number], ReviewObservation{forgeObservation(&prs.Reviews[i]), r})
			}
		}
	}
	for number := range out {
		slices.SortFunc(out[number], func(a, b ReviewObservation) int { return compareStrings(a.ID, b.ID) })
	}
	return out
}

func historyByPR(prs *PullRequests) map[int64]*PRCheckHistory {
	out := make(map[int64]*PRCheckHistory, len(prs.Requests))
	for i := range prs.Requests {
		pr, ok := prs.Requests[i].Payload.(event.PullRequest)
		if !ok {
			continue
		}
		state := pr.HistoryState
		if state == "" {
			state = "unavailable"
		}
		h := &PRCheckHistory{ReadState: state, Layer: layer.Outcome}
		if state == "read" {
			h.Commits = &event.Population{Total: pr.Commits, Listed: pr.Listed}
			h.ByCommit = commitHistories(prs, pr.Number)
		}
		out[pr.Number] = h
	}
	return out
}

func commitHistories(prs *PullRequests, number int64) []CommitCheckHistory {
	out := []CommitCheckHistory{}
	for i := range prs.Listings {
		c, ok := prs.Listings[i].Payload.(event.PullRequestCommit)
		if !ok || c.Number != number {
			continue
		}
		h := CommitCheckHistory{Commit: c.Commit, Suites: c.Suites}
		if c.Suites != nil {
			h.SuiteObservations, h.CheckObservations = commitChecks(prs, number, c.Commit)
			if c.Suites.Total == 0 {
				h.CheckObservations = []CheckObservation{}
			}
		}
		out = append(out, h)
	}
	slices.SortFunc(out, func(a, b CommitCheckHistory) int { return compareStrings(a.Commit, b.Commit) })
	return out
}

func commitChecks(prs *PullRequests, number int64, commit string) ([]SuiteObservation, []CheckObservation) {
	suites := []SuiteObservation{}
	var checks []CheckObservation
	for i := range prs.Suites {
		if s, ok := prs.Suites[i].Payload.(event.CheckSuite); ok && s.Number == number && s.Commit == commit {
			suites = append(suites, SuiteObservation{forgeObservation(&prs.Suites[i]), s})
			if s.Runs != nil && checks == nil {
				checks = []CheckObservation{}
			}
		}
	}
	for i := range prs.HistoricalChecks {
		if c, ok := prs.HistoricalChecks[i].Payload.(event.Check); ok && c.Number == number && c.Commit == commit {
			checks = append(checks, CheckObservation{forgeObservation(&prs.HistoricalChecks[i]), c})
		}
	}
	slices.SortFunc(suites, func(a, b SuiteObservation) int { return compareStrings(a.ID, b.ID) })
	slices.SortFunc(checks, func(a, b CheckObservation) int { return compareStrings(a.ID, b.ID) })
	return suites, checks
}
