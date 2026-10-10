package attribution

import (
	"slices"
	"time"

	"github.com/assaio/assaio/internal/event"
)

// How a pull request holds a commit.
const (
	viaListed = "listed"
	viaMerged = "merged-as"
)

// ReasonListedUnreadable is why a session with no candidate is not "nothing reached the branch": a
// pull request updated after it started lists a commit this clone cannot read.
const ReasonListedUnreadable = "listed-commit-not-readable-here"

// PullRequests is what a forge reported and what the collector could not make a candidate of:
// everything session-commit/v3 reads beyond the commits.
type PullRequests struct {
	// Requests are scm.pull_request.observed events, Listings scm.pull_request.commit.observed.
	Requests         []event.Event
	Listings         []event.Event
	Reviews          []event.Event
	Checks           []event.Event
	Suites           []event.Event
	HistoricalChecks []event.Event
	RepositoryTotal  *int64
	HistoryRead      int
	HistoryBackTo    time.Time
	// Lines maps a merge commit to the first-parent line from it, as far as its pull request's
	// commit count reaches: where a rebase merge laid the commits it replayed.
	Lines map[string][]string
	// MadeHere is how many listed commits a HEAD reflog records as made here became observations:
	// the candidates R1 adds, which session-commit/v2 never sees.
	MadeHere int
	// NotInReflog were present but no HEAD reflog records making them here; NotLocal were absent;
	// Unreadable were recorded but could not be read; BeforeWindow were made here before the window.
	NotInReflog, NotLocal, Unreadable, BeforeWindow []string
	ObservedAt                                      time.Time
	// ReadBackTo is the update time the read reached: a pull request last updated earlier is not in
	// Requests.
	ReadBackTo time.Time
}

// LinkChanges applies session-commit/v3's rules to results Match computed over commits that already
// include the listed ones this clone made (R1): it links each candidate and alternative to its pull
// requests (R2), names the one pull request every candidate shares (R3), and gives a session left
// with no candidate the reason a listed commit it cannot read supplies (R4). It changes no status,
// method, confidence or ambiguity.
func LinkChanges(results []Result, commits []event.Event, prs *PullRequests) {
	idx := prs.index(commits)
	for i := range results {
		r := &results[i]
		for j := range r.Candidates {
			r.Candidates[j].Changes = idx.linksOf(r.Candidates[j].CommitID)
		}
		for j := range r.Alternatives {
			r.Alternatives[j].Changes = idx.linksOf(r.Alternatives[j].CommitID)
		}
		r.Change = oneChange(r.Candidates)
		if r.Reason == reasonNoCandidate && idx.unreadableSince(r.Session.StartedAt) {
			r.Reason = ReasonListedUnreadable
		}
	}
}

type changeIndex struct {
	links map[string][]ChangeLink
	// unreadable are the update times of the pull requests listing a commit this clone cannot read.
	unreadable []time.Time
}

func (p *PullRequests) index(commits []event.Event) changeIndex {
	byID := make(map[string]*event.Event, len(commits))
	for i := range commits {
		byID[commits[i].ID] = &commits[i]
	}
	idx := changeIndex{links: map[string][]ChangeLink{}}
	updated := map[int64]time.Time{}
	merges := map[string]bool{}
	for i := range p.Requests {
		if pr, ok := p.Requests[i].Payload.(event.PullRequest); ok && pr.MergeCommit != "" {
			merges[pr.MergeCommit] = true
		}
	}
	for i := range p.Requests {
		pr, ok := p.Requests[i].Payload.(event.PullRequest)
		if !ok {
			continue
		}
		updated[pr.Number] = p.Requests[i].OccurredAt
		link := linkFrom(pr.Number, viaMerged, &p.Requests[i])
		for _, hash := range p.mergeWrote(&pr, byID, merges) {
			idx.add(hash, &link)
		}
	}
	cannotRead := map[string]bool{}
	for _, hash := range append(slices.Clone(p.NotLocal), p.Unreadable...) {
		cannotRead[hash] = true
	}
	for i := range p.Listings {
		listed, ok := p.Listings[i].Payload.(event.PullRequestCommit)
		if !ok {
			continue
		}
		link := linkFrom(listed.Number, viaListed, &p.Listings[i])
		idx.add(listed.Commit, &link)
		if cannotRead[listed.Commit] {
			idx.unreadable = append(idx.unreadable, updated[listed.Number])
		}
	}
	for hash := range idx.links {
		slices.SortFunc(idx.links[hash], func(a, b ChangeLink) int {
			if a.Number != b.Number {
				return int(a.Number - b.Number)
			}
			return compareStrings(a.Via, b.Via)
		})
	}
	return idx
}

func (idx *changeIndex) add(hash string, link *ChangeLink) {
	for i := range idx.links[hash] {
		if idx.links[hash][i].Number == link.Number && idx.links[hash][i].Via == link.Via {
			return
		}
	}
	idx.links[hash] = append(idx.links[hash], *link)
}

// linksOf is never nil once pull requests were read: an empty list says "in no commit list read",
// which a missing field -- the offline document's -- does not.
func (idx *changeIndex) linksOf(hash string) []ChangeLink {
	if links := idx.links[hash]; links != nil {
		return links
	}
	return []ChangeLink{}
}

func (idx *changeIndex) unreadableSince(start time.Time) bool {
	return slices.ContainsFunc(idx.unreadable, func(updated time.Time) bool { return !updated.Before(start) })
}

// mergeWrote is what a pull request's merge put on the branch: its merge commit and, after a rebase
// merge, each commit before it on the first-parent line that the forge replayed too. A squash or a
// merge commit stops the walk at itself, and so does another pull request's merge commit: a replay
// can write fewer commits than the pull request counts, and the line runs on into the one before.
func (p *PullRequests) mergeWrote(pr *event.PullRequest, byID map[string]*event.Event, merges map[string]bool) []string {
	if pr.MergeCommit == "" {
		return nil
	}
	out := []string{pr.MergeCommit}
	for k, hash := range p.Lines[pr.MergeCommit] {
		commit, ok := byID[hash]
		if !ok || !evidenceOf(commit).rebased || k > 0 && merges[hash] {
			break
		}
		if k > 0 {
			out = append(out, hash)
		}
	}
	return out
}

func linkFrom(number int64, via string, e *event.Event) ChangeLink {
	return ChangeLink{
		Number: number, Via: via, Source: e.Source,
		TimeSource: e.TimeSource, Provenance: e.Provenance, Privacy: e.Privacy,
	}
}

// oneChange is the pull request every candidate belongs to, when every candidate belongs to one and
// they share exactly one. A stack or a reopened pull request lists one commit twice, and then no
// single pull request is the answer.
func oneChange(candidates []Candidate) int64 {
	var shared []int64
	for i := range candidates {
		numbers := changeNumbers(candidates[i].Changes)
		if len(numbers) == 0 {
			return 0
		}
		if i == 0 {
			shared = numbers
			continue
		}
		shared = slices.DeleteFunc(shared, func(n int64) bool { return !slices.Contains(numbers, n) })
	}
	if len(shared) != 1 {
		return 0
	}
	return shared[0]
}

func changeNumbers(links []ChangeLink) []int64 {
	var out []int64
	for _, l := range links {
		if !slices.Contains(out, l.Number) {
			out = append(out, l.Number)
		}
	}
	return out
}

func compareStrings(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
