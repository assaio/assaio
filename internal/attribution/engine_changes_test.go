package attribution

import (
	"slices"
	"testing"

	"github.com/assaio/assaio/internal/event"
)

// v3 is session-commit/v3 over whatever commits and pull requests a stand-in hands it, reduced to
// the corpus vocabulary; adjust edits its results before they are.
func v3(f *Fixture, commits []event.Event, prs *PullRequests, adjust func([]Result)) Links {
	results := Match(fixtureSessions(f), commits, f.Confirmed, epoch.Add(7*day))
	if prs != nil {
		LinkChanges(results, commits, prs)
	}
	if adjust != nil {
		adjust(results)
	}
	return linksFrom(results)
}

// headOnly is session-commit/v2: the commits on HEAD, no pull request read.
func headOnly(f *Fixture) Links {
	commits := slices.DeleteFunc(slices.Clone(f.Commits), func(e event.Event) bool { return slices.Contains(f.Listed, e.ID) })
	return v3(f, commits, nil, nil)
}

// presence trusts the object store: every listed commit this clone has is a candidate, fetched or not.
func presence(f *Fixture) Links {
	return v3(f, append(slices.Clone(f.Commits), f.Fetched...), f.Changes, nil)
}

// presentForgeBlind trusts every listed commit present here and reads none of them with the
// forge's flag: a merge GitHub wrote onto a branch someone pulled becomes work of their own.
func presentForgeBlind(f *Fixture) Links {
	fetched := slices.Clone(f.Fetched)
	for i := range fetched {
		if c, ok := fetched[i].Payload.(event.Commit); ok {
			c.CommittedByForge = false
			fetched[i].Payload = c
		}
	}
	return v3(f, append(slices.Clone(f.Commits), fetched...), f.Changes, nil)
}

// groupedAsOne reports a session whose candidates share one pull request as resolved -- the
// alternative ADR 0022 rejects, since one pull request can hold two people's commits.
func groupedAsOne(f *Fixture) Links {
	return v3(f, f.Commits, f.Changes, func(results []Result) {
		for i := range results {
			if results[i].Change != 0 {
				results[i].Ambiguous = false
			}
		}
	})
}

// namedMergeOnly links a rebase merge's last commit, the one the forge names, and no other.
func namedMergeOnly(f *Fixture) Links {
	prs := *f.Changes
	prs.Lines = nil
	return v3(f, f.Commits, &prs, nil)
}

// anyShared places a session in the first pull request any candidate belongs to.
func anyShared(f *Fixture) Links {
	return v3(f, f.Commits, f.Changes, func(results []Result) {
		for i := range results {
			for j := range results[i].Candidates {
				if links := results[i].Candidates[j].Changes; len(links) > 0 {
					results[i].Change = links[0].Number
					break
				}
			}
		}
	})
}

// linkedOnly asks only the candidates a pull request lists which pull request they share.
func linkedOnly(f *Fixture) Links {
	return v3(f, f.Commits, f.Changes, func(results []Result) {
		for i := range results {
			linked := slices.DeleteFunc(slices.Clone(results[i].Candidates), func(c Candidate) bool { return len(c.Changes) == 0 })
			results[i].Change = oneChange(linked)
		}
	})
}

// Each stand-in gets one thing about pull requests wrong. If one ever passes the scenario written
// against it, that scenario has stopped defending anything.
func TestEnginesThatMisreadPullRequestsFailTheChangeScenarios(t *testing.T) {
	for _, tt := range []struct {
		scenario, engine string
		run              func(*Fixture) Links
	}{
		{"branch-commit-behind-a-squash", "commits on HEAD only", headOnly},
		{"two-authors-one-change", "commits on HEAD only", headOnly},
		{"rebase-merge-two-commits", "commits on HEAD only", headOnly},
		{"teammate-branch-commit-during-session", "every commit present", presence},
		{"forge-update-branch-merge", "every present commit, no forge flag", presentForgeBlind},
		{"two-authors-one-change", "one pull request as one answer", groupedAsOne},
		{"teammate-pull-request-checked-out-for-review", "every commit present", presence},
		{"rebase-merge-two-commits", "the named merge commit only", namedMergeOnly},
		{"stacked-changes", "any shared pull request", anyShared},
		{"one-candidate-outside-every-change", "linked candidates only", linkedOnly},
	} {
		t.Run(tt.scenario+"/"+tt.engine, func(t *testing.T) {
			s, ok := Get(tt.scenario)
			if !ok {
				t.Fatalf("scenario %q is missing", tt.scenario)
			}
			f := buildOrSkip(t, &s)
			if len(Check(&s, &f, tt.run(&f))) == 0 {
				t.Fatalf("an engine using %s passed a scenario written to defeat it", tt.engine)
			}
		})
	}
}
