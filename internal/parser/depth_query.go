package parser

import (
	"slices"
	"strings"
)

// The questions every other package asks the matrix. Capability is answered here and only
// here, so no surface keeps a second opinion about what a source can do.

// PluginPrefix namespaces an out-of-tree exec parser's records, so a plugin can never
// impersonate a built-in source (internal/plugin, ADR 0003).
const PluginPrefix = "plugin:"

// pluginDepth is what any out-of-tree exec parser can be assumed to support. Its record
// shape (ADR 0003) has no line, edit or tool-call field at all, and it declares turn or
// session grain per record -- so a per-turn signal cannot be assumed either, for the same
// reason a session-total source answers none.
var pluginDepth = Depth{
	Tier: Standard, Tokens: true,
	Answers: answers(costSignals, reasoningSignals),
	Gaps: []string{
		"the exec parser protocol carries no line, edit or tool-call fields",
		"grain is declared per record, so per-turn signals cannot be assumed",
	},
}

// Depths returns the matrix, deepest first.
func Depths() []Depth {
	out := make([]Depth, len(depths))
	copy(out, depths)
	return out
}

// Tools names every in-tree source, deepest first. Every surface that has to know which
// tools exist -- sync validation, `clear --tool`, the docs -- reads this instead of keeping
// its own list.
func Tools() []string {
	out := make([]string, len(depths))
	for i := range depths {
		out[i] = depths[i].Tool
	}
	return out
}

// Answers reports whether tool produces the data behind signal id -- the one capability
// question every surface asks, so no metric keeps a second opinion about a source. A name
// outside the matrix answers the exec protocol's floor only when it is a plugin: a collector
// that is not a usage source at all (internal/vcs) answers none of these.
func Answers(tool, id string) bool {
	d, ok := DepthOf(tool)
	switch {
	case ok:
	case strings.HasPrefix(tool, PluginPrefix):
		d = pluginDepth
	default:
		return false
	}
	return slices.Contains(d.Answers, id)
}

// DepthOf returns tool's declared depth. An out-of-tree exec plugin is absent by design:
// its depth is whatever its author implemented, so it is read from the records it actually
// emits rather than promised by a table assaio maintains.
func DepthOf(tool string) (Depth, bool) {
	for i := range depths {
		if depths[i].Tool == tool {
			return depths[i], true
		}
	}
	return Depth{}, false
}

// HasFullActivity reports whether tool answers every activity signal -- lines, edits, tool
// calls and rework -- rather than merely one of them. Deliberately not named for the Activity
// axis above: that axis is the tier table's one-bit summary and reads true for a source that
// records changed lines and nothing else, so the two would disagree under one name (ADR 0008).
func HasFullActivity(tool string) bool { return answersAll(tool, answers(lineSignals, editSignals)) }

// HasLineOutput reports whether tool contributes changed-line counts at all. That is the
// question behind "cost only", and it is not the same one as full activity capture.
func HasLineOutput(tool string) bool { return answersAll(tool, lineSignals) }

func answersAll(tool string, ids []string) bool {
	for _, id := range ids {
		if !Answers(tool, id) {
			return false
		}
	}
	return true
}
