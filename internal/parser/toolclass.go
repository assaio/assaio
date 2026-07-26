package parser

// toolClass is what a tool call was for.
type toolClass int

const (
	classOther toolClass = iota
	classRead
	classSearch
	classCommand
	classWrite
)

// toolClasses is an allowlist: an unlisted name -- an MCP tool, a newly shipped built-in,
// anything unknown -- falls into classOther by construction rather than being guessed at.
// It holds every tool vocabulary the parsers see; the names do not collide across tools.
var toolClasses = map[string]toolClass{
	"Read":         classRead,
	"NotebookRead": classRead,
	"WebFetch":     classRead,
	"Grep":         classSearch,
	"Glob":         classSearch,
	"WebSearch":    classSearch,
	"ToolSearch":   classSearch,
	"Bash":         classCommand,
	"BashOutput":   classCommand,
	"KillShell":    classCommand,
	"Edit":         classWrite,
	"Write":        classWrite,
	"MultiEdit":    classWrite,
	"NotebookEdit": classWrite,
	// Codex CLI names its tools in lower snake_case; the vocabulary varies by version.
	"read":               classRead,
	"read_file":          classRead,
	"view":               classRead,
	"open_page":          classRead,
	"fetch":              classRead,
	"grep":               classSearch,
	"glob":               classSearch,
	"search":             classSearch,
	"web_search":         classSearch,
	"shell":              classCommand,
	"exec":               classCommand,
	"bash":               classCommand,
	"run":                classCommand,
	"local_shell":        classCommand,
	"container.exec":     classCommand,
	"edit":               classWrite,
	"write":              classWrite,
	"apply_patch":        classWrite,
	"str_replace_editor": classWrite,
}

// ToolCounts splits tool calls by purpose, mirroring usage.Record's ToolReads..ToolOther.
// Shared by the parsers whose logs name their tool calls; the name is classified here and
// then dropped -- neither it nor any tool input is ever stored (PRIVACY.md).
type ToolCounts struct {
	Reads, Searches, Commands, Writes, Other int64
}

// Add tallies one tool call under the purpose its name maps to.
func (c *ToolCounts) Add(name string) {
	switch toolClasses[name] {
	case classRead:
		c.Reads++
	case classSearch:
		c.Searches++
	case classCommand:
		c.Commands++
	case classWrite:
		c.Writes++
	default:
		c.Other++
	}
}

// Total is every bucket summed; it must equal the record's ToolCalls.
func (c *ToolCounts) Total() int64 {
	return c.Reads + c.Searches + c.Commands + c.Writes + c.Other
}
