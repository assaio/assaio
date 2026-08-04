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
}

// Hash is the commit git created for tag, or "" when the scenario has no such commit.
func (f *Fixture) Hash(tag string) string { return f.Hashes[tag] }
