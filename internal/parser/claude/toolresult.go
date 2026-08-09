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
	AgentID       string      `json:"agentId"`
	AgentType     string      `json:"agentType"`
	ResolvedModel string      `json:"resolvedModel"`
	Usage         *tokenUsage `json:"usage"`
	ToolStats     *toolStats  `json:"toolStats"`
	FilePath      string      `json:"filePath"`
	// Type discriminates how the edit is described: "create" carries the new file's whole
	// body in Content and an empty StructuredPatch, everything else describes itself as a
	// patch. Content is read for its line count only and is never stored (PRIVACY.md).
	//
	// It stays raw because this one struct covers unrelated tool outcomes that disagree on
	// the field's type: a created file writes a string there, a completed sub-agent writes
	// its content blocks as an array. Declaring it `string` failed the whole unmarshal for
	// the second kind, which silently dropped 477 sub-agent records and 21,369 of their
	// lines on the maintainer's corpus -- a union arm typed from one arm's evidence.
	Type            string          `json:"type"`
	Content         json.RawMessage `json:"content"`
	StructuredPatch []patchHunk     `json:"structuredPatch"`
}

// createdFileType is the toolUseResult.type Claude Code stamps on a file it created.
const createdFileType = "create"

// lineCounts returns what this edit result changed. A creation is the whole file arriving
// as Content beside an empty patch, so a reader that only walks StructuredPatch counts a
// created file as zero added lines -- which is what assaio did, for every file Claude Code
// ever wrote. Measured across 5,602 real transcripts: 5,599 creations carrying 625,012
// uncounted lines, against 382,278 the corpus reported. Codex described a creation the same
// way and lost the same lines (B119); the shared counter is parser.ContentLines.
func (t *toolResult) lineCounts() (added, removed int64) {
	if body, ok := t.createdBody(); ok {
		return parser.ContentLines(body), 0
	}
	return countPatchLines(t.StructuredPatch)
}

// createdBody reports the new file's body when this result describes a creation. Content
// that is not a string belongs to another tool outcome sharing the field, so it is no body.
func (t *toolResult) createdBody() (string, bool) {
	if t.Type != createdFileType || len(t.StructuredPatch) != 0 || len(t.Content) == 0 {
		return "", false
	}
	var body string
	if err := json.Unmarshal(t.Content, &body); err != nil {
		return "", false
	}
	return body, body != ""
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
// its parent turn); an edit result attributes its added/removed/rework line counts to the
// last assistant record. It reports whether the line matched either shape.
func (st *parseState) applyToolResult(l *line, cf *carryForward) bool {
	t, action := classifyToolResult(l.ToolUseResult)
	switch action {
	case actionSubAgent:
		st.out = append(st.out, subAgentRecord(l, &t, cf))
		return true
	case actionStub:
		return true
	case actionEdit:
		added, removed := t.lineCounts()
		st.rework.attribute(st.out, st.last, t.FilePath, added, removed)
		return true
	default:
		return false
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
// The work it aggregates ran inside a sub-agent even though the line reporting it sits in
// the parent transcript, so Sidechain is 1 regardless of the line's own marker. Granularity
// is "session": this one record totals a whole sub-agent run, and the honesty rule is that a
// summary of many requests is never labelled as one of them (docs/extending.md).
func subAgentRecord(l *line, t *toolResult, cf *carryForward) usage.Record {
	r := usage.Record{
		Tool:               tool,
		SessionID:          l.SessionID,
		Timestamp:          l.Timestamp,
		Model:              t.ResolvedModel,
		InputTokens:        parser.NonNeg(t.Usage.Input),
		OutputTokens:       parser.NonNeg(t.Usage.Output),
		CacheReadTokens:    parser.NonNeg(t.Usage.CacheRead),
		CacheWriteTokens:   parser.NonNeg(t.Usage.CacheWrite),
		CacheWrite1hTokens: t.Usage.cacheWrite1h(),
		DedupeKey:          agentDedupePrefix + t.AgentID,
		Cwd:                cf.cwd,
		Project:            cf.project(),
		GitBranch:          cf.gitBranch,
		Entrypoint:         cf.entrypoint,
		Granularity:        "session",
		Sidechain:          1,
		Agent:              t.AgentType,
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
