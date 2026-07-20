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
	// IsCompactSummary and Subtype mark a context-compaction event: the transcript's
	// context overflowed and was auto-summarized. Either discriminates it.
	IsCompactSummary bool   `json:"isCompactSummary"`
	Subtype          string `json:"subtype"`
	Message          struct {
		Model string `json:"model"`
		// Content is a plain string on ordinary user messages and an array of blocks
		// on assistant turns; kept raw so a user line's string form never fails the
		// line's outer unmarshal.
		Content json.RawMessage `json:"content"`
		Usage   tokenUsage      `json:"usage"`
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

// Parse reads a Claude Code transcript (JSONL). Assistant entries carry turn usage and,
// from their content blocks, tool-call/edit activity counts. A completed sub-agent's
// toolUseResult (Task tool) never overlaps its parent turn's usage and is emitted as its
// own record, deduped by agentId; its async-launch stub (agentId with no usage yet) is
// skipped. Edit-result, tool-denial, and compaction-boundary lines attribute line,
// rework, rejection, and compaction counts to the most recently emitted assistant
// record; rework tracking (reworkTracker) is scoped to this single Parse call, per
// AGENTS.md's "parsers stay hermetic" -- it never touches the filesystem, and the file
// paths it keys on live only for this call's duration. Assistant records are
// de-duplicated by uuid to avoid double-counting streamed retries. cwd, gitBranch, and
// entrypoint are tracked across all line types and stamped onto each emitted record from
// their latest seen value. skipped counts lines that failed to unmarshal as JSON or
// assistant entries missing a uuid (DedupeKey must never be empty); a scanner-level
// error still aborts the parse.
func Parse(r io.Reader) ([]usage.Record, int, error) {
	sc := parser.NewScanner(r)
	seen := make(map[string]struct{})
	var cf carryForward
	var out []usage.Record
	var skipped int
	lastAssistant := -1
	rt := make(reworkTracker)

	for sc.Scan() {
		raw := sc.Bytes()
		if len(raw) == 0 {
			continue
		}
		var l line
		if err := json.Unmarshal(raw, &l); err != nil {
			skipped++
			continue
		}
		cf.observe(&l)
		var skippedDelta int
		out, lastAssistant, skippedDelta = applyLine(&l, &cf, seen, out, lastAssistant, rt)
		skipped += skippedDelta
	}
	if err := sc.Err(); err != nil {
		return out, skipped, fmt.Errorf("scan claude transcript: %w", err)
	}
	return out, skipped, nil
}

// applyLine folds one already-unmarshaled line into out, returning the (possibly grown)
// records, the index of the most recently emitted assistant record, and a skipped count
// delta.
func applyLine(l *line, cf *carryForward, seen map[string]struct{}, out []usage.Record, lastAssistant int, rt reworkTracker) ([]usage.Record, int, int) {
	if markDenial(out, lastAssistant, l.ToolDenialKind) {
		return out, lastAssistant, 0
	}
	if markCompaction(out, lastAssistant, l.IsCompactSummary, l.Subtype) {
		return out, lastAssistant, 0
	}
	if next, handled := applyToolResult(l, cf, out, lastAssistant, rt); handled {
		return next, lastAssistant, 0
	}
	return appendAssistant(l, cf, seen, out, lastAssistant)
}

// markDenial attributes a tool-use denial to out[lastAssistant] and reports whether kind
// indicated one.
func markDenial(out []usage.Record, lastAssistant int, kind string) bool {
	if kind == "" {
		return false
	}
	if lastAssistant >= 0 {
		out[lastAssistant].Rejected++
	}
	return true
}

// appendAssistant appends a new record for an assistant turn, deduping by uuid.
func appendAssistant(l *line, cf *carryForward, seen map[string]struct{}, out []usage.Record, lastAssistant int) ([]usage.Record, int, int) {
	if l.Type != "assistant" || l.Message.Model == "" {
		return out, lastAssistant, 0
	}
	if l.UUID == "" {
		return out, lastAssistant, 1
	}
	if _, dup := seen[l.UUID]; dup {
		return out, lastAssistant, 0
	}
	seen[l.UUID] = struct{}{}
	out = append(out, recordFromLine(l, cf))
	return out, len(out) - 1, 0
}

func recordFromLine(l *line, cf *carryForward) usage.Record {
	toolCalls, edits := countToolUse(l.Message.Content)
	return usage.Record{
		Tool:             tool,
		SessionID:        l.SessionID,
		Timestamp:        l.Timestamp,
		Model:            l.Message.Model,
		InputTokens:      parser.NonNeg(l.Message.Usage.Input),
		OutputTokens:     parser.NonNeg(l.Message.Usage.Output),
		CacheReadTokens:  parser.NonNeg(l.Message.Usage.CacheRead),
		CacheWriteTokens: parser.NonNeg(l.Message.Usage.CacheWrite),
		DedupeKey:        l.UUID,
		Cwd:              cf.cwd,
		Project:          cf.project(),
		GitBranch:        cf.gitBranch,
		Entrypoint:       cf.entrypoint,
		Granularity:      "turn",
		ToolCalls:        toolCalls,
		Edits:            edits,
	}
}
