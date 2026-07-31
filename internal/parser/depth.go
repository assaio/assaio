package parser

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
	// Gaps names what this source does not carry, in a reader's terms. Required below Deep.
	Gaps []string
}

// depths is the matrix, deepest first. A parser that gains a capability updates its row
// here in the same change -- this is the one place the answer lives, and doctor, the
// coverage validator and the docs all read it rather than each keeping their own list.
var depths = []Depth{
	{
		Tool: "claude-code", Tier: Deep,
		Tokens: true, Activity: true, Attribution: true,
	},
	{
		Tool: "codex", Tier: Standard,
		Tokens: true, Activity: true, Attribution: false,
		Gaps: []string{
			"no skill or sub-agent labels, so its turns are absent from the attribution split",
			"tool-use denials are not recorded, and call failures only for file edits",
		},
	},
	{
		Tool: "gemini-cli", Tier: Standard,
		Tokens: true, Activity: false, Attribution: false,
		Gaps: []string{
			"no line, edit or tool-call signals, so it contributes cost but no output figures",
			"tool-use tokens are folded into output, and ~/.gemini may be shared with other tools",
		},
	},
	{
		Tool: "copilot-cli", Tier: Standard,
		Tokens: true, Activity: true, Attribution: false,
		Gaps: []string{
			"totals exist only when a session ends, so one record covers a whole session and per-turn figures exclude it",
			"code changes are counted once per session with no per-model split, so they are credited whole to the model that made the most requests",
		},
	},
	{
		Tool: "cline", Tier: Standard,
		Tokens: true, Activity: false, Attribution: false,
		Gaps: []string{
			"no line, edit or tool-call signals, so it contributes cost but no output figures",
			"its own per-request cost is recorded but recomputed from tokens for cross-tool consistency",
		},
	},
}

// Depths returns the matrix, deepest first.
func Depths() []Depth {
	out := make([]Depth, len(depths))
	copy(out, depths)
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

// HasActivity reports whether tool's parser extracts line and edit activity, not just
// tokens and cost. A source outside the matrix reports false: the exec-plugin protocol
// carries no activity fields, so a plugin's records hold none.
func HasActivity(tool string) bool {
	d, ok := DepthOf(tool)
	return ok && d.Activity
}
