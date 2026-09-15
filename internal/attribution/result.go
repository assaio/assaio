package attribution

import (
	"time"

	"github.com/assaio/assaio/internal/event"
)

const (
	// Algorithm versions the method set carried by every attribution document.
	Algorithm = "session-commit/v1"

	statusMatched   = "matched"
	statusAmbiguous = "ambiguous"
	statusUnmatched = "unmatched"

	methodConfirmed = "manual-confirmation"
	methodOverlap   = "project-time-overlap"
	methodFollowing = "project-time-following"
	methodNone      = "none"

	confidenceHigh         = "high"
	confidenceMedium       = "medium"
	confidenceLow          = "low"
	confidenceInsufficient = "insufficient"

	relationOverlap   = "overlaps-session"
	relationFollowing = "follows-session"
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
	CommitID     string               `json:"commitId"`
	OccurredAt   time.Time            `json:"occurredAt"`
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

// Document is the stable public output of one local attribution pass.
type Document struct {
	Algorithm      string    `json:"algorithm"`
	Project        string    `json:"project"`
	Since          time.Time `json:"since"`
	ObservedAt     time.Time `json:"observedAt"`
	MaxGapSeconds  int64     `json:"maxGapSeconds"`
	WindowSessions int       `json:"windowSessions"`
	OtherProjects  int       `json:"otherProjectSessions"`
	ProjectUnknown int       `json:"projectUnknownSessions"`
	SkippedCommits int       `json:"skippedCommits"`
	Summary        Summary   `json:"summary"`
	Results        []Result  `json:"results"`
}
