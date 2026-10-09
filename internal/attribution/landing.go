package attribution

import (
	"time"

	"github.com/assaio/assaio/internal/event"
)

// A commit's evidence is the time or times that say when its work happened. Git records two,
// and which of them means the work depends on who wrote the commit.
type evidence struct {
	// times are the moments a session must contain, or precede within the following window,
	// to be a candidate.
	times []time.Time
	// landed marks a commit the forge stamped when it merged the work: its time is an upper
	// bound on when the work happened, so it is never evidence of work inside a session.
	landed bool
	// excluded marks a commit that carries no work of its own.
	excluded bool
	// rebased marks a commit the forge replayed, judged at the time it was first written.
	rebased bool
}

// evidenceOf reads a commit observation:
//   - the forge's own merge commit is excluded: the branch commits it joins carry the work;
//   - a forge commit written and committed at the same moment -- a squash, a web edit, the
//     Revert button -- is landed, judged only as landing after the session;
//   - a forge commit written earlier -- a rebase merge -- is judged at its author time;
//   - any other commit at both times, since an amend, a local rebase or a cherry-pick keeps the
//     author time while the committer time marks the later rewrite, and either may be the
//     session's work. A local merge stays a candidate: git reports no lines for any merge, and
//     one made by hand can carry a conflict resolution.
func evidenceOf(commit *event.Event) evidence {
	payload, _ := commitPayload(commit.Payload)
	authored := payload.AuthoredAt
	if authored.IsZero() {
		authored = commit.OccurredAt
	}
	switch {
	case payload.CommittedByForge && payload.Parents > 1:
		return evidence{excluded: true}
	case payload.CommittedByForge && !authored.Before(commit.OccurredAt):
		return evidence{times: []time.Time{commit.OccurredAt}, landed: true}
	case payload.CommittedByForge:
		return evidence{times: []time.Time{authored}, rebased: true}
	case authored.Equal(commit.OccurredAt):
		return evidence{times: []time.Time{commit.OccurredAt}}
	default:
		return evidence{times: []time.Time{authored, commit.OccurredAt}}
	}
}

// relate places a commit's evidence against a session: one of its times inside the session, a
// landing inside or after it, or its earliest time in the following window. judged is the time
// the relation rests on. A landed commit is never inside a session: the session was running when
// the forge merged, which says nothing about who wrote the work.
func relate(ev evidence, session *Session) (relation string, judged time.Time, ok bool) {
	if !ev.landed {
		for _, t := range ev.times {
			if !t.Before(session.StartedAt) && !t.After(session.EndedAt) {
				return relationOverlap, t, true
			}
		}
	}
	for _, t := range ev.times {
		if t.Before(session.StartedAt) || t.After(session.EndedAt.Add(DefaultMaxGap)) {
			continue
		}
		if judged.IsZero() || t.Before(judged) {
			judged = t
		}
	}
	switch {
	case judged.IsZero():
		return "", time.Time{}, false
	case ev.landed:
		return relationLanded, judged, true
	default:
		return relationFollowing, judged, true
	}
}

// landsLater reports a forge-landed commit after the session's following window: the session's
// work may have shipped in it, and nothing but its time links the two.
func landsLater(ev evidence, session *Session) bool {
	return ev.landed && ev.times[0].After(session.EndedAt.Add(DefaultMaxGap))
}

// ForgeOf counts what forge detection found among commits and what it did to results.
func ForgeOf(commits []event.Event, results []Result) Forge {
	f := Forge{Detection: ForgeDetection}
	for i := range commits {
		switch ev := evidenceOf(&commits[i]); {
		case ev.excluded:
			f.MergeCommits++
		case ev.landed:
			f.LandedCommits++
		case ev.rebased:
			f.RebasedCommits++
		}
	}
	for i := range results {
		if results[i].Reason == ReasonLaterLanding {
			f.LaterLanding++
		}
	}
	return f
}

// ReasonLaterLanding is why a session with no candidate is not "nothing reached the branch": a
// forge landed a commit after its window, and the session's work may be in it.
const ReasonLaterLanding = "later-forge-landing"

func landedLater(session *Session, commits []event.Event) bool {
	for i := range commits {
		if commits[i].Subject.Project == session.Project && landsLater(evidenceOf(&commits[i]), session) {
			return true
		}
	}
	return false
}

// relationConfirmed places a confirmed commit against its session for display. A confirmation
// wins whatever the evidence says, including for a commit the rules exclude, so the relation is
// read from the committer time alone.
func relationConfirmed(commit *event.Event, session *Session) (string, time.Time) {
	if commit.OccurredAt.After(session.EndedAt) {
		return relationFollowing, commit.OccurredAt
	}
	return relationOverlap, commit.OccurredAt
}
