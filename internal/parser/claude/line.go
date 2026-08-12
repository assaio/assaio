package claude

import (
	"encoding/json"
	"path/filepath"
	"time"
)

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
	// AgentID names the sub-agent a line belongs to. A sub-agent transcript records its
	// PARENT's sessionId, so this is the only thing separating its sequence from the parent's.
	AgentID string `json:"agentId"`
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
		// StopReason is why the response ended. Read for the step timeline's outcome only:
		// on the audited corpus it is tool_use 18,330 times, end_turn 512 and max_tokens on
		// 5 of 5,706 transcripts, so no signal claims anything about truncation.
		StopReason string `json:"stop_reason"`
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
