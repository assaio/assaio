package report

import (
	"github.com/assaio/assaio/internal/pseudonym"
)

// MemberIdentity is how far an export may go in naming who produced a row. It exists
// because refusing `--by member` protects the *ranking* and not the data: a central store's
// default `--by day` rows carried the synced name into JSON and CSV, one pivot away from the
// leaderboard the refusal exists to prevent (B182).
type MemberIdentity int

const (
	// MemberPseudonymous replaces every member with this install's stable pseudonym. It is
	// the zero value on purpose: the export that nobody thought about is the safe one.
	MemberPseudonymous MemberIdentity = iota
	// MemberIdentified keeps the stored member value: a legacy name or a sync v2 digest.
	// Only an operator who asked for it gets this; the export labels the choice.
	MemberIdentified
)

// MemberDisclosure states which identity an export carries, or "" when no row names a
// member at all -- a purely local store has nothing to disclose and a note about members
// there would be noise.
func MemberDisclosure(rows []Row, id MemberIdentity) string {
	if !hasMember(rows) {
		return ""
	}
	if id == MemberIdentified {
		return "Member shows stored values: legacy names may remain; migrated sync v2 values are keyed digests."
	}
	return "Member values are additionally pseudonymized for this report; use --identify to show stored values, which may be legacy names or sync v2 digests."
}

// hasMember reports whether any row names a member, i.e. whether these rows came from a
// central store at all.
func hasMember(rows []Row) bool {
	for i := range rows {
		if rows[i].Member != "" {
			return true
		}
	}
	return false
}

// resolveMembers rewrites Member in place under id. The mapping is memoized per distinct
// name rather than per row: a year of one team's usage is many rows over few people, and
// the HMAC behind a pseudonym is the expensive half.
func resolveMembers(rows []Row, id MemberIdentity) {
	if id == MemberIdentified {
		return
	}
	seen := map[string]string{}
	for i := range rows {
		name := rows[i].Member
		if name == "" {
			continue
		}
		label, ok := seen[name]
		if !ok {
			label = pseudonym.For("member", name)
			seen[name] = label
		}
		rows[i].Member = label
	}
}
