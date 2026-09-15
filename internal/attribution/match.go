package attribution

import (
	"sort"
	"time"

	"github.com/assaio/assaio/internal/event"
)

// DefaultMaxGap keeps the delayed-commit conformance case without pretending that
// temporal proximity remains meaningful without a bound.
const DefaultMaxGap = 48 * time.Hour

// Match derives session-to-commit results without storing either observations or edges.
func Match(sessions []Session, commits []event.Event, confirmed map[string]string, observedAt time.Time) []Result {
	orderedSessions := append([]Session(nil), sessions...)
	sort.Slice(orderedSessions, func(i, j int) bool {
		if orderedSessions[i].ID != orderedSessions[j].ID {
			return orderedSessions[i].ID < orderedSessions[j].ID
		}
		return orderedSessions[i].StartedAt.Before(orderedSessions[j].StartedAt)
	})
	orderedCommits := commitObservations(commits)
	results := make([]Result, 0, len(orderedSessions))
	for i := range orderedSessions {
		results = append(results, matchSession(&orderedSessions[i], orderedCommits, confirmed, observedAt))
	}
	return results
}

func matchSession(session *Session, commits []event.Event, confirmed map[string]string, observedAt time.Time) Result {
	result := emptyResult(session)
	if session.ID == "" || session.Project == "" {
		result.Reason = "project-or-session-unavailable"
		return result
	}
	if session.StartedAt.IsZero() || session.EndedAt.Before(session.StartedAt) {
		result.Reason = "invalid-session-window"
		return result
	}

	plausible := candidatesFor(session, commits)
	if commitID := confirmed[session.ID]; commitID != "" {
		applyConfirmation(&result, commits, plausible, commitID)
		return result
	}
	overlaps, following := splitCandidates(plausible)
	switch {
	case len(overlaps) > 0:
		result.Status = statusMatched
		result.Method = methodOverlap
		result.Confidence = confidenceMedium
		result.Candidates = overlaps
		result.Alternatives = following
	case len(following) == 1:
		result.Status = statusMatched
		result.Method = methodFollowing
		result.Confidence = confidenceLow
		result.Candidates = following
	case len(following) > 1:
		result.Status = statusAmbiguous
		result.Method = methodFollowing
		result.Ambiguous = true
		result.Reason = "competing-following-commit-candidates"
		result.Candidates = following
	default:
		if observedAt.Before(session.EndedAt.Add(DefaultMaxGap)) {
			result.Reason = "no-candidate-window-still-open"
		} else {
			result.Reason = "no-commit-candidate"
		}
	}
	return result
}

func emptyResult(session *Session) Result {
	return Result{
		Session: *session, Status: statusUnmatched, Method: methodNone,
		Confidence: confidenceInsufficient, Provenance: event.Derived,
		Candidates: []Candidate{}, Alternatives: []Candidate{},
	}
}

func applyConfirmation(result *Result, commits []event.Event, plausible []Candidate, commitID string) {
	for i := range commits {
		if commits[i].ID != commitID {
			continue
		}
		winner := candidateFrom(&commits[i], &result.Session)
		result.Status = statusMatched
		result.Method = methodConfirmed
		result.Confidence = confidenceHigh
		result.Provenance = event.Manual
		result.Candidates = []Candidate{winner}
		for i := range plausible {
			candidate := &plausible[i]
			if candidate.CommitID != commitID {
				result.Alternatives = append(result.Alternatives, *candidate)
			}
		}
		return
	}
	result.Reason = "confirmed-commit-unobserved"
}

func commitObservations(events []event.Event) []event.Event {
	out := make([]event.Event, 0, len(events))
	for i := range events {
		if events[i].Type == event.TypeCommit {
			if _, ok := commitPayload(events[i].Payload); ok {
				out = append(out, events[i])
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OccurredAt.Equal(out[j].OccurredAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].OccurredAt.Before(out[j].OccurredAt)
	})
	return out
}

func candidatesFor(session *Session, commits []event.Event) []Candidate {
	end := session.EndedAt.Add(DefaultMaxGap)
	out := make([]Candidate, 0)
	for i := range commits {
		commit := &commits[i]
		if commit.Subject.Project != session.Project || commit.OccurredAt.Before(session.StartedAt) || commit.OccurredAt.After(end) {
			continue
		}
		out = append(out, candidateFrom(commit, session))
	}
	return out
}

func candidateFrom(commit *event.Event, session *Session) Candidate {
	payload, _ := commitPayload(commit.Payload)
	relation, gap := relationOverlap, int64(0)
	if commit.OccurredAt.After(session.EndedAt) {
		relation = relationFollowing
		gap = int64(commit.OccurredAt.Sub(session.EndedAt).Seconds())
	}
	return Candidate{
		CommitID: commit.ID, OccurredAt: commit.OccurredAt, Relation: relation, GapSeconds: gap,
		Source: commit.Source, TimeSource: commit.TimeSource, Privacy: commit.Privacy,
		Provenance: commit.Provenance, FilesChanged: payload.FilesChanged, Files: payload.Files,
	}
}

func commitPayload(payload event.Payload) (event.Commit, bool) {
	switch commit := payload.(type) {
	case event.Commit:
		return commit, true
	case *event.Commit:
		return *commit, true
	default:
		return event.Commit{}, false
	}
}

func splitCandidates(candidates []Candidate) (overlaps, following []Candidate) {
	for i := range candidates {
		candidate := &candidates[i]
		if candidate.Relation == relationOverlap {
			overlaps = append(overlaps, *candidate)
		} else {
			following = append(following, *candidate)
		}
	}
	return nonNil(overlaps), nonNil(following)
}

func nonNil(candidates []Candidate) []Candidate {
	if candidates == nil {
		return []Candidate{}
	}
	return candidates
}
