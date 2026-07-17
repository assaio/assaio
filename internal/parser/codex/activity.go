package codex

import (
	"encoding/json"
	"strings"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

// activity accumulates edit/tool-call/compaction signals seen between two token_count
// events, so they can be flushed onto the turn record the next token_count closes.
type activity struct {
	linesAdded, linesRemoved int64
	edits, toolCalls         int64
	compactions, reworkLines int64
}

// flushInto folds a into r and resets a to zero, so nothing is ever counted twice.
func (a *activity) flushInto(r *usage.Record) {
	r.LinesAdded += a.linesAdded
	r.LinesRemoved += a.linesRemoved
	r.Edits += a.edits
	r.ToolCalls += a.toolCalls
	r.Compactions += a.compactions
	r.ReworkLines += a.reworkLines
	*a = activity{}
}

// payloadKind peeks at a payload's discriminator before deciding how to decode the rest
// of it, mirroring the tolerant unmarshal-then-filter pattern tokenCount already uses.
type payloadKind struct {
	Type string `json:"type"`
}

// patchApplyEnd is the payload of a completed Codex file-edit application. Changes is
// kept as raw per-file entries so one malformed file's diff never drops the others in
// the same event.
type patchApplyEnd struct {
	Success bool                       `json:"success"`
	Changes map[string]json.RawMessage `json:"changes"`
}

// patchFileChange is one file's entry in a patch_apply_end's changes map. The map key
// (an absolute file path) is used only to scope rework tracking in memory and is never
// copied onto a usage.Record (PRIVACY.md).
type patchFileChange struct {
	UnifiedDiff string `json:"unified_diff"`
}

// applyEventMsg routes an event_msg payload by its own type: token_count carries usage
// deltas (existing behavior), patch_apply_end carries edit activity. Other event_msg
// kinds (agent_message, task_started, ...) are filtered, not skipped.
func (st *parseState) applyEventMsg(payload json.RawMessage) {
	var k payloadKind
	if err := json.Unmarshal(payload, &k); err != nil {
		st.skipped++
		return
	}
	switch k.Type {
	case "token_count":
		st.applyTokenCount(payload)
	case "patch_apply_end":
		st.applyPatchApplyEnd(payload)
	}
}

// applyResponseItem counts a response_item payload as a tool call when it is a call
// (function_call, custom_tool_call), never its output counterpart, so a call and its
// result are never double-counted.
func (st *parseState) applyResponseItem(payload json.RawMessage) {
	var k payloadKind
	if err := json.Unmarshal(payload, &k); err != nil {
		st.skipped++
		return
	}
	switch k.Type {
	case "function_call", "custom_tool_call":
		st.pending.toolCalls++
	}
}

// applyPatchApplyEnd counts one successful patch application as a single edit, then
// counts added/removed diff lines and rework per file. A failed application (success
// false) contributes nothing -- Codex already rolled it back.
func (st *parseState) applyPatchApplyEnd(payload json.RawMessage) {
	var p patchApplyEnd
	if err := json.Unmarshal(payload, &p); err != nil {
		st.skipped++
		return
	}
	if !p.Success {
		return
	}
	st.pending.edits++
	for file, raw := range p.Changes {
		added, removed := diffLineCounts(raw)
		st.pending.linesAdded += added
		st.pending.linesRemoved += removed
		st.pending.reworkLines += parser.Rework(st.addedSoFar, file, added, removed)
	}
}

// diffLineCounts counts a unified_diff's "+"/"-" prefixed body lines; a malformed entry
// (not even a valid patchFileChange) contributes 0 rather than aborting the whole event.
// File headers ("--- a/x", "+++ b/x") and the hunk header ("@@ ... @@") share the "+"/"-"
// markers but are not body lines, so they must not count as added/removed lines.
func diffLineCounts(raw json.RawMessage) (added, removed int64) {
	var c patchFileChange
	if err := json.Unmarshal(raw, &c); err != nil {
		return 0, 0
	}
	for _, ln := range strings.Split(c.UnifiedDiff, "\n") {
		switch {
		case strings.HasPrefix(ln, "+++ "), strings.HasPrefix(ln, "--- "), strings.HasPrefix(ln, "@@"):
		case strings.HasPrefix(ln, "+"):
			added++
		case strings.HasPrefix(ln, "-"):
			removed++
		}
	}
	return added, removed
}
