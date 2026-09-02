package trace

import (
	"testing"
	"time"

	"github.com/assaio/assaio/internal/store"
)

// seqOf builds a sequence belonging to one team member, which the store's own key pairs with the
// session id.
func seqOf(member, session string, n int) store.Timeline {
	t := seq(session, "", entrypointCLI, origin, n)
	t.Member = member
	return t
}

// The invariant every narrowed surface depends on: after ForSessions the sequences describe
// exactly the sessions the figures beside them do. A session id alone is not the key -- two
// members' installs can carry the same locally generated id, and blending them would put one
// person's sequences in another's panel.
func TestForSessionsKeepsExactlyTheSessionsNamed(t *testing.T) {
	set := New([]store.Timeline{
		seqOf("alice", "s1", 3),
		seqOf("bob", "s1", 5),
		seqOf("alice", "s2", 7),
	})

	got := set.ForSessions([]store.SessionRow{{SessionID: "s1", Member: "alice"}})
	if got.Sequences() != 1 || got.Steps() != 3 {
		t.Fatalf("narrowed to %d sequence(s) / %d step(s), want alice's 1 / 3",
			got.Sequences(), got.Steps())
	}
	if held := got.All()[0]; held.Member != "alice" {
		t.Errorf("kept the sequence of member %q, want alice -- the session id is not the key", held.Member)
	}
}

// A session the set holds no sequence for narrows to nothing rather than to everything: the
// empty answer is what a detector reports as "no sequence was stored", and the whole set would
// be a window-wide figure printed under a one-project heading.
func TestForSessionsNarrowsToNothingWhenNoneMatch(t *testing.T) {
	set := New([]store.Timeline{seq("s1", "", entrypointCLI, origin, 3)})

	got := set.ForSessions([]store.SessionRow{{SessionID: "other"}})
	if got.Sequences() != 0 || !got.Empty() {
		t.Fatalf("narrowed to %d sequence(s) / %d step(s), want none", got.Sequences(), got.Steps())
	}
	if none := set.ForSessions(nil); none.Sequences() != 0 {
		t.Errorf("ForSessions(nil) kept %d sequence(s), want none", none.Sequences())
	}
}

// The horizon says how far back the store's window reaches, so it travels with the narrowed set
// rather than being recomputed from the subset. Recomputing made the newest step of whichever
// project a drill selected into "now", and the same session then read as still running on one
// panel and as finished on another.
func TestForSessionsKeepsTheStoresHorizon(t *testing.T) {
	old := origin.AddDate(0, 0, -30)
	set := New([]store.Timeline{
		seq("ancient", "", entrypointCLI, old, 2),
		seq("recent", "", entrypointCLI, origin, 2),
	})

	got := set.ForSessions([]store.SessionRow{{SessionID: "ancient"}})
	if !got.Oldest.Equal(set.Oldest) || !got.Newest.Equal(set.Newest) {
		t.Errorf("narrowed horizon = %v..%v, want the set's %v..%v",
			got.Oldest, got.Newest, set.Oldest, set.Newest)
	}
	if want := origin.Add(time.Minute); !got.Newest.Equal(want) {
		t.Errorf("Newest = %v, want %v -- the subset's own last step is not the window's", got.Newest, want)
	}
}

// Applying it unconditionally has to be safe, which is what makes every surface able to apply it
// rather than only the filtered paths.
func TestForSessionsIsIdentityWhenEverySessionIsGiven(t *testing.T) {
	set := New([]store.Timeline{
		seq("s1", "", entrypointCLI, origin, 3),
		seq("s2", "agent-1", entrypointCLI, origin, 4),
	})

	got := set.ForSessions([]store.SessionRow{{SessionID: "s1"}, {SessionID: "s2"}})
	if got.Sequences() != set.Sequences() || got.Steps() != set.Steps() {
		t.Fatalf("identity narrowing gave %d/%d, want %d/%d",
			got.Sequences(), got.Steps(), set.Sequences(), set.Steps())
	}
	if !got.Oldest.Equal(set.Oldest) || !got.Newest.Equal(set.Newest) {
		t.Errorf("identity narrowing moved the horizon to %v..%v", got.Oldest, got.Newest)
	}
}

// A sub-agent's sequence carries its parent's session id, so narrowing to that session keeps the
// sub-agent's sequence too -- and the scope, not the narrowing, is what keeps the two apart.
func TestForSessionsKeepsASubAgentWithItsParentSession(t *testing.T) {
	set := New([]store.Timeline{
		seq("s1", "", entrypointCLI, origin, 3),
		seq("s1", "agent-1", entrypointCLI, origin, 4),
	})

	got := set.ForSessions([]store.SessionRow{{SessionID: "s1"}})
	if got.Sequences() != 2 {
		t.Fatalf("kept %d sequence(s), want the main loop and the sub-agent's", got.Sequences())
	}
	if v := got.Scope(Interactive); len(v.Sequences) != 1 || v.ExcludedSequences != 1 {
		t.Errorf("the interactive view of the narrowed set holds %d / excludes %d, want 1 / 1",
			len(v.Sequences), v.ExcludedSequences)
	}
}
