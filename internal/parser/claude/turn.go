package claude

import (
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

// appendAssistant appends a new record for an assistant turn, deduping by uuid.
func appendAssistant(l *line, cf *carryForward, seen map[string]struct{}, out []usage.Record, lastAssistant int, act *blockActivity) ([]usage.Record, int, int) {
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
	out = append(out, recordFromLine(l, cf, act))
	return out, len(out) - 1, 0
}

func recordFromLine(l *line, cf *carryForward, act *blockActivity) usage.Record {
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
		ToolCalls:        act.byPurpose.Total(),
		Edits:            act.edits,
		ToolReads:        act.byPurpose.Reads,
		ToolSearches:     act.byPurpose.Searches,
		ToolCommands:     act.byPurpose.Commands,
		ToolWrites:       act.byPurpose.Writes,
		ToolOther:        act.byPurpose.Other,
		Sidechain:        boolFlag(l.IsSidechain),
		Skill:            l.AttributionSkill,
		Agent:            l.AttributionAgent,
	}
}

// boolFlag renders a log's boolean marker in the record's 0/1 form.
func boolFlag(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
