package analyze

import (
	"strconv"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/store"
)

// attributionSources names every in-tree source that labels a turn with a skill or a
// sub-agent, alphabetically.
func attributionSources() []string {
	return sourcesAnswering(parser.SignalSkillTokens, parser.SignalAgentTokens)
}

// attributedTokens is the larger dimension's token total. Reading it off the dimension the
// share was computed in would report zero for a window holding a single label, since no
// share exists there to carry a total with it.
func attributedTokens(skills, agents []store.AttributionRow) int64 {
	return max(attributionTokens(skills), attributionTokens(agents))
}

// attributionTokens sums one attribution dimension's tokens.
func attributionTokens(rows []store.AttributionRow) int64 {
	var sum int64
	for i := range rows {
		sum += rows[i].Tokens
	}
	return sum
}

// topAttributionShare is the largest single skill's or sub-agent's share of its own
// dimension -- the concentration this metric reads -- together with that dimension's own
// token total, so a caller can never render the share against the other dimension's sum.
// Only a dimension holding at least two entries counts: with one, the share is 100%
// whatever happened. ok is false when neither dimension has two, so the caller reports no
// verdict instead of a guaranteed one. Rows arrive sorted by Tokens descending.
func topAttributionShare(skills, agents []store.AttributionRow) (share float64, total int64, ok bool) {
	for _, rows := range [][]store.AttributionRow{skills, agents} {
		if len(rows) < skillMinEntries {
			continue
		}
		dimension := attributionTokens(rows)
		if s := shareOf(rows[0].Tokens, dimension); !ok || s > share {
			share, total, ok = s, dimension, true
		}
	}
	return share, total, ok
}

// attributionBars ranks skills and sub-agents together, each labelled by dimension so a
// skill is never mistaken for an agent. topN caps each dimension separately.
func attributionBars(skills, agents []store.AttributionRow, topN int) []Bar {
	bars := make([]Bar, 0, 2*topN)
	bars = append(bars, dimensionBars(skills, "skill", topN)...)
	bars = append(bars, dimensionBars(agents, "agent", topN)...)
	return bars
}

// dimensionBars renders one attribution dimension's top entries, scaled against that
// dimension's own largest so the two dimensions stay independently readable. The name is
// the Label alone -- skill and sub-agent names are user-authored, so --anonymize replaces
// the whole Label; carrying the dimension in Value keeps it legible once that happens.
func dimensionBars(rows []store.AttributionRow, kind string, topN int) []Bar {
	kept := rows
	if topN > 0 && len(kept) > topN {
		kept = kept[:topN]
	}
	var maxTokens int64
	if len(kept) > 0 {
		maxTokens = kept[0].Tokens
	}
	total := attributionTokens(rows)
	out := make([]Bar, 0, len(kept))
	for i := range kept {
		out = append(out, Bar{
			Label: kept[i].Name,
			Value: kind + " · " + humanize.Count(kept[i].Tokens) + " tokens · " + humanize.PercentAt(shareOf(kept[i].Tokens, total), 0) + " · " + strconv.FormatInt(kept[i].Lines, 10) + " lines",
			Frac:  fracOf(kept[i].Tokens, maxTokens),
		})
	}
	return out
}
