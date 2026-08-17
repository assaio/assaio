package share

import (
	"strconv"
	"strings"

	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/store"
)

// distinctTools counts the tools that contributed usage to this window. A tool is a
// vendor's name, which the redaction rule allows on a card; a project is not.
func distinctTools(rows []store.UsageRow) int {
	seen := map[string]struct{}{}
	for i := range rows {
		if rows[i].Tool != "" {
			seen[rows[i].Tool] = struct{}{}
		}
	}
	return len(seen)
}

// rejectionCount is how many tool calls a human declined across the window: the one
// figure on the card that counts what the person did rather than what the agent did.
func rejectionCount(rows []store.UsageRow) int64 {
	var n int64
	for i := range rows {
		n += rows[i].Rejected
	}
	return n
}

// commandShare is the share of classified tool calls that ran a command, over the same
// ToolReads..ToolOther split and the same "classified calls" denominator explore-produce
// reads. It is the one figure this package derives rather than quotes, because that validator
// publishes the command share as a Bar and verdicts indexes only Figures.
func commandShare(rows []store.UsageRow) num {
	var commands, classified int64
	for i := range rows {
		r := &rows[i]
		commands += r.ToolCommands
		classified += r.ToolReads + r.ToolSearches + r.ToolCommands + r.ToolWrites + r.ToolOther
	}
	if classified == 0 {
		return num{}
	}
	frac := float64(commands) / float64(classified)
	// PercentAt, not %.0f: it renders 0.4% as "<1%" and 99.6% as ">99%" instead of "0%" and
	// "100%". Every other share on the card comes through it, and a derived one may not be the
	// exception that claims a bound it did not reach.
	return num{v: frac * 100, ok: true, s: humanize.PercentAt(frac, 0)}
}

// parseNum reads a plain decimal a validator published ("97.0"), returning not-ok for
// anything else so a caller omits what rested on it rather than substituting a zero.
func parseNum(s string) num {
	v, err := strconv.ParseFloat(strings.TrimSpace(strings.ReplaceAll(s, ",", "")), 64)
	if err != nil {
		return num{}
	}
	return num{v: v, ok: true}
}

// parseLeadingPercent reads the percentage a note opens with ("24% on weekends"), keeping
// the published token as the form any copy renders.
func parseLeadingPercent(note string) num {
	fields := strings.Fields(strings.TrimSpace(note))
	if len(fields) == 0 {
		return num{}
	}
	got := parseNum(strings.TrimSuffix(fields[0], "%"))
	got.s = fields[0]
	return got
}
