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
	// ID names a tool_use block; ToolUseID is how the tool_result answering it quotes that
	// name. The pair is what lets a step's outcome land on the call it belongs to.
	ID        string `json:"id"`
	ToolUseID string `json:"tool_use_id"`
	// Input is a call's own arguments, kept raw for the reason toolResult.Content is: tools
	// disagree on the field's type, and one non-object input would fail the whole content
	// array's unmarshal and take every block on the line with it.
	Input json.RawMessage `json:"input"`
}

// toolTarget is the narrow view of a call's input: the file it names, and nothing else. Two
// fields are the whole struct deliberately -- an edit's input carries the code being written,
// and a struct that does not name a field never holds it (PRIVACY.md).
type toolTarget struct {
	FilePath     string `json:"file_path"`
	NotebookPath string `json:"notebook_path"`
}

// targetPath is the file this call names, "" when it names none. Read from the call's own
// input rather than from its result, which is where the edit path used to come from: a result
// path exists only for an edit that succeeded, so 344 failed edits and every one of 36,846
// reads carried no target at all on the maintainer's store.
func (b *contentBlock) targetPath() string {
	if len(b.Input) == 0 {
		return ""
	}
	var t toolTarget
	if err := json.Unmarshal(b.Input, &t); err != nil {
		return ""
	}
	if t.FilePath != "" {
		return t.FilePath
	}
	return t.NotebookPath
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

// decodeBlocks reads one message's content array. raw is a plain string on most user
// messages, which carries no blocks: the mismatched unmarshal is ignored and reported as
// none rather than propagated.
func decodeBlocks(raw json.RawMessage) []contentBlock {
	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil
	}
	return blocks
}

// countBlocks walks one message's already-decoded content blocks.
func countBlocks(blocks []contentBlock) blockActivity {
	var a blockActivity
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
