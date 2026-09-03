package trace

import "github.com/assaio/assaio/internal/store"

// The vocabulary a detector declares itself over, and the one rule that decides which member a
// sequence belongs to. Kept apart from the Set that holds the sequences: what the populations ARE
// is a domain decision, and reading a scope out of a set is mechanics.

// The scopes a sequence belongs to. Declaring one is not a convenience: 89% of the main-loop
// sequences on the audited store are Programmatic and hold 5.7% of its steps, so the two
// populations cannot share a figure.
const (
	// Interactive is a person's own session: the main loop of a run started from a terminal.
	Interactive = "interactive"
	// SubAgent is a sequence a sub-agent ran inside another session. Its thrashing belongs to a
	// tool the person delegated to, not to the person, which is why it is never blended in.
	SubAgent = "sub-agent"
	// Programmatic is an SDK caller's run -- scripted or one-shot, with nobody there to interrupt.
	Programmatic = "programmatic"
	// Unstated is a run whose source records no entrypoint at all. It is a scope of its own
	// rather than a bucket the others absorb: Copilot CLI writes none, and folding its
	// interactive work into Programmatic would be a claim about it nothing measured.
	Unstated = "unstated"
)

// The entrypoints a source writes. Only "cli" is a person at a terminal; the others are a
// library or a script calling the tool. Claude Code writes the three sdk-/cli names; Codex
// names the same distinction in its session_meta.source -- "exec" for a scripted `codex exec`
// run, "cli" for its terminal UI, and the two words mean there what they mean here.
//
// "vscode", which Codex writes for its desktop app, is deliberately absent: a person drives it,
// but not from a terminal, and Interactive is a claim about a terminal session. It falls to
// Unstated with everything else this build has not been shown, which is 1 session of 2,500 on
// the audited corpus -- an IDE scope for that is an abstraction with no demand behind it.
const (
	entrypointCLI      = "cli"
	entrypointSDKPy    = "sdk-py"
	entrypointSDKCLIRs = "sdk-cli"
	entrypointExec     = "exec"
)

// Scope classifies one sequence. The sub-agent test comes first because it is a fact about the
// sequence, while the entrypoint is a fact about the session holding it: a sub-agent launched
// from a cli session is still a sub-agent's work.
func Scope(t *store.Timeline) string {
	switch {
	case t.SubAgent():
		return SubAgent
	case t.Entrypoint == entrypointCLI:
		return Interactive
	case t.Entrypoint == entrypointSDKPy || t.Entrypoint == entrypointSDKCLIRs || t.Entrypoint == entrypointExec:
		return Programmatic
	case t.Entrypoint == "":
		return Unstated
	default:
		// An entrypoint this build has never seen is not silently promoted to a person's session.
		return Unstated
	}
}
