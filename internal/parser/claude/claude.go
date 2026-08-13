// Package claude parses Claude Code session transcripts into normalized usage records.
package claude

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

const tool = "claude-code"

// tokenUsage mirrors the token-count shape shared by an assistant message and a
// completed sub-agent's toolUseResult.
type tokenUsage struct {
	Input      int64 `json:"input_tokens"`
	Output     int64 `json:"output_tokens"`
	CacheWrite int64 `json:"cache_creation_input_tokens"`
	CacheRead  int64 `json:"cache_read_input_tokens"`
	// Creation splits CacheWrite by the cache lifetime each portion bought. Absent on
	// older transcripts, where the split is unknown rather than all-5-minute.
	Creation *struct {
		Ephemeral5m int64 `json:"ephemeral_5m_input_tokens"`
		Ephemeral1h int64 `json:"ephemeral_1h_input_tokens"`
	} `json:"cache_creation"`
}

// cacheWrite1h is the portion of this turn's cache write that bought a 1-hour lifetime,
// clamped to the write it is part of: the two disagreed on 44 of 355,063 audited turns.
func (u *tokenUsage) cacheWrite1h() int64 {
	if u.Creation == nil {
		return 0
	}
	return parser.Subset(u.Creation.Ephemeral1h, parser.NonNeg(u.CacheWrite))
}

// parseState is one Parse call's accumulator. It is scoped to a single Parse, per
// AGENTS.md's "parsers stay hermetic": nothing in it outlives the call, and the file paths
// rework keys on live only for its duration.
type parseState struct {
	out []usage.Record
	// seenLine guards a transcript that repeats a line verbatim -- a streamed retry.
	seenLine map[string]struct{}
	// inCompaction marks that the previous line was a compaction marker, so the boundary and
	// the summary Claude Code writes for one overflow count as one event.
	inCompaction bool
	// byMessage maps an API response id to the record carrying it. Claude writes one JSONL
	// line per content block of a single response, each repeating that response's usage, so
	// a record per line counted one request's tokens once per block.
	byMessage map[string]int
	// last indexes the most recent assistant record: the turn a later tool result, denial,
	// error or compaction boundary belongs to.
	last    int
	rework  reworkTracker
	skipped int
	steps   *stepRecorder
	// sessionID and timeline are taken from whichever line first names them, and stamped onto
	// every step at the end; a transcript's opening lines can precede both.
	sessionID string
	timeline  string
}

func newParseState() *parseState {
	return &parseState{
		seenLine:  make(map[string]struct{}),
		byMessage: make(map[string]int),
		last:      -1,
		rework:    make(reworkTracker),
		steps:     newStepRecorder(),
	}
}

// Parse reads a Claude Code transcript (JSONL). Assistant entries carry turn usage and,
// from their content blocks, tool-call/edit activity counts. One API response is written as
// one line per content block, all sharing a message.id and repeating that response's usage,
// so the lines of a response fold into a single record: its tokens are taken once and its
// activity summed (see mergeResponse). A completed sub-agent's toolUseResult (Task tool)
// never overlaps its parent turn's usage and is emitted as its own record, deduped by
// agentId; its async-launch stub (agentId with no usage yet) is skipped. Edit-result,
// tool-denial, failed-tool-result, and compaction-boundary lines attribute line, rework,
// rejection, error, and compaction counts to the most recently emitted assistant record.
// A line repeated verbatim is dropped by uuid. cwd, gitBranch, and entrypoint are tracked
// across all line types and stamped onto each emitted record from their latest seen value.
// skipped counts lines that failed to unmarshal as JSON or assistant entries carrying
// neither a message.id nor a uuid (DedupeKey must never be empty); a scanner-level error
// still aborts the parse.
func Parse(r io.Reader) ([]usage.Record, int, error) {
	recs, _, skipped, err := ParseAll(r)
	return recs, skipped, err
}

// ParseSteps reads the same transcript for its step sequence alone.
func ParseSteps(r io.Reader) ([]usage.Step, int, error) {
	_, steps, skipped, err := ParseAll(r)
	return steps, skipped, err
}

// ParseAll reads a transcript once and returns both readings of it: the usage records and the
// step sequence behind them. One pass rather than two, because two passes meant two orders of
// checks over the same line and the readings drifted apart in exactly the ways a shared line
// struct could not prevent -- a denial attributed differently, a token total folded by
// different arithmetic. Parse and ParseSteps are wrappers so no caller had to change.
func ParseAll(r io.Reader) ([]usage.Record, []usage.Step, int, error) {
	sc := parser.NewScanner(r)
	var cf carryForward
	st := newParseState()

	for sc.Scan() {
		raw := sc.Bytes()
		if len(raw) == 0 {
			continue
		}
		var l line
		if err := json.Unmarshal(raw, &l); err != nil {
			st.skipped++
			continue
		}
		cf.observe(&l)
		st.observeIdentity(&l)
		st.applyLine(&l, &cf)
	}
	if err := sc.Err(); err != nil {
		return st.out, st.steps.stamp(st.sessionID, st.timeline), st.skipped,
			fmt.Errorf("scan claude transcript: %w", err)
	}
	return st.out, st.steps.stamp(st.sessionID, st.timeline), st.skipped, nil
}

// observeIdentity records which session, and which sequence within it, this file belongs to.
func (st *parseState) observeIdentity(l *line) {
	if l.SessionID != "" {
		st.sessionID = l.SessionID
	}
	if l.AgentID != "" {
		st.timeline = l.AgentID
	}
}

// applyLine folds one already-unmarshaled line into the state, unless the transcript has
// already shown this uuid.
func (st *parseState) applyLine(l *line, cf *carryForward) {
	if st.seen(l.UUID) {
		return
	}
	blocks := decodeBlocks(l.Message.Content)
	if st.markDenial(l.ToolDenialKind) {
		st.steps.denial(blocks)
		return
	}
	act := countBlocks(blocks)
	st.markToolErrors(act.errors)
	wasCompacting := st.inCompaction
	if st.markCompaction(l.IsCompactSummary, l.Subtype) {
		// st.last < 0 is the case the record path drops: a compaction before any assistant turn
		// has nothing to attribute to. The step has to drop it too, or the two readings count
		// different numbers of the same event.
		if !wasCompacting && st.last >= 0 {
			st.steps.compaction(l)
		}
		return
	}
	st.steps.resolveResults(blocks)
	if st.applyToolResult(l, cf) {
		return
	}
	st.appendAssistant(l, cf, &act, blocks)
}

// markToolErrors attributes failed tool results, which a later user line carries, to the
// turn that made the calls. A denied call never reaches here: its line returns above,
// keeping the friction signal distinct from a human's refusal.
func (st *parseState) markToolErrors(errors int64) {
	if st.last < 0 {
		return
	}
	st.out[st.last].ToolErrors += errors
}

// markDenial attributes a tool-use denial to the last assistant record and reports whether
// kind indicated one.
func (st *parseState) markDenial(kind string) bool {
	if kind == "" {
		return false
	}
	if st.last >= 0 {
		st.out[st.last].Rejected++
	}
	return true
}
