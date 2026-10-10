package github

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/assaio/assaio/internal/event"
)

func attachHistory(ctx context.Context, run Runner, root string, repo Repository, since time.Time, base *walk) (walk, error) {
	history, err := readQuery(ctx, run, root, repo, since, historyQuery)
	if err != nil {
		return walk{}, fmt.Errorf("reading listed-commit check history: %w", err)
	}
	if err := validateHistory(&history, "p", since); err != nil {
		return walk{}, fmt.Errorf("invalid listed-commit check history: %w", err)
	}
	for id := range base.nodes {
		nd := base.nodes[id]
		nd.HistoryState = "outside-history-read"
		if h, ok := history.nodes[id]; ok {
			nd.HistoryState = "unavailable"
			if h.History != nil && h.History.TotalCount != nil && h.History.Nodes != nil {
				if sameCommitPopulation(&nd, &h) {
					nd.History, nd.HistoryState = h.History, "read"
				} else {
					nd.HistoryState = "commit-list-changed"
				}
			}
		}
		base.nodes[id] = nd
	}
	return history, nil
}

// Both bounded connections must name the same unique commits before their facts can be joined.
func sameCommitPopulation(base, history *node) bool {
	h := history.History
	if base.Number != history.Number || base.Commits.TotalCount == nil || *base.Commits.TotalCount != *h.TotalCount || len(base.Commits.Nodes) != len(h.Nodes) {
		return false
	}
	set := make(map[string]bool, len(base.Commits.Nodes))
	for _, c := range base.Commits.Nodes {
		if set[c.Commit.OID] {
			return false
		}
		set[c.Commit.OID] = true
	}
	for _, c := range h.Nodes {
		if c == nil || c.Commit == nil || !set[c.Commit.OID] {
			return false
		}
		delete(set, c.Commit.OID)
	}
	return len(set) == 0
}

func historyObservations(nd *node, src event.Source, project string, observedAt time.Time) ([]event.Event, []event.Event, error) {
	var suites, checks []event.Event
	if nd.History == nil {
		return suites, checks, nil
	}
	suiteIDs, runIDs := map[string]bool{}, map[string]bool{}
	for _, c := range nd.History.Nodes {
		if c == nil || c.Commit == nil {
			return nil, nil, errors.New("history has an unavailable commit node")
		}
		connection := c.Commit.CheckSuites
		if connection == nil || connection.TotalCount == nil || connection.Nodes == nil {
			continue
		}
		for _, s := range connection.Nodes {
			if s == nil || !nodeID(s.ID) || suiteIDs[s.ID] {
				return nil, nil, errors.New("history has an unavailable or duplicate suite id")
			}
			suiteIDs[s.ID] = true
			at := s.UpdatedAt
			if at.IsZero() {
				at = s.CreatedAt
			}
			at, timeSource := statedOrRead(at, observedAt)
			suite := event.CheckSuite{
				Number: nd.Number, Commit: c.Commit.OID, State: strings.ToLower(s.Status), Conclusion: strings.ToLower(s.Conclusion),
				CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
			}
			if s.CheckRuns != nil && s.CheckRuns.TotalCount != nil && s.CheckRuns.Nodes != nil {
				suite.Runs = &event.Population{Total: *s.CheckRuns.TotalCount, Listed: int64(len(s.CheckRuns.Nodes))}
				for _, r := range s.CheckRuns.Nodes {
					if r == nil || !nodeID(r.ID) || runIDs[r.ID] {
						return nil, nil, errors.New("history has an unavailable or duplicate check run id")
					}
					runIDs[r.ID] = true
					at := r.CompletedAt
					if at.IsZero() {
						at = r.StartedAt
					}
					at, timeSource := statedOrRead(at, observedAt)
					e := envelope(event.TypeCheck, nd.ID+":"+r.ID, at, timeSource, event.GrainCheck, src, project, observedAt)
					e.Payload = event.Check{
						Number: nd.Number, Kind: "run", State: strings.ToLower(r.Status), Conclusion: strings.ToLower(r.Conclusion),
						Commit: c.Commit.OID, Suite: s.ID, StartedAt: r.StartedAt, CompletedAt: r.CompletedAt,
					}
					checks = append(checks, e)
				}
			}
			e := envelope(event.TypeCheckSuite, nd.ID+":"+s.ID, at, timeSource, event.GrainSuite, src, project, observedAt)
			e.Payload = suite
			suites = append(suites, e)
		}
	}
	return suites, checks, nil
}

func suitesOf(nd *node, hash string) *event.Population {
	if nd.History == nil {
		return nil
	}
	for _, c := range nd.History.Nodes {
		if c != nil && c.Commit != nil && c.Commit.OID == hash && c.Commit.CheckSuites != nil && c.Commit.CheckSuites.TotalCount != nil && c.Commit.CheckSuites.Nodes != nil {
			return &event.Population{Total: *c.Commit.CheckSuites.TotalCount, Listed: int64(len(c.Commit.CheckSuites.Nodes))}
		}
	}
	return nil
}

func statedOrRead(at, observedAt time.Time) (time.Time, string) {
	if at.IsZero() {
		return observedAt, event.TimeIngestTime
	}
	return at, event.TimeStated
}
