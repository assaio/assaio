package claude

import (
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

// appendAssistant folds an assistant line into the records. Lines sharing a message.id are
// the content blocks of one API response, so the first one appends a record and the rest
// merge into it. st.last then points at the response *this line belonged to*, which is what a
// following tool result, denial or error is attributed to -- correct, since the result answers
// the tool_use block that just arrived. For the 8-in-48,440 response whose blocks are not
// adjacent that rewinds the cursor to the earlier record, which is the same rule rather than
// an exception to it.
func (st *parseState) appendAssistant(l *line, cf *carryForward, act *blockActivity, blocks []contentBlock) {
	if l.Type != "assistant" || l.Message.Model == "" {
		return
	}
	key := l.Message.ID
	if key == "" {
		key = l.UUID
	}
	if key == "" {
		st.skipped++
		return
	}
	if at, ok := st.byMessage[key]; ok {
		mergeResponse(&st.out[at], l, act)
		st.last = at
		st.recordStep(l, cf, key, blocks)
		return
	}
	st.out = append(st.out, recordFromLine(l, cf, act, key))
	st.last = len(st.out) - 1
	st.byMessage[key] = st.last
	st.recordStep(l, cf, key, blocks)
}

// recordStep mirrors this response into the step sequence. The token total is read back off
// the record rather than recomputed, so the two readings cannot fold a multi-line response by
// different arithmetic.
func (st *parseState) recordStep(l *line, cf *carryForward, key string, blocks []contentBlock) {
	r := &st.out[st.last]
	st.steps.assistant(l, key, parser.SumNonNeg(
		r.InputTokens, r.OutputTokens, r.CacheReadTokens, r.CacheWriteTokens,
	))
	st.steps.toolCalls(l, key, cf.cwd, blocks)
}

// mergeResponse folds another content block of an already-recorded API response into it.
// The two halves are counted differently on purpose: the response's token usage is repeated
// verbatim on every one of its lines, so summing it counted one request several times --
// the whole of what this fixes -- while the activity on each line is that block's own and
// has to be added. Output tokens are the exception among the token fields: the earlier lines
// of a response carry a partial count that grows to the true total on the last one, so the
// largest is the answer rather than the first.
func mergeResponse(r *usage.Record, l *line, act *blockActivity) {
	u := &l.Message.Usage
	r.InputTokens = max(r.InputTokens, parser.NonNeg(u.Input))
	r.OutputTokens = max(r.OutputTokens, parser.NonNeg(u.Output))
	r.CacheReadTokens = max(r.CacheReadTokens, parser.NonNeg(u.CacheRead))
	r.CacheWriteTokens = max(r.CacheWriteTokens, parser.NonNeg(u.CacheWrite))
	r.CacheWrite1hTokens = max(r.CacheWrite1hTokens, u.cacheWrite1h())

	r.ToolCalls += act.byPurpose.Total()
	r.Edits += act.edits
	r.ToolReads += act.byPurpose.Reads
	r.ToolSearches += act.byPurpose.Searches
	r.ToolCommands += act.byPurpose.Commands
	r.ToolWrites += act.byPurpose.Writes
	r.ToolOther += act.byPurpose.Other

	// A label the first block did not carry is still this response's label.
	if r.Skill == "" {
		r.Skill = l.AttributionSkill
	}
	if r.Agent == "" {
		r.Agent = l.AttributionAgent
	}
	if r.CacheMissReason == "" {
		r.CacheMissReason = parser.VocabularyToken(l.Message.Diagnostics.CacheMissReason.Type)
	}
}

func recordFromLine(l *line, cf *carryForward, act *blockActivity, dedupeKey string) usage.Record {
	return usage.Record{
		Tool:               tool,
		SessionID:          l.SessionID,
		Timestamp:          l.Timestamp,
		Model:              l.Message.Model,
		InputTokens:        parser.NonNeg(l.Message.Usage.Input),
		OutputTokens:       parser.NonNeg(l.Message.Usage.Output),
		CacheReadTokens:    parser.NonNeg(l.Message.Usage.CacheRead),
		CacheWriteTokens:   parser.NonNeg(l.Message.Usage.CacheWrite),
		CacheWrite1hTokens: l.Message.Usage.cacheWrite1h(),
		CacheMissReason:    parser.VocabularyToken(l.Message.Diagnostics.CacheMissReason.Type),
		DedupeKey:          dedupeKey,
		Cwd:                cf.cwd,
		Project:            cf.project(),
		GitBranch:          cf.gitBranch,
		Entrypoint:         cf.entrypoint,
		Granularity:        "turn",
		ToolCalls:          act.byPurpose.Total(),
		Edits:              act.edits,
		ToolReads:          act.byPurpose.Reads,
		ToolSearches:       act.byPurpose.Searches,
		ToolCommands:       act.byPurpose.Commands,
		ToolWrites:         act.byPurpose.Writes,
		ToolOther:          act.byPurpose.Other,
		Sidechain:          boolFlag(l.IsSidechain),
		Skill:              l.AttributionSkill,
		Agent:              l.AttributionAgent,
	}
}

// boolFlag renders a log's boolean marker in the record's 0/1 form.
func boolFlag(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
