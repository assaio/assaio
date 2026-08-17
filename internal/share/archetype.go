package share

import "fmt"

// archetypeCount is the closed size of the set. It is on the card ("06 of 12") because a
// set with a stated size is a collection, and a collection invites the next person to
// find theirs -- which an open-ended label never does.
const archetypeCount = 12

// kind is one archetype: a shape of working, never a rank. fires reads the same shares
// every other surface reads, and the first that matches in this order wins, so two people
// with the same window always get the same name.
type kind struct {
	name   string
	flavor func(f *facts) string
	fires  func(f *facts) bool
}

// kinds is ordered from most specific to least. The last entry has no condition, so the
// classification is total: there is no window that comes back without an archetype.
var kinds = []kind{
	{
		name: "The Delegator",
		flavor: func(f *facts) string {
			return f.delegation.text() + " of the work ran inside sub-agents. You set the brief and let it go."
		},
		fires: func(f *facts) bool { return f.delegation.ok && f.delegation.v >= 25 },
	},
	{
		name: "The Conductor",
		flavor: func(f *facts) string {
			return f.conversational.text() + " of sessions never touch a file. The decision lands before the code does."
		},
		fires: func(f *facts) bool { return f.conversational.ok && f.conversational.v >= 80 },
	},
	{
		name: "The Operator",
		flavor: func(f *facts) string {
			return f.commands.text() + " of the classified calls were commands. You drive the machine, not the editor."
		},
		fires: func(f *facts) bool { return f.commands.ok && f.commands.v >= 45 },
	},
	{
		name: "The Night Shift",
		flavor: func(f *facts) string {
			return fmt.Sprintf("%s of sessions started outside working hours, %s on weekends.", f.offHours.text(), f.weekend.text())
		},
		fires: func(f *facts) bool { return f.offHours.ok && f.offHours.v >= 60 },
	},
	{
		name: "The Refiner",
		flavor: func(f *facts) string {
			return f.rework.text() + " of written lines came back out again. You iterate until it fits."
		},
		fires: func(f *facts) bool { return f.rework.ok && f.rework.v >= 12 },
	},
	{
		name: "The Cartographer",
		flavor: func(f *facts) string {
			return f.conversational.text() + " of sessions end without an edit, and few of the calls are commands. You read before you build."
		},
		fires: func(f *facts) bool { return f.conversational.ok && f.conversational.v >= 60 && f.commands.or(100) < 30 },
	},
	{
		name: "The Sprinter",
		flavor: func(f *facts) string {
			return thousandsOf(f.sessions) + " sessions across the window, and a lot of them on every active day."
		},
		fires: func(f *facts) bool { return f.sessionsPerDay.ok && f.sessionsPerDay.v >= 40 },
	},
	{
		name: "The Economist",
		flavor: func(f *facts) string {
			return fmt.Sprintf("Only %s of tokens on premium models. You spend where it counts.", f.premium.text())
		},
		fires: func(f *facts) bool { return f.premium.ok && f.premium.v <= 55 },
	},
	{
		name:   "The Marathoner",
		flavor: func(f *facts) string { return "Longest focused session: " + f.focusedP95 + "." },
		fires:  func(f *facts) bool { return f.compaction.ok && f.compaction.v >= 10 },
	},
	{
		name: "The Archivist",
		flavor: func(f *facts) string {
			return f.cacheRead.text() + " of context served from cache. Almost nothing gets re-sent."
		},
		fires: func(f *facts) bool { return f.cacheRead.ok && f.cacheRead.v >= 98 },
	},
	{
		name: "The Newcomer",
		flavor: func(f *facts) string {
			return "The window is young. Run this again in a month and the shape will have a name."
		},
		fires: func(f *facts) bool { return f.sessions < 25 },
	},
	{
		name: "The Builder",
		flavor: func(_ *facts) string {
			return "No single dimension stands out. The mix is even across the board."
		},
		fires: func(_ *facts) bool { return true },
	},
}

// classify names the window's archetype. It always returns one: the final kind fires
// unconditionally, so no card renders a blank where an identity should be.
func classify(f *facts) Archetype {
	for i := range kinds {
		if kinds[i].fires(f) {
			return Archetype{Name: kinds[i].name, Flavor: kinds[i].flavor(f), Index: i + 1, Of: archetypeCount}
		}
	}
	last := len(kinds) - 1
	return Archetype{Name: kinds[last].name, Flavor: kinds[last].flavor(f), Index: len(kinds), Of: archetypeCount}
}
