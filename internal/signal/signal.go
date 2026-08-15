// Package signal is the catalog of what assaio can report and what each thing means. A
// source-depth matrix answers "what can this tool tell me"; this answers the question people
// actually have, "can it tell me *this*" -- and, through Coverage, whether it can tell them
// on their own machine. See docs/adr/0008-signal-catalog.md.
package signal

import "github.com/assaio/assaio/internal/layer"

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
	// GrainStep is one observation inside a session's sequence -- a model turn or a single tool
	// call. Finer than GrainTurn: a turn holds the calls it made, and a figure about tool calls
	// read at turn grain cannot say what order they came in (ADR 0012).
	GrainStep    = "step"
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
	// Layer is which of the four measurement layers this signal sits on. It is on every entry
	// because relabeling one is how a product like this starts lying, and the mistake is easy
	// to make from an ID alone: ai.step.outcome is an *activity* signal -- how one call ended
	// inside a session -- not an outcome one, which is whether the code held (ROADMAP.md,
	// "Four layers, never relabeled"; ADR 0013).
	Layer layer.Layer
	// Grains lists where reading it is honest.
	Grains []string
	// ZeroMeans is why this exists: "no rework happened" and "this source never recorded
	// rework" are different facts, and a metric that confuses them lies confidently.
	ZeroMeans string
	// Describe is one line of what it is, including any caveat that travels with it.
	Describe string
}
