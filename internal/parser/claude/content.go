package claude

import "encoding/json"

// contentBlock is one entry of an assistant message's content array.
type contentBlock struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// editToolNames are the assistant tool_use block names that edit files.
var editToolNames = map[string]bool{
	"Edit":         true,
	"Write":        true,
	"NotebookEdit": true,
	"MultiEdit":    true,
}

// countToolUse counts every tool_use content block and, among those, the file-editing
// ones. raw is a plain string on most user messages, which is not an edit-tool call:
// the mismatched unmarshal is ignored and reported as zero rather than propagated.
func countToolUse(raw json.RawMessage) (toolCalls, edits int64) {
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return 0, 0
	}
	for _, b := range blocks {
		if b.Type != "tool_use" {
			continue
		}
		toolCalls++
		if editToolNames[b.Name] {
			edits++
		}
	}
	return toolCalls, edits
}
