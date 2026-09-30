package github

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/assaio/assaio/internal/event"
)

const sourceName = "gh"

// Read is one read of a repository's pull requests: the repository gh resolved, an observation
// per pull request and per listed commit, and the update time the read reached.
type Read struct {
	Repository Repository
	Requests   []event.Event
	Listings   []event.Event
	BackTo     time.Time
}

// ReadPullRequests reads the pull requests of the repository gh resolves for root that were
// updated at or after since; observedAt stamps the observations. A pull request the contract
// rejects fails the read: nothing here comes from assaio's own parsing that a skip could excuse.
func ReadPullRequests(ctx context.Context, run Runner, root, project string, since, observedAt time.Time, build string) (Read, error) {
	repo, err := Resolve(ctx, run, root)
	if err != nil {
		return Read{}, err
	}
	w, err := read(ctx, run, root, repo, since)
	if err != nil {
		return Read{}, err
	}
	got := Read{Repository: repo, BackTo: w.backTo}
	ids := make([]string, 0, len(w.nodes))
	for id := range w.nodes {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	src := event.Source{Name: sourceName, Build: build}
	for _, id := range ids {
		nd := w.nodes[id]
		if !nodeID(nd.ID) {
			return Read{}, fmt.Errorf("pull request #%d from %s has no usable id", nd.Number, repo)
		}
		request, listings := observations(&nd, src, project, observedAt)
		all := append([]event.Event{request}, listings...)
		for i := range all {
			if err := all[i].Validate(); err != nil {
				return Read{}, fmt.Errorf("pull request #%d from %s: %w", nd.Number, repo, err)
			}
		}
		got.Requests = append(got.Requests, request)
		got.Listings = append(got.Listings, listings...)
	}
	return got, nil
}

func observations(nd *node, src event.Source, project string, observedAt time.Time) (event.Event, []event.Event) {
	pr := event.PullRequest{
		Number: nd.Number, State: strings.ToLower(nd.State), MergedAt: nd.MergedAt.UTC(),
		Commits: nd.Commits.TotalCount, Listed: int64(len(nd.Commits.Nodes)),
	}
	if nd.MergeCommit != nil {
		pr.MergeCommit = nd.MergeCommit.OID
	}
	request := envelope(event.TypePullRequest, nd.ID, nd.UpdatedAt.UTC(), event.TimeStated, event.GrainChange, src, project, observedAt)
	request.Payload = pr
	listings := make([]event.Event, 0, len(nd.Commits.Nodes))
	for _, c := range nd.Commits.Nodes {
		l := envelope(event.TypePullRequestCommit, nd.ID+":"+c.Commit.OID, observedAt, event.TimeIngestTime,
			event.GrainCommit, src, project, observedAt)
		l.Payload = event.PullRequestCommit{Number: nd.Number, Commit: c.Commit.OID}
		listings = append(listings, l)
	}
	return request, listings
}

func envelope(typ, id string, at time.Time, timeSource, grain string, src event.Source, project string, observedAt time.Time) event.Event {
	return event.Event{
		SpecVersion: event.SpecVersion, Type: typ, ID: id, Source: src,
		OccurredAt: at, ObservedAt: observedAt, TimeSource: timeSource, Grain: grain,
		// Pull requests and their commits identify work to anyone who can read the repository
		// (ADR 0020).
		Privacy: event.LocalOnly, Provenance: event.Parsed, Subject: event.Subject{Project: project},
	}
}

// nodeID accepts the forge's opaque node ids -- letters, digits, '_', '-' and '=' -- and nothing
// that could carry text into an observation's id.
func nodeID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' && r != '=' {
			return false
		}
	}
	return true
}
