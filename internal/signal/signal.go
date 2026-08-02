// Package signal is the catalog of what assaio can report and what each thing means. A
// source-depth matrix answers "what can this tool tell me"; this answers the question people
// actually have, "can it tell me *this*" -- and, through Coverage, whether it can tell them
// on their own machine. See docs/adr/0008-signal-catalog.md.
package signal

// How a signal's value came to be. "attributed" is named in ADR 0008 and lands with
// attribution edges (B85), which is what will first produce a signal deserving it.
const (
	// Observed is read from a source artifact and is a fact about what happened.
	Observed = "observed"
	// Estimated is modelled, so it carries an error bar somewhere in its description.
	Estimated = "estimated"
	// Derived is computed by assaio from other signals.
	Derived = "derived"
)

// The grains a signal is meaningful at. A signal read at a grain it does not support is the
// mistake this field exists to prevent -- per-turn efficiency over session-total records
// says nothing.
const (
	GrainTurn    = "turn"
	GrainSession = "session"
	GrainDay     = "day"
	GrainWindow  = "window"
)

// Signal is one declared thing assaio can report. It declares what the number *means*;
// which sources can produce it is declared by those sources (parser.Depth.Answers), so
// neither can drift into being a second opinion about the other.
type Signal struct {
	// ID is stable and public from the moment it is printed: <domain>.<thing>.<measure>,
	// the same shape the event contract uses (ADR 0007).
	ID    string
	Title string
	// Unit names what the number counts, so nobody reads tokens as dollars.
	Unit string
	// Status is Observed, Estimated or Derived.
	Status string
	// Grains lists where reading it is honest.
	Grains []string
	// ZeroMeans is why this exists: "no rework happened" and "this source never recorded
	// rework" are different facts, and a metric that confuses them lies confidently.
	ZeroMeans string
	// Describe is one line of what it is, including any caveat that travels with it.
	Describe string
}
