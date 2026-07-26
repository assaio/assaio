package codex

import "encoding/json"

// responseItem is a response_item payload. Name and Status matter only for the two
// tool-call kinds; Name is classified into a purpose during parsing and never stored, and
// neither are the call's arguments, input, or output (PRIVACY.md).
type responseItem struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Status string `json:"status"`
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
	switch it.Type {
	case "function_call", "custom_tool_call":
		st.pending.byPurpose.Add(it.Name)
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
