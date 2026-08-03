package parser

import (
	"slices"
	"strings"
)

// The depth tiers, ordered by what a source can support. Deep is separated from Standard by
// attribution; Standard from ImportOnly by whether the figures can be attributed to a
// session at all. Granularity is a documented gap within Standard rather than a tier of its
// own: a source that totals per session still answers session questions honestly, it just
// cannot answer one asked per turn.
const (
	// Deep carries tokens, per-turn activity, and the labels that say what the work was.
	Deep = "deep"
	// Standard carries reliable usage whose gaps -- missing activity signals, or a coarser
	// record granularity -- are documented rather than implied.
	Standard = "standard"
	// ImportOnly carries billing or aggregate figures that cannot support session-level
	// conclusions.
	ImportOnly = "import-only"
)

// Depth is what one source can actually tell you. "Supports Gemini CLI" is misleading when
// it means tokens but no edits, so every source publishes its per-field capability and,
// below Deep, the specific gaps behind the label.
type Depth struct {
	Tool string
	Tier string
	// Tokens, Activity and Attribution are the three capability axes: what it cost, what
	// was produced, and what the work was labeled as.
	Tokens, Activity, Attribution bool
	// Answers lists the ids of the signals this source can actually answer (internal/signal).
	// The three axes above summarise a source for the tier table; this is the per-signal
	// truth, and the two are not interchangeable -- "has activity" is a single bit over a
	// source that reports changed lines but no edit count, no tool calls and no turns. What
	// Gaps says in prose, this says in a form the signal catalog can compute with.
	Answers []string
	// Gaps names what this source does not carry, in a reader's terms. Required below Deep.
	Gaps []string
}

// depths is the matrix, deepest first. A parser that gains a capability updates its row
// here in the same change -- this is the one place the answer lives, and doctor, the
// coverage validator and the docs all read it rather than each keeping their own list.
// The signal groups sources are built from, so a row states what it adds rather than
// repeating a list. Ids are internal/signal's; a test there asserts every one is declared.
var (
	costSignals = []string{
		"ai.tokens.total", "ai.tokens.input", "ai.tokens.output",
		"ai.tokens.cache_read", "ai.tokens.cache_write",
		"ai.cost.estimated", "ai.sessions.count",
	}
	// reasoningSignals are declared per source rather than bundled with cost: Claude Code
	// and Cline never surface a thinking count, so claiming it for them would report support
	// for a figure their records can only ever leave at zero.
	reasoningSignals = []string{"ai.tokens.reasoning"}
	// perTurnSignals need records at turn grain: a source that totals a whole session has
	// no second timestamp to measure a gap or a turn against.
	perTurnSignals = []string{"ai.turns.count", "ai.session.active_minutes"}
	lineSignals    = []string{"ai.lines.added", "ai.lines.removed"}
	editSignals    = []string{"ai.edits.count", "ai.tool_calls.count", "ai.rework.lines"}
)

func answers(groups ...[]string) []string {
	var out []string
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

var depths = []Depth{
	{
		Tool: "claude-code", Tier: Deep,
		Tokens: true, Activity: true, Attribution: true,
		Answers: answers(costSignals, perTurnSignals, lineSignals, editSignals,
			[]string{"ai.tool_errors.count", "ai.skill.tokens", "ai.agent.tokens"}),
	},
	{
		Tool: "codex", Tier: Standard,
		Tokens: true, Activity: true, Attribution: false,
		// tool_errors is absent deliberately: Codex marks failures only for file edits, so
		// counting it would read partial coverage as a clean run -- the same reason the
		// friction validator excludes it rather than treating silence as success.
		Answers: answers(costSignals, reasoningSignals, perTurnSignals, lineSignals, editSignals),
		Gaps: []string{
			"no skill or sub-agent labels, so its turns are absent from the attribution split",
			"tool-use denials are not recorded, and call failures only for file edits",
		},
	},
	{
		Tool: "gemini-cli", Tier: Standard,
		Tokens: true, Activity: false, Attribution: false,
		Answers: answers(costSignals, reasoningSignals, perTurnSignals),
		Gaps: []string{
			"no line, edit or tool-call signals, so it contributes cost but no output figures",
			"tool-use tokens are folded into output, and ~/.gemini may be shared with other tools",
		},
	},
	{
		Tool: "copilot-cli", Tier: Standard,
		Tokens: true, Activity: true, Attribution: false,
		// Lines but nothing else: a whole-session total carries no turn, edit or tool-call
		// count, so every per-turn signal is absent rather than zero.
		Answers: answers(costSignals, reasoningSignals, lineSignals),
		Gaps: []string{
			"totals exist only when a session ends, so one record covers a whole session and per-turn figures exclude it",
			"code changes are counted once per session with no per-model split, so they are credited whole to the model that made the most requests",
		},
	},
	{
		Tool: "cline", Tier: Standard,
		Tokens: true, Activity: false, Attribution: false,
		Answers: answers(costSignals, perTurnSignals),
		Gaps: []string{
			"no line, edit or tool-call signals, so it contributes cost but no output figures",
			"its own per-request cost is recorded but recomputed from tokens for cross-tool consistency",
		},
	},
}

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
// its own list, which is how a new parser used to ship half-wired.
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
