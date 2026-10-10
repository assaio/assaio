package github

import (
	"errors"
	"fmt"
	"time"

	"github.com/assaio/assaio/internal/event"
)

// Validate the independent read before the join, including PRs the base walk did not retain.
func validateHistory(w *walk, project string, observedAt time.Time) error {
	for id := range w.nodes {
		nd := w.nodes[id]
		if !nodeID(nd.ID) || nd.Number < 1 {
			return errors.New("history has an unusable pull request identity")
		}
		if nd.History == nil || nd.History.TotalCount == nil || nd.History.Nodes == nil {
			continue
		}
		if err := validatePopulation(*nd.History.TotalCount, len(nd.History.Nodes), 100); err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, c := range nd.History.Nodes {
			if c == nil || c.Commit == nil || seen[c.Commit.OID] {
				return errors.New("history has an unavailable or duplicate listed commit")
			}
			seen[c.Commit.OID] = true
			e := envelope(event.TypePullRequestCommit, nd.ID+":"+c.Commit.OID, observedAt, event.TimeIngestTime,
				event.GrainCommit, event.Source{Name: sourceName}, project, observedAt)
			e.Payload = event.PullRequestCommit{Number: nd.Number, Commit: c.Commit.OID, Suites: suitesOf(&nd, c.Commit.OID)}
			if err := e.Validate(); err != nil {
				return err
			}
			conn := c.Commit.CheckSuites
			if conn == nil || conn.TotalCount == nil || conn.Nodes == nil {
				continue
			}
			if err := validatePopulation(*conn.TotalCount, len(conn.Nodes), 10); err != nil {
				return err
			}
			for _, suite := range conn.Nodes {
				if suite != nil && suite.CheckRuns != nil && suite.CheckRuns.TotalCount != nil && suite.CheckRuns.Nodes != nil {
					if err := validatePopulation(*suite.CheckRuns.TotalCount, len(suite.CheckRuns.Nodes), 5); err != nil {
						return err
					}
				}
			}
		}
		suites, runs, err := historyObservations(&nd, event.Source{Name: sourceName}, project, observedAt)
		if err != nil {
			return err
		}
		for _, events := range [][]event.Event{suites, runs} {
			for i := range events {
				if err := events[i].Validate(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validatePopulation(total int64, listed, bound int) error {
	if total < 0 || int64(listed) > total || listed > bound {
		return fmt.Errorf("invalid connection population %d/%d (bound %d)", listed, total, bound)
	}
	return nil
}
