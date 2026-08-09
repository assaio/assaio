// Package claude parses Claude Code session transcripts into normalized usage records.
package claude

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

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

type line struct {
	Type           string          `json:"type"`
	UUID           string          `json:"uuid"`
	Timestamp      time.Time       `json:"timestamp"`
	SessionID      string          `json:"sessionId"`
	Cwd            string          `json:"cwd"`
	GitBranch      string          `json:"gitBranch"`
	Entrypoint     string          `json:"entrypoint"`
	ToolUseResult  json.RawMessage `json:"toolUseResult"`
	ToolDenialKind string          `json:"toolDenialKind"`
	// IsSidechain marks a line written inside a sub-agent's own transcript.
	IsSidechain bool `json:"isSidechain"`
	// AttributionSkill and AttributionAgent are the skill and sub-agent-type labels Claude
	// Code stamps on a turn; category labels only, never prompt or file content.
	AttributionSkill string `json:"attributionSkill"`
	AttributionAgent string `json:"attributionAgent"`
	// IsCompactSummary and Subtype mark a context-compaction event: the transcript's
	// context overflowed and was auto-summarized. Either discriminates it.
	IsCompactSummary bool   `json:"isCompactSummary"`
	Subtype          string `json:"subtype"`
	Message          struct {
		Model string `json:"model"`
		// Content is a plain string on ordinary user messages and an array of blocks
		// on assistant turns; kept raw so a user line's string form never fails the
		// line's outer unmarshal.
		// ID is the API response's own id. Several JSONL lines carry the same one when a
		// response has several content blocks, and each repeats that response's usage.
		ID      string          `json:"id"`
		Content json.RawMessage `json:"content"`
		Usage   tokenUsage      `json:"usage"`
		// Diagnostics carries the vendor's own reason a prompt could not be served from
		// cache -- a closed vocabulary, never prompt or file content.
		Diagnostics struct {
			CacheMissReason struct {
				Type string `json:"type"`
			} `json:"cache_miss_reason"`
		} `json:"diagnostics"`
	} `json:"message"`
}

// carryForward tracks cwd, gitBranch, and entrypoint across all line types in a
// transcript, so each emitted record can be stamped with their latest seen value.
type carryForward struct {
	cwd, gitBranch, entrypoint string
}

func (c *carryForward) observe(l *line) {
	if l.Cwd != "" {
		c.cwd = l.Cwd
	}
	if l.GitBranch != "" {
		c.gitBranch = l.GitBranch
	}
	if l.Entrypoint != "" {
		c.entrypoint = l.Entrypoint
	}
}

func (c *carryForward) project() string {
	if c.cwd == "" {
		return ""
	}
	return filepath.Base(c.cwd)
}

// parseState is one Parse call's accumulator. It is scoped to a single Parse, per
// AGENTS.md's "parsers stay hermetic": nothing in it outlives the call, and the file paths
// rework keys on live only for its duration.
type parseState struct {
	out []usage.Record
	// seenLine guards a transcript that repeats a line verbatim -- a streamed retry.
	seenLine map[string]struct{}
	// byMessage maps an API response id to the record carrying it. Claude writes one JSONL
	// line per content block of a single response, each repeating that response's usage, so
	// a record per line counted one request's tokens once per block.
	byMessage map[string]int
	// last indexes the most recent assistant record: the turn a later tool result, denial,
	// error or compaction boundary belongs to.
	last    int
	rework  reworkTracker
	skipped int
}

func newParseState() *parseState {
	return &parseState{
		seenLine:  make(map[string]struct{}),
		byMessage: make(map[string]int),
		last:      -1,
		rework:    make(reworkTracker),
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
		st.applyLine(&l, &cf)
	}
	if err := sc.Err(); err != nil {
		return st.out, st.skipped, fmt.Errorf("scan claude transcript: %w", err)
	}
	return st.out, st.skipped, nil
}

// applyLine folds one already-unmarshaled line into the state, unless the transcript has
// already shown this uuid.
func (st *parseState) applyLine(l *line, cf *carryForward) {
	if st.seen(l.UUID) {
		return
	}
	if st.markDenial(l.ToolDenialKind) {
		return
	}
	act := countBlocks(l.Message.Content)
	st.markToolErrors(act.errors)
	if st.markCompaction(l.IsCompactSummary, l.Subtype) {
		return
	}
	if st.applyToolResult(l, cf) {
		return
	}
	st.appendAssistant(l, cf, &act)
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
