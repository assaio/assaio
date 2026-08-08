package codex

import (
	"encoding/json"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

// activity accumulates edit/tool-call/compaction signals seen between two token_count
// events, so they can be flushed onto the turn record the next token_count closes.
type activity struct {
	linesAdded, linesRemoved int64
	edits, compactions       int64
	reworkLines, toolErrors  int64
	byPurpose                parser.ToolCounts
	// patchWrites counts file edits Codex reports as their own patch_apply_end event
	// instead of as a named tool call. Without them a Codex window's write bucket is
	// always empty and the explore-versus-produce split reads 0% produced whatever the
	// agent did.
	patchWrites int64
}

// flushInto folds a into r and resets a to zero, so nothing is ever counted twice. The
// tool-call total is the purpose split summed, so the two can never disagree.
func (a *activity) flushInto(r *usage.Record) {
	r.LinesAdded += a.linesAdded
	r.LinesRemoved += a.linesRemoved
	r.Edits += a.edits
	r.Compactions += a.compactions
	r.ReworkLines += a.reworkLines
	r.ToolErrors += a.toolErrors
	// A version that names its patch calls already counted them, so the event-derived
	// count is dropped rather than added twice.
	patches := a.patchWrites
	if a.byPurpose.Writes > 0 {
		patches = 0
	}
	r.ToolCalls += a.byPurpose.Total() + patches
	r.ToolReads += a.byPurpose.Reads
	r.ToolSearches += a.byPurpose.Searches
	r.ToolCommands += a.byPurpose.Commands
	r.ToolWrites += a.byPurpose.Writes + patches
	r.ToolOther += a.byPurpose.Other
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

// applyPatchApplyEnd counts one successful patch application as a single edit, then
// counts added/removed diff lines and rework per file. A failed application (success
// false) contributes no lines -- Codex already rolled it back -- only a tool error.
func (st *parseState) applyPatchApplyEnd(payload json.RawMessage) {
	var p patchApplyEnd
	if err := json.Unmarshal(payload, &p); err != nil {
		st.skipped++
		return
	}
	// Counted on failure too: the write was attempted, and an error with no call to divide
	// by makes every rate over the two meaningless.
	st.pending.patchWrites++
	if !p.Success {
		st.pending.toolErrors++
		return
	}
	st.pending.edits++
	for file, raw := range p.Changes {
		added, removed := changeLineCounts(raw)
		st.pending.linesAdded += added
		st.pending.linesRemoved += removed
		st.pending.reworkLines += parser.Rework(st.addedSoFar, file, added, removed)
	}
}
