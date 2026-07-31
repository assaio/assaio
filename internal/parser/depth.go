package parser

// The depth tiers, ordered by what a source can support. The discriminator between the
// first two is attribution; the discriminator for the third is record granularity, since a
// source that only reports whole-session or daily aggregates cannot answer a question
// asked per turn no matter how accurate its totals are.
const (
	// Deep carries tokens, per-turn activity, and the labels that say what the work was.
	Deep = "deep"
	// Standard carries reliable per-turn usage with activity gaps that are documented.
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
