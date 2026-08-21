package analyze

import (
	"slices"
	"strings"
)

// Capability names one kind of evidence an analyzer reads. It exists because Input has been
// the only way to say so: eight concrete store row types on one struct, every new evidence
// source pushing a ninth onto it, and every analyzer free to reach for anything (B102). A
// capability is the narrow version of that question -- what does this metric actually need --
// and it is the same vocabulary on both sides of the exec boundary, so a metric plugin can
// declare its needs in the handshake and be sent only those (ADR 0011, B168).
//
// The set is closed on purpose. An analyzer that needs something not on this list needs a new
// capability, reviewed, rather than a field quietly added to a struct nobody owns.
type Capability string

const (
	// CapUsage is the window's daily usage rows: tokens, cost basis, lines, activity counts.
	CapUsage Capability = "usage"
	// CapSessions is the window's per-session records: turns, peak context, active minutes.
	CapSessions Capability = "sessions"
	// CapTrace is the stored step sequences -- what each session did, in what order (ADR 0012).
	// It is by far the largest payload, which is why declaring it matters.
	CapTrace Capability = "trace"
	// CapAttribution is per-skill and per-sub-agent token totals.
	CapAttribution Capability = "attribution"
	// CapTurnSizing is per-model raw turn counts at the grain the daily aggregate hides.
	CapTurnSizing Capability = "turn-sizing"
	// CapCacheMisses is the stated cache-miss reasons per tool.
	CapCacheMisses Capability = "cache-misses"
	// CapPrices is the price table and the plan cost the window is compared against.
	CapPrices Capability = "prices"
)

// capabilities is every capability in the order a reader should meet them, and the set an
// unknown name is rejected against.
var capabilities = []Capability{
	CapUsage, CapSessions, CapTrace, CapAttribution, CapTurnSizing, CapCacheMisses, CapPrices,
}

// Capabilities returns the closed set, for the registries and the wire that publish it.
func Capabilities() []Capability { return slices.Clone(capabilities) }

// ValidCapability reports whether c is one this build understands. The boundary rejects what
// it cannot interpret rather than passing it through as a name nobody defined.
func ValidCapability(c Capability) bool { return slices.Contains(capabilities, c) }

// Needs is implemented by a Validator that declares the evidence it reads. A validator that
// does not implement it declares nothing, and nothing is what it is held to: it runs on every
// window exactly as it did before, owning its own empty case. That is what lets this migrate a
// family at a time instead of as a flag day -- declaring is what buys the withheld reason and
// the smaller plugin payload, and not declaring costs nothing but those.
//
// "Needs everything" is deliberately not the default. It would withhold every undeclared
// validator the moment any capability were absent, which on a store with no sub-agent
// attribution is most of them.
type Needs interface {
	Needs() []Capability
}

// NeededBy returns what v declared, or nil when it declared nothing.
func NeededBy(v Validator) []Capability {
	n, ok := v.(Needs)
	if !ok {
		return nil
	}
	return slices.Clone(n.Needs())
}

// Missing returns the capabilities v needs that in cannot answer, in declaration order. It is
// the difference between "this metric found nothing" and "this window could not be asked" --
// two sentences a reader acts on differently, and which a validator producing its own empty
// result cannot tell apart.
func Missing(v Validator, in *Input) []Capability {
	var missing []Capability
	for _, c := range NeededBy(v) {
		if !in.Has(c) {
			missing = append(missing, c)
		}
	}
	return missing
}

// Has reports whether this Input carries anything for c. Emptiness is the whole test: the
// runner cannot tell a capability the caller chose not to load from one whose window is
// genuinely empty, and it must not, because both mean the same thing to an analyzer -- there
// is nothing here to read.
func (in *Input) Has(c Capability) bool {
	switch c {
	case CapUsage:
		return len(in.Usage) > 0
	case CapSessions:
		return len(in.Sessions) > 0
	case CapTrace:
		return !in.Trace.Empty()
	case CapAttribution:
		return len(in.Skills) > 0 || len(in.Agents) > 0
	case CapTurnSizing:
		return len(in.TurnSizing) > 0
	case CapCacheMisses:
		return len(in.CacheMisses) > 0
	case CapPrices:
		return len(in.Prices) > 0
	default:
		return false
	}
}

// withheldFor records which declared inputs this window could not supply. It does not touch the
// verdict: a validator owns its own empty case, and several say something the runner could not
// -- skill-economics separates "nothing ran" from "what ran carried no labels", which is a
// better sentence than any generic one. What the runner adds is the machine-readable reason,
// so every surface can state the same thing and a reader can tell a metric that measured
// nothing from one that was never asked.
func withheldFor(r *Result, missing []Capability) {
	if len(missing) == 0 {
		return
	}
	r.Withheld = missing
	r.Caveats = append(r.Caveats, withheldCaveat(missing))
}

// withheldCaveat names the declared inputs the window did not carry. It names the capability
// rather than describing the metric, because the fix is about the data: a window with no step
// timeline wants a `backfill`, one with no attributed turns wants sessions that used a skill or
// a sub-agent.
func withheldCaveat(missing []Capability) string {
	names := make([]string, len(missing))
	for i, c := range missing {
		names[i] = string(c)
	}
	return "Withheld input: this metric declares that it reads " + strings.Join(names, ", ") +
		", and this window carries none of it. Whatever is above rests on less than the metric asks for."
}
