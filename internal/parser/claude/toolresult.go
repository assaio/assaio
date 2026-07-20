package claude

import (
	"encoding/json"
	"strings"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

// toolResult is the shape of a line's toolUseResult field. It covers two unrelated
// tool outcomes that share the same wire location: a completed sub-agent (Task tool)
// call, identified by AgentID with a populated Usage, and a file edit, identified by a
// non-nil StructuredPatch. An AgentID with a nil Usage is a sub-agent's async-launch
// stub, not yet complete. FilePath identifies an edit's target file for in-memory
// rework tracking ONLY -- it is never copied onto a usage.Record or stored (PRIVACY.md).
type toolResult struct {
	AgentID         string      `json:"agentId"`
	ResolvedModel   string      `json:"resolvedModel"`
	Usage           *tokenUsage `json:"usage"`
	ToolStats       *toolStats  `json:"toolStats"`
	FilePath        string      `json:"filePath"`
	StructuredPatch []patchHunk `json:"structuredPatch"`
}

// toolStats summarizes a sub-agent's own edits.
type toolStats struct {
	LinesAdded   int64 `json:"linesAdded"`
	LinesRemoved int64 `json:"linesRemoved"`
}

// patchHunk is one hunk of an edit-tool result's structuredPatch; only the diff lines
// matter for counting added/removed code.
type patchHunk struct {
	Lines []string `json:"lines"`
}

type resultAction int

const (
	actionNone resultAction = iota
	actionSubAgent
	actionStub
	actionEdit
)

// applyToolResult interprets l's toolUseResult: a completed sub-agent call appends its
// own additional usage record (deduped by agentId, since sub-agent usage never overlaps
// its parent turn); an edit result attributes its added/removed/rework line counts to
// out[lastAssistant] via rt. It reports whether the line matched either shape.
func applyToolResult(l *line, cf *carryForward, out []usage.Record, lastAssistant int, rt reworkTracker) ([]usage.Record, bool) {
	t, action := classifyToolResult(l.ToolUseResult)
	switch action {
	case actionSubAgent:
		return append(out, subAgentRecord(l, &t, cf)), true
	case actionStub:
		return out, true
	case actionEdit:
		added, removed := countPatchLines(t.StructuredPatch)
		rt.attribute(out, lastAssistant, t.FilePath, added, removed)
		return out, true
	default:
		return out, false
	}
}

func classifyToolResult(raw json.RawMessage) (toolResult, resultAction) {
	t, ok := parseToolResult(raw)
	if !ok {
		return toolResult{}, actionNone
	}
	switch {
	case t.AgentID != "" && t.Usage != nil:
		return t, actionSubAgent
	case t.AgentID != "":
		return t, actionStub
	case t.StructuredPatch != nil:
		return t, actionEdit
	default:
		return t, actionNone
	}
}

func parseToolResult(raw json.RawMessage) (toolResult, bool) {
	if len(raw) == 0 {
		return toolResult{}, false
	}
	var t toolResult
	if err := json.Unmarshal(raw, &t); err != nil {
		return toolResult{}, false
	}
	return t, true
}

// subAgentRecord builds the additional usage record for a completed sub-agent call.
// DedupeKey is the agentId: unique and stable, so re-parsing a transcript is idempotent.
func subAgentRecord(l *line, t *toolResult, cf *carryForward) usage.Record {
	r := usage.Record{
		Tool:             tool,
		SessionID:        l.SessionID,
		Timestamp:        l.Timestamp,
		Model:            t.ResolvedModel,
		InputTokens:      parser.NonNeg(t.Usage.Input),
		OutputTokens:     parser.NonNeg(t.Usage.Output),
		CacheReadTokens:  parser.NonNeg(t.Usage.CacheRead),
		CacheWriteTokens: parser.NonNeg(t.Usage.CacheWrite),
		DedupeKey:        agentDedupePrefix + t.AgentID,
		Cwd:              cf.cwd,
		Project:          cf.project(),
		GitBranch:        cf.gitBranch,
		Entrypoint:       cf.entrypoint,
		Granularity:      "turn",
	}
	if t.ToolStats != nil {
		r.LinesAdded = parser.NonNeg(t.ToolStats.LinesAdded)
		r.LinesRemoved = parser.NonNeg(t.ToolStats.LinesRemoved)
	}
	return r
}

// countPatchLines counts "+" and "-" prefixed diff lines across every hunk. The code
// content itself is never inspected beyond its prefix.
func countPatchLines(hunks []patchHunk) (added, removed int64) {
	for _, h := range hunks {
		for _, ln := range h.Lines {
			switch {
			case strings.HasPrefix(ln, "+"):
				added++
			case strings.HasPrefix(ln, "-"):
				removed++
			}
		}
	}
	return added, removed
}
