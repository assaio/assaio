package attribution

import (
	"context"
	"crypto/sha1" //nolint:gosec // names a commit this clone never had; no security property rests on it
	"encoding/hex"
	"fmt"
	"slices"
	"time"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/vcs"
)

// buildChanges turns the scenario's pull requests into the observations the connector would emit,
// and reads the listed commits through the collector `evidence --github` uses, so the reflog, the
// forge flag and the window decide what an engine sees exactly as they do for a user.
func (s *Scenario) buildChanges(ctx context.Context, dir string, f *Fixture) error {
	project := vcs.Project(dir)
	observed := epoch.Add(7 * day)
	prs := &PullRequests{Lines: map[string][]string{}, ObservedAt: observed, ReadBackTo: epoch.Add(-7 * day)}
	var listed []string
	seen := map[string]bool{}
	for i := range f.Commits {
		seen[f.Commits[i].ID] = true
	}
	for i := range s.Changes {
		spec := &s.Changes[i]
		hashes := make([]string, 0, len(spec.Lists))
		for _, tag := range spec.Lists {
			hash := f.Hash(tag)
			if hash == "" {
				hash = neverHad(tag)
			}
			hashes = append(hashes, hash)
			if !seen[hash] {
				seen[hash] = true
				listed = append(listed, hash)
			}
		}
		request, err := s.pullRequest(spec, f, int64(len(hashes)))
		if err != nil {
			return err
		}
		prs.Requests = append(prs.Requests, request)
		for _, hash := range hashes {
			l := listing(spec, hash, observed)
			if err := l.Validate(); err != nil {
				return err
			}
			prs.Listings = append(prs.Listings, l)
		}
		if merge := f.Hash(spec.Merge); merge != "" {
			line, err := vcs.FirstParents(ctx, dir, merge, len(hashes))
			if err != nil {
				return err
			}
			prs.Lines[merge] = line
		}
	}
	got, err := vcs.CollectListed(ctx, dir, project, listed, epoch.Add(-7*day), epoch, "corpus")
	if err != nil {
		return err
	}
	for i := range got.Events {
		f.Listed = append(f.Listed, got.Events[i].ID)
	}
	f.Commits = append(f.Commits, got.Events...)
	f.Fetched, _, _ = vcs.CollectHashes(ctx, dir, project, got.NotInReflog, epoch.Add(-7*day), epoch, "corpus")
	prs.MadeHere, prs.NotInReflog, prs.NotLocal = len(got.Events), got.NotInReflog, got.NotLocal
	prs.Unreadable, prs.BeforeWindow = got.Unreadable, got.BeforeWindow
	f.Changes = prs
	return nil
}

func (s *Scenario) pullRequest(spec *changeSpec, f *Fixture, listed int64) (event.Event, error) {
	pr := event.PullRequest{Number: spec.Number, State: event.ChangeOpen, Commits: listed, Listed: listed}
	updated := epoch.Add(spec.Updated)
	if merge := f.Hash(spec.Merge); merge != "" {
		i := slices.IndexFunc(f.Commits, func(e event.Event) bool { return e.ID == merge })
		if i < 0 {
			return event.Event{}, fmt.Errorf("pull request %s merges %s, which HEAD does not reach", spec.Tag, spec.Merge)
		}
		pr.State, pr.MergeCommit, pr.MergedAt = event.ChangeMerged, merge, f.Commits[i].OccurredAt
		if spec.Updated == 0 {
			updated = pr.MergedAt
		}
	}
	e := forgeEvent(event.TypePullRequest, "PR_"+spec.Tag, updated, event.TimeStated, event.GrainChange, pr)
	e.ObservedAt = epoch.Add(7 * day)
	err := e.Validate()
	return e, err
}

func listing(spec *changeSpec, hash string, observed time.Time) event.Event {
	return forgeEvent(event.TypePullRequestCommit, "PR_"+spec.Tag+":"+hash, observed, event.TimeIngestTime,
		event.GrainCommit, event.PullRequestCommit{Number: spec.Number, Commit: hash})
}

func forgeEvent(typ, id string, at time.Time, timeSource, grain string, payload event.Payload) event.Event {
	return event.Event{
		SpecVersion: event.SpecVersion, Type: typ, ID: id, Source: event.Source{Name: "gh", Build: "corpus"},
		OccurredAt: at, ObservedAt: at, TimeSource: timeSource, Grain: grain,
		Privacy: event.LocalOnly, Provenance: event.Parsed, Payload: payload,
	}
}

// neverHad is a well-formed hash for a listed commit this clone never received.
func neverHad(tag string) string {
	sum := sha1.Sum([]byte("never-had:" + tag)) //nolint:gosec // see the import
	return hex.EncodeToString(sum[:])
}
