package attribution

import (
	"time"

	"github.com/assaio/assaio/internal/event"
)

const (
	// Algorithm versions the method set carried by every attribution document.
	Algorithm = "session-commit/v2"

	statusMatched   = "matched"
	statusAmbiguous = "ambiguous"
	statusUnmatched = "unmatched"

	methodConfirmed = "manual-confirmation"
	methodOverlap   = "project-time-overlap"
	methodFollowing = "project-time-following"
	methodLanded    = "project-time-forge-landing"
	methodNone      = "none"

	confidenceHigh         = "high"
	confidenceMedium       = "medium"
	confidenceLow          = "low"
	confidenceInsufficient = "insufficient"

	relationOverlap   = "overlaps-session"
	relationFollowing = "follows-session"
	relationLanded    = "landed-by-forge"

	// ForgeDetection names what the collector recognises as a forge's own commit. A forge it
	// does not recognise merges as an ordinary committer, and its merges are judged as local.
	ForgeDetection = "github.com-committer"
)

// Session is the content-free part of a stored session that attribution consumes.
type Session struct {
	ID        string    `json:"id"`
	Project   string    `json:"project"`
	Tool      string    `json:"tool"`
	StartedAt time.Time `json:"startedAt"`
	EndedAt   time.Time `json:"endedAt"`
}

// Candidate is one commit observation and the temporal evidence relating it to a session.
type Candidate struct {
	CommitID   string    `json:"commitId"`
	OccurredAt time.Time `json:"occurredAt"`
	// EvidenceAt is the commit time the relation was judged from: its committer time, its
	// author time after a rewrite, or the moment a forge landed it.
	EvidenceAt   time.Time            `json:"evidenceAt"`
	Relation     string               `json:"relation"`
	GapSeconds   int64                `json:"gapSeconds"`
	Source       event.Source         `json:"source"`
	TimeSource   string               `json:"timeSource"`
	Privacy      string               `json:"privacy"`
	Provenance   string               `json:"provenance"`
	FilesChanged int64                `json:"filesChanged"`
	Files        event.FileCategories `json:"files"`
}

// Result keeps accepted or competing candidates distinct from alternatives a stronger
// method outranked. Ambiguous candidates are never reduced to a single winner.
type Result struct {
	Session      Session     `json:"session"`
	Status       string      `json:"status"`
	Method       string      `json:"method"`
	Confidence   string      `json:"confidence"`
	Ambiguous    bool        `json:"ambiguous"`
	Provenance   string      `json:"provenance"`
	Reason       string      `json:"reason,omitempty"`
	Candidates   []Candidate `json:"candidates"`
	Alternatives []Candidate `json:"alternatives"`
}

// Summary states both candidate coverage and resolved coverage over the same population.
type Summary struct {
	Population        int     `json:"population"`
	Matched           int     `json:"matched"`
	Ambiguous         int     `json:"ambiguous"`
	Unmatched         int     `json:"unmatched"`
	CandidateCovered  int     `json:"candidateCovered"`
	CandidateCoverage float64 `json:"candidateCoverage"`
	ResolvedCoverage  float64 `json:"resolvedCoverage"`
}

// Document is the stable public output of one local attribution pass. Project is "" when no
// stored usage resolved to the repository: no name the store shows belongs to it.
type Document struct {
	Algorithm      string    `json:"algorithm"`
	Project        string    `json:"project"`
	Since          time.Time `json:"since"`
	ObservedAt     time.Time `json:"observedAt"`
	MaxGapSeconds  int64     `json:"maxGapSeconds"`
	WindowSessions int       `json:"windowSessions"`
	OtherProjects  int       `json:"otherProjectSessions"`
	ProjectUnknown int       `json:"projectUnknownSessions"`
	// IdentityUnresolved counts sessions under this repository's name whose rows never resolved
	// to a repository, so they are neither matched nor counted as another project's.
	IdentityUnresolved int `json:"identityUnresolvedSessions"`
	SkippedCommits     int `json:"skippedCommits"`
	// Forge says which commits were recognised as a forge's own and what that did to the
	// population, so "no forge commit" is never read where detection could not have seen one.
	Forge   Forge    `json:"forge"`
	Summary Summary  `json:"summary"`
	Results []Result `json:"results"`
}

// Forge is what forge detection found in the window. Landed commits were stamped when the forge
// merged them; rebased ones were replayed by it and judged at the time they were written; merge
// commits were excluded as carrying no work; LaterLanding counts sessions left unmatched while a
// landed commit lies after their following window.
type Forge struct {
	Detection      string `json:"detection"`
	LandedCommits  int    `json:"landedCommits"`
	RebasedCommits int    `json:"rebasedCommits"`
	MergeCommits   int    `json:"mergeCommits"`
	LaterLanding   int    `json:"unmatchedBeforeLaterLanding"`
}

// Recognised reports whether detection found any commit a forge wrote.
func (f *Forge) Recognised() bool { return f.LandedCommits+f.RebasedCommits+f.MergeCommits > 0 }
