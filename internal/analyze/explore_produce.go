package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/store"
)

const (
	exploreName     = "explore-produce"
	exploreTitle    = "Explore vs Produce"
	exploreDescribe = "What the tool calls were for: reading and searching the codebase versus writing code in it."
	// exploreHowToRead is Result.HowToRead for this validator -- see its doc comment.
	exploreHowToRead = "Exploring is how an agent earns the right to write, so a high read share is not waste -- unfamiliar or large codebases legitimately need more of it. What this flags is the extreme: a window that reads and searches endlessly and almost never writes."
	// exploreWatchFloor is the produce share below which a window reads as searching far
	// more than it writes.
	exploreWatchFloor = 0.05
	// exploreMinCalls is the classified-call floor below which the split is too thin to
	// call: a handful of calls is not a working pattern.
	exploreMinCalls = 50
)

func init() { Register(exploreValidator{}) }

// exploreValidator reads the purpose split of a window's tool calls: how much of the work
// was looking around versus changing code.
type exploreValidator struct{}

func (exploreValidator) Name() string     { return exploreName }
func (exploreValidator) Title() string    { return exploreTitle }
func (exploreValidator) Describe() string { return exploreDescribe }

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (exploreValidator) Analyze(in Input) Result {
	r := Result{Name: exploreName, Title: exploreTitle, Describe: exploreDescribe, HowToRead: exploreHowToRead}
	if len(in.Usage) == 0 {
		r.Read = noDataRead
		r.Takeaway = "No usage in this window."
		return r
	}
	m := buildToolMix(in.Usage)
	r.restsOn(int(m.Classified), "classified tool calls")
	r.covering(m.Coverage())
	if m.Classified == 0 {
		r.Read = noDataRead
		r.Purity = 0.5
		r.Bars = []Bar{}
		r.Takeaway = "No tool in this window records what its tool calls were for."
		r.Caveats = append(r.Caveats, exploreCoverageCaveat(m))
		return r
	}
	sufficient := m.Classified >= exploreMinCalls
	producing := m.ProduceShare() >= exploreWatchFloor
	balanced := sufficient && producing

	r.Read = exploreRead(sufficient, balanced)
	r.Purity = explorePurity(m, sufficient)
	r.Figures = []Figure{
		{Label: "classified calls", Value: strconv.FormatInt(m.Classified, 10)},
		{Label: "produce share", Value: formatPercent(m.ProduceShare(), 0), Note: "writes, of classified calls"},
		{Label: "explore share", Value: formatPercent(m.ExploreShare(), 0), Note: "reads + searches"},
		{Label: "reads per write", Value: exploreRatio(m.Reads+m.Searches, m.Writes)},
	}
	r.Bars = toolMixBars(m)
	r.Takeaway = exploreTakeaway(sufficient, balanced)
	if m.Coverage() < 1 {
		r.Caveats = append(r.Caveats, exploreCoverageCaveat(m))
	}
	return r
}

// toolMix is the window's tool calls split by purpose, beside the total tool calls seen so
// coverage can be stated honestly.
type toolMix struct {
	Reads, Searches, Commands, Writes, Other int64
	// Classified is the sum of the five buckets: calls from tools that name their calls.
	Classified int64
	// AllCalls is every tool call in the window, including tools that record no names.
	AllCalls int64
}

// buildToolMix sums the per-purpose tool-call counts across the window's usage rows.
func buildToolMix(rows []store.UsageRow) toolMix {
	var m toolMix
	for i := range rows {
		m.Reads += rows[i].ToolReads
		m.Searches += rows[i].ToolSearches
		m.Commands += rows[i].ToolCommands
		m.Writes += rows[i].ToolWrites
		m.Other += rows[i].ToolOther
		m.AllCalls += rows[i].ToolCalls
	}
	m.Classified = m.Reads + m.Searches + m.Commands + m.Writes + m.Other
	return m
}

// classifiedCalls is one row's tool calls that carry a purpose. Zero means the row predates
// the capture or came from a tool that names no calls -- either way, nothing about what its
// calls did can be read from it.
func classifiedCalls(r *store.UsageRow) int64 {
	return r.ToolReads + r.ToolSearches + r.ToolCommands + r.ToolWrites + r.ToolOther
}

// ProduceShare is the share of classified calls that wrote code.
func (m toolMix) ProduceShare() float64 { return shareOf(m.Writes, m.Classified) }

// ExploreShare is the share of classified calls that read or searched.
func (m toolMix) ExploreShare() float64 { return shareOf(m.Reads+m.Searches, m.Classified) }

// Coverage is the share of the window's tool calls that carry a purpose, 1 when every call
// came from a tool that names its calls.
func (m toolMix) Coverage() float64 {
	if m.AllCalls == 0 {
		return 0
	}
	return shareOf(m.Classified, m.AllCalls)
}

// shareOf divides n by total as a 0..1 share, 0 when total is zero.
func shareOf(n, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total)
}

// exploreRead reports the neutral no-verdict Read while too few classified calls exist to
// read a pattern, rather than a favorable one earned from a handful of calls.
func exploreRead(sufficient, balanced bool) Read {
	if !sufficient {
		return noDataRead
	}
	return readFor(balanced, "Balanced")
}

// explorePurity rises with the produce share, saturating at a quarter of calls writing --
// beyond that more writing says nothing further about whether exploration is proportionate.
func explorePurity(m toolMix, sufficient bool) float64 {
	if !sufficient {
		return 0.5
	}
	return clamp01(m.ProduceShare() / 0.25)
}

// exploreRatio renders how many reads and searches accompany each write, "—" when nothing
// was written -- never a divide-by-zero standing in for a real ratio.
func exploreRatio(explore, writes int64) string {
	if writes == 0 {
		return "—"
	}
	return strconv.FormatFloat(float64(explore)/float64(writes), 'f', 1, 64) + "×"
}

// toolMixBars ranks the purpose buckets by call count, largest first.
func toolMixBars(m toolMix) []Bar {
	buckets := []struct {
		label string
		n     int64
	}{
		{"write", m.Writes},
		{"read", m.Reads},
		{"search", m.Searches},
		{"command", m.Commands},
		{"other", m.Other},
	}
	var maxN int64
	for _, b := range buckets {
		if b.n > maxN {
			maxN = b.n
		}
	}
	bars := make([]Bar, 0, len(buckets))
	for _, b := range buckets {
		bars = append(bars, Bar{
			Label: b.label,
			Value: strconv.FormatInt(b.n, 10) + " calls · " + formatPercent(shareOf(b.n, m.Classified), 0),
			Frac:  fracOf(b.n, maxN),
		})
	}
	return bars
}

// exploreCoverageCaveat states what share of the window's tool calls carry a purpose, so a
// split computed from part of the data never reads as covering all of it. The gap is
// history: a source that names no calls records none at all and so cannot lower it -- what
// does is usage ingested before the purpose split was captured.
func exploreCoverageCaveat(m toolMix) string {
	return "Prov.: " + formatPercent(m.Coverage(), 0) + " of tool calls record what they were for. Sessions ingested before this build captured it read as unclassified -- run `backfill` to restate them. A source that names no tool calls records none, so it neither raises nor lowers this -- `assaio-agent signals coverage` says which do."
}

func exploreTakeaway(sufficient, balanced bool) string {
	switch {
	case !sufficient:
		return "Too few classified tool calls this window to read the explore-versus-produce split."
	case balanced:
		return "Exploration is feeding real writes, not running in circles."
	default:
		return "Almost all tool calls read or search and very few write -- worth checking whether the agent is stuck exploring."
	}
}
