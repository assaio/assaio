package analyze

// The tool-call purpose split itself, apart from the verdict built on it: what the calls
// were for, over the sources that name a call at all.

import (
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/store"
)

// toolMix is the window's tool calls split by purpose, beside the total tool calls seen so
// coverage can be stated honestly.
type toolMix struct {
	Reads, Searches, Commands, Writes, Other int64
	// Classified is the sum of the five buckets: calls from tools that name their calls.
	Classified int64
	// AllCalls is every tool call in the window, including tools that record no names.
	AllCalls int64
}

// buildToolMix sums the per-purpose tool-call counts across the sources that name their
// tool calls at all. A source that names none contributes neither a purpose nor a call to
// the denominator: it cannot lower the coverage figure with work it was never able to
// classify, which is the difference between "these calls went unnamed" and "this tool does
// not name calls".
func buildToolMix(rows []store.UsageRow) toolMix {
	var m toolMix
	for i := range rows {
		if !parser.Answers(rows[i].Tool, parser.SignalToolCallsCount) {
			continue
		}
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
