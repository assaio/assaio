package codex

import "encoding/json"

// responseItem is a response_item payload. Name and Status matter only for the two
// tool-call kinds; Name is classified into a purpose during parsing and never stored, and
// neither are the call's arguments, input, or output (PRIVACY.md).
type responseItem struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Status string `json:"status"`
	// Role separates the model's own output from the turn's input. A rollout writes the user
	// prompt and the harness's developer instructions as message items too, and opening a
	// turn's assistant step on one of those would place the model's response before it ran.
	Role string `json:"role"`
	// CallID is what the call's output quotes, and is the step's identity: it is stable across
	// a re-read where a positional counter is only stable while the file ahead of it does not
	// change.
	CallID string `json:"call_id"`
}

// modelOutput reports whether this item is the model speaking, which is what opens a turn.
// Reasoning, an assistant message and a tool call are all responses; a user or developer
// message is the turn's input.
func (it *responseItem) modelOutput() bool {
	switch it.Type {
	case "reasoning", "function_call", "custom_tool_call":
		return true
	case "message":
		return it.Role == "assistant"
	}
	return false
}

// applyResponseItem counts a response_item payload as a tool call when it is a call
// (function_call, custom_tool_call), never its output counterpart, so a call and its
// result are never double-counted.
func (st *parseState) applyResponseItem(payload json.RawMessage) {
	var it responseItem
	if err := json.Unmarshal(payload, &it); err != nil {
		st.skipped++
		return
	}
	if it.modelOutput() {
		st.steps.openTurn(st)
	}
	switch it.Type {
	case "function_call", "custom_tool_call":
		st.pending.byPurpose.Add(it.Name)
		st.steps.toolCall(st, it.CallID, it.Name, it.Status)
		if failedStatus(it.Status) {
			st.pending.toolErrors++
		}
	}
}

// failedStatus reports whether a call's own status marks it as failed. Only explicit
// failure words count -- an absent or unknown status is never read as an error.
func failedStatus(status string) bool {
	switch status {
	case "failed", "incomplete":
		return true
	}
	return false
}
