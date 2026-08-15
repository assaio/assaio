// Package layer is the closed vocabulary of measurement layers every assaio figure states.
// Everything assaio reports sits on exactly one of them and never moves between them:
// promoting an output figure to an outcome claim is, by ROADMAP.md's own account, the most
// likely way this project starts lying. See docs/adr/0013-measurement-layers.md.
package layer

// Layer is which of the four layers a figure's verdict rests on.
type Layer string

// The four layers, in the order a claim strengthens.
const (
	// Activity is what happened: tokens, turns, tool calls, edits. Cost belongs here too --
	// it is what the activity was billed at, not evidence that anything was produced.
	Activity Layer = "activity"
	// Output is what was produced: changed lines, commits, PRs, tests. Lines reward greenfield
	// and boilerplate, punish debugging and analysis, and can grow with unnecessary
	// complexity, which is why an output figure is framed as cost efficiency and never as
	// productivity.
	Output Layer = "output"
	// Outcome is whether it held: merged, survived, passed CI, review burden, reverted. No
	// built-in validator claims this layer today; `survival` is the one local signal that
	// reaches it, and the rest needs the server stage (ROADMAP.md).
	Outcome Layer = "outcome"
	// Impact is value to a user or a business, and needs context assaio does not have. Nothing
	// here claims it. The constant exists so the vocabulary is closed rather than open-ended:
	// a surface that could not name impact would leave the word free for something else to
	// take.
	Impact Layer = "impact"
)

// All returns the four layers in order, weakest claim first.
func All() []Layer { return []Layer{Activity, Output, Outcome, Impact} }

// Valid reports whether l is one of the four. Anything else is a fifth layer nobody defined,
// which is the drift this vocabulary exists to prevent.
func Valid(l Layer) bool {
	switch l {
	case Activity, Output, Outcome, Impact:
		return true
	default:
		return false
	}
}
