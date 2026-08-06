package parser

import (
	"slices"
	"sort"
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

// SourcesAnswering names every in-tree source answering at least one of ids, alphabetically.
// It is what a coverage caveat and the signal catalog both print, read from the matrix here
// rather than by each surface walking the matrix and testing membership for itself -- a
// second walk is a second opinion about what a source can do. An out-of-tree plugin is absent for
// the same reason DepthOf omits it: its capability is whatever its author implemented.
func SourcesAnswering(ids ...string) []string {
	var out []string
	for i := range depths {
		if answersAny(depths[i].Tool, ids) {
			out = append(out, depths[i].Tool)
		}
	}
	sort.Strings(out)
	return out
}

func answersAny(tool string, ids []string) bool {
	for _, id := range ids {
		if Answers(tool, id) {
			return true
		}
	}
	return false
}

// SignalsAnsweredBy lists the signal ids tool produces, sorted. It is Answers asked the
// other way round, for a consumer that has to be told a source's capability rather than
// query it -- an out-of-tree metric plugin reads JSON and cannot call Answers at all, so
// without this every plugin is exposed to the silence-as-zero bug ADR 0011 fixed in-tree.
func SignalsAnsweredBy(tool string) []string {
	d, ok := DepthOf(tool)
	switch {
	case ok:
	case strings.HasPrefix(tool, PluginPrefix):
		d = pluginDepth
	default:
		return nil
	}
	out := make([]string, len(d.Answers))
	copy(out, d.Answers)
	sort.Strings(out)
	return out
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
