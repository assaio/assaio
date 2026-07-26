package cli

import (
	"fmt"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// demoWindowDays is the span the bundled sample data covers, and the window demo queries --
// wide enough for the week-over-week trend and staleness signals to be real.
const demoWindowDays = 30

// demoProjects are the sample repositories usage is spread across, so report/effectiveness
// show a real per-project breakdown.
var demoProjects = []string{"web-app", "api-service", "data-pipeline"}

// demoModels weights the sample mix toward premium opus, with cheaper sonnet and a small
// slice of an unpriced model, so the litmus shows a confident premium/cheaper split, a real
// est.-savings line (opus repriced on sonnet), and the honest "unpriced (unknown model)"
// figure -- "acme-llm-v2" is deliberately absent from the price table.
var demoModels = []string{
	"claude-opus-4-5", "claude-opus-4-5", "claude-opus-4-5",
	"claude-sonnet-4-5", "claude-sonnet-4-5", "acme-llm-v2",
}

// demoAttribution cycles the sample's skill and sub-agent labels, never both on one turn,
// with unlabelled turns between them -- a real log attributes only part of its work. Three
// labels per dimension is what makes the concentration read meaningful: with one, its share
// is 100% by construction.
var demoAttribution = []struct{ Skill, Agent string }{
	{Skill: "code-review"},
	{},
	{Agent: "general-purpose"},
	{},
	{Skill: "test-authoring"},
	{},
	{Agent: "explore"},
	{},
	{Skill: "docs"},
	{},
	{Agent: "test-runner"},
	{},
}

// demoRecords builds the bundled sample usage: a cycle of sessions per active day across
// demoWindowDays, deterministic (no randomness) so the demo reads identically every run.
// It skips one day each week so active-days stays below the window, keeping the pace honest.
func demoRecords(now time.Time) []usage.Record {
	var recs []usage.Record
	seq := 0
	for d := demoWindowDays - 1; d >= 1; d-- {
		day := demoDayStart(now.AddDate(0, 0, -d))
		// Weekends carry no sample sessions: the fixture depicts an ordinary working week,
		// and seeding weekend work would make the demo's own rhythm verdict depend on which
		// weekday "today" happens to be in the reader's timezone.
		if wd := day.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		for s := 0; s <= d%2; s++ {
			recs = append(recs, demoSession(day, d, s, &seq)...)
		}
	}
	return recs
}

// demoDayStart pins a sample day to mid-morning. Inheriting the hour from the invocation
// clock made the identical fixture read as ordinary working hours before lunch and as
// off-hours work after dinner -- a verdict that moved with nothing but the wall clock.
func demoDayStart(day time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), 10, 0, 0, 0, day.Location())
}

// demoSession builds one session: several turn records sharing a session_id, minutes apart
// (so ActiveMinutes accrues), on one project and model, with realistic tokens/lines/edits
// and an occasional compaction on longer sessions.
func demoSession(day time.Time, dayIdx, sIdx int, seq *int) []usage.Record {
	model := demoModels[(dayIdx+sIdx)%len(demoModels)]
	project := demoProjects[(dayIdx+sIdx)%len(demoProjects)]
	sessionID := fmt.Sprintf("demo-%d-%d", dayIdx, sIdx)
	turns := 3 + (dayIdx+sIdx)%5
	recs := make([]usage.Record, 0, turns)
	for tIdx := range turns {
		*seq++
		edits := 1 + int64(tIdx%3)
		calls := demoToolSplit(tIdx, edits)
		labels := demoAttribution[(dayIdx+sIdx+tIdx)%len(demoAttribution)]
		recs = append(recs, usage.Record{
			Tool:             "claude-code",
			SessionID:        sessionID,
			Timestamp:        day.Add(time.Duration(tIdx*4) * time.Minute),
			Model:            model,
			InputTokens:      1200 + int64(tIdx)*300,
			OutputTokens:     600 + int64(tIdx)*150,
			CacheReadTokens:  8000 + int64(tIdx)*1500,
			CacheWriteTokens: 400,
			DedupeKey:        fmt.Sprintf("demo-%d", *seq),
			Project:          project,
			Entrypoint:       "cli",
			Granularity:      "turn",
			LinesAdded:       12 + int64((dayIdx+tIdx)%20),
			LinesRemoved:     int64((dayIdx + tIdx) % 5),
			Edits:            edits,
			ToolCalls:        calls.total(),
			ToolReads:        calls.Reads,
			ToolSearches:     calls.Searches,
			ToolCommands:     calls.Commands,
			ToolWrites:       calls.Writes,
			ToolOther:        calls.Other,
			ToolErrors:       demoToolErrors(dayIdx, tIdx),
			Rejected:         int64((dayIdx + tIdx) % 3 / 2),
			ReworkLines:      int64((dayIdx + tIdx) % 4),
			Compactions:      demoCompactions(tIdx, turns),
			Skill:            labels.Skill,
			Agent:            labels.Agent,
			Sidechain:        demoSidechain(labels.Agent),
		})
	}
	return recs
}

// demoToolCalls is one turn's tool calls split by purpose. The buckets are the source of
// truth and ToolCalls is their sum, the invariant every parser holds and the sync boundary
// enforces (internal/server.validateToolPurposeSplit).
type demoToolCalls struct{ Reads, Searches, Commands, Writes, Other int64 }

func (c demoToolCalls) total() int64 { return c.Reads + c.Searches + c.Commands + c.Writes + c.Other }

// demoToolSplit is a read-heavy mix with one write per edit -- the shape of real agent work,
// and enough writes for the explore/produce split to read as balanced rather than stuck.
func demoToolSplit(tIdx int, edits int64) demoToolCalls {
	return demoToolCalls{
		Reads:    2 + int64(tIdx%3),
		Searches: 1 + int64(tIdx%2),
		Commands: 1 + int64((tIdx+1)%2),
		Writes:   edits,
		Other:    int64(tIdx % 2),
	}
}

// demoToolErrors fails one call on some turns: enough friction for the panel to read a real
// rate, well under the share that says the setup itself is broken.
func demoToolErrors(dayIdx, tIdx int) int64 {
	if (dayIdx+tIdx)%5 == 0 {
		return 1
	}
	return 0
}

// demoSidechain marks a sub-agent-labelled turn as having run inside one, so the sample's
// delegation share and its agent labels tell the same story.
func demoSidechain(agent string) int64 {
	if agent == "" {
		return 0
	}
	return 1
}

// demoCompactions marks the last turn of a long session as compacted, so context health
// carries a little signal without reading as unhealthy.
func demoCompactions(tIdx, turns int) int64 {
	if turns >= 7 && tIdx == turns-1 {
		return 1
	}
	return 0
}
