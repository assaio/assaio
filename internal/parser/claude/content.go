package claude

import (
	"encoding/json"

	"github.com/assaio/assaio/internal/parser"
)

// contentBlock is one entry of a message's content array. IsError is carried only by the
// tool_result blocks a user line answers a tool call with.
type contentBlock struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	IsError bool   `json:"is_error"`
}

// editToolNames are the assistant tool_use block names that edit files.
var editToolNames = map[string]bool{
	"Edit":         true,
	"Write":        true,
	"NotebookEdit": true,
	"MultiEdit":    true,
}

// blockActivity is what one message's content blocks contribute to a turn: an assistant
// line supplies the tool_use side, the user line answering it the failed-result side.
type blockActivity struct {
	edits, errors int64
	byPurpose     parser.ToolCounts
}

// countBlocks walks one message's content blocks. raw is a plain string on most user
// messages, which carries no blocks: the mismatched unmarshal is ignored and reported as
// zero rather than propagated.
func countBlocks(raw json.RawMessage) blockActivity {
	var blocks []contentBlock
	var a blockActivity
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return a
	}
	for _, b := range blocks {
		switch b.Type {
		case "tool_use":
			a.byPurpose.Add(b.Name)
			if editToolNames[b.Name] {
				a.edits++
			}
		case "tool_result":
			if b.IsError {
				a.errors++
			}
		}
	}
	return a
}
