// Package runtime inspects a self-hosted inference deployment's own telemetry endpoints -- one
// snapshot, read-only, nothing stored. It is the feasibility slice of Runtime Insights and is
// marked experimental: it exists to find out whether combining runtime evidence with assaio's
// usage evidence changes a decision anybody actually makes, not to become a GPU dashboard.
//
// It is a separate evidence plane from the coding-session parsers and shares no code with
// them. A hosted Claude, Codex or Gemini session runs on the vendor's accelerators, which this
// cannot see and must never claim to: only infrastructure the reader operates is observable
// here.
package runtime

import "time"

// Availability is what one snapshot could establish about a source. Unreachable and empty are
// deliberately different answers: one says assaio could not look, the other that it looked and
// the exporter published nothing.
const (
	// AvailabilityServing means the endpoint answered and published metrics.
	AvailabilityServing = "serving"
	// AvailabilityEmpty means it answered with no metric this adapter recognizes.
	AvailabilityEmpty = "empty"
	// AvailabilityUnreachable means assaio could not read it at all.
	AvailabilityUnreachable = "unreachable"
)

// Kind is how a family may be read. It is copied from the exposition's own `# TYPE` and never
// inferred, because it decides the one thing a single snapshot cannot do.
const (
	KindGauge     = "gauge"
	KindCounter   = "counter"
	KindHistogram = "histogram"
	KindUnknown   = "unknown"
)

// Capability is one metric family an adapter knows how to look for, kept as data so the
// catalog is readable as a list of what assaio can and cannot see.
type Capability struct {
	// Key is assaio's name for the thing being measured.
	Key string `json:"key"`
	// Metric is the exporter's own metric name, kept verbatim: it is the only name the
	// vendor's documentation is about, and the only one a reader can grep for.
	Metric string `json:"metric"`
	// Unit is what this quantity is measured in, and Unit source says who said so -- the
	// exporter's own `# UNIT` line, or assaio reading the vendor's documentation. A unit
	// nobody stated is empty rather than guessed.
	Unit       string `json:"unit,omitempty"`
	UnitSource string `json:"unitSource,omitempty"`
	// Summary is what the number means, in one line.
	Summary string `json:"summary"`
}

// Finding is what a snapshot found for one capability.
type Finding struct {
	Capability
	// Present reports whether the exporter published this family at all. False means
	// unavailable -- never zero, which is a value somebody measured.
	Present bool `json:"present"`
	// Kind is the type the document declared for it.
	Kind string `json:"kind,omitempty"`
	// Readings are the values, one per label set. Populated for gauges and for a histogram's
	// count; a counter's cumulative value is here too, and Note says what it is not.
	Readings []Reading `json:"readings,omitempty"`
	// Note carries the limit that travels with this reading, above all the one a single
	// snapshot imposes on a counter.
	Note string `json:"note,omitempty"`
}

// Reading is one value with the labels identifying what it is about.
type Reading struct {
	Labels map[string]string `json:"labels,omitempty"`
	Value  float64           `json:"value"`
}

// Snapshot is everything one read of one source established.
type Snapshot struct {
	// Source is the adapter that read it: "vllm", "dcgm".
	Source string `json:"source"`
	// Origin is where it was read from -- a file path or an endpoint URL -- so a reader can
	// tell a deterministic replay from a live read.
	Origin string `json:"origin"`
	// ObservedAt is when assaio read it, not when the exporter measured it. No exporter states
	// the second, and presenting the first as if it were would be a claim nobody made.
	ObservedAt time.Time `json:"observedAt"`
	// Availability is one of the constants above.
	Availability string `json:"availability"`
	// Error is why an unreachable source was unreachable, in the terms the reader can act on.
	Error string `json:"error,omitempty"`
	// SkippedLines is how many lines this read could not parse, and Truncated whether a limit
	// stopped it. Either one makes every absence below unproven, which Limits says out loud.
	SkippedLines int  `json:"skippedLines"`
	Truncated    bool `json:"truncated"`
	// Findings is one entry per capability in the adapter's catalog, present or not, so the
	// output is the same shape whatever the deployment exposes.
	Findings []Finding `json:"findings"`
}

// Missing returns the capabilities this snapshot did not find, which is the half of the answer
// a reader has to see: an inspection that only listed what it found would read as complete.
func (s *Snapshot) Missing() []Finding {
	var out []Finding
	for i := range s.Findings {
		if !s.Findings[i].Present {
			out = append(out, s.Findings[i])
		}
	}
	return out
}

// Present returns the capabilities this snapshot did find.
func (s *Snapshot) Present() []Finding {
	var out []Finding
	for i := range s.Findings {
		if s.Findings[i].Present {
			out = append(out, s.Findings[i])
		}
	}
	return out
}
