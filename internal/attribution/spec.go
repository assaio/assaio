package attribution

import (
	"time"

	"github.com/assaio/assaio/internal/event"
	"github.com/assaio/assaio/internal/usage"
)

// commitSpec is one commit a scenario asks for: a tag the expectations refer to it by, the
// branch it lands on, the files it touches, and when it happens relative to the scenario's
// start. Times are relative so a fixture is reproducible rather than dated.
type commitSpec struct {
	Tag    string
	Branch string
	Files  []string
	At     time.Duration
	// Author is the git identity that made it, for the scenarios where two people are
	// working in the same window.
	Author string
	// Authored is the author time when it differs from At, the committer time: an amend, a
	// rebase or a cherry-pick keeps the time the commit was first written.
	Authored *time.Duration
	// Forge commits as the forge's own identity, the way GitHub.com writes the commits it makes.
	Forge bool
	// Lands brings work onto the main line the way a forge merges a pull request: "squash" writes
	// one commit with the tree of branch From, "merge" a two-parent commit joining it, "rebase"
	// replays the commit tagged From. Empty is an ordinary commit.
	Lands string
	From  string
	// Onto is the branch a landing writes to, when not the main line: GitHub's "Update branch"
	// merges the main line onto a pull request's branch.
	Onto string
	// Fetched makes the commit elsewhere and fetches it: it is written onto a remote-tracking ref
	// with plumbing, so no commit of it is ever made here -- a teammate's pushed branch.
	Fetched bool
	// CheckedOut then checks the fetched commit out, the way a reviewer looks at a pull request.
	CheckedOut bool
}

// changeSpec is one pull request the scenario's forge reports: the commits it lists and the commit
// its merge wrote, by tag. A listed tag no commit carries is a hash this clone never had. Updated
// is when the forge last changed it, relative to the scenario's start; a merged one defaults to its
// merge.
type changeSpec struct {
	Tag     string
	Number  int64
	Lists   []string
	Merge   string
	Updated time.Duration
}

// sessionSpec is one AI session that ran against the repository: who ran it, when it
// started and ended, and how many lines it reported adding.
type sessionSpec struct {
	ID     string
	Member string
	Start  time.Duration
	End    time.Duration
	Lines  int64
	// Parent is the session that spawned this one, for a sub-agent. Empty otherwise.
	Parent string
}

// Fixture is one scenario, built: a real repository on disk, the commit observations read
// back out of it, the usage records for its sessions, and the table mapping each scenario
// tag to the hash git actually produced.
type Fixture struct {
	Root     string
	Commits  []event.Event
	Sessions []usage.Record
	// Hashes maps a scenario's commit tag to the hash of the commit git created for it.
	Hashes map[string]string
	// Confirmed maps a session id to the commit hash a human confirmed for it, resolved
	// from the scenario's Corrections.
	Confirmed map[string]string
	// Changes is what the scenario's forge reports, nil when it has no pull request. Listed are the
	// listed commits this clone made, appended to Commits; Fetched the listed commits present here
	// but never made or checked out, which no honest engine sees as candidates.
	Changes *PullRequests
	Listed  []string
	Fetched []event.Event
}

// Hash is the commit git created for tag, or "" when the scenario has no such commit.
func (f *Fixture) Hash(tag string) string { return f.Hashes[tag] }
