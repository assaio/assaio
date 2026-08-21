package runtime

import (
	"time"

	"github.com/assaio/assaio/internal/runtime/openmetrics"
)

// counterNote is what travels with every counter this package reports. A counter is cumulative
// since the exporter started; turning it into a rate needs two reads and the time between
// them, and this command deliberately takes one. Printing "12,481 requests" beside a per-second
// figure the reader assumed is the single easiest way to make this feature lie.
const counterNote = "cumulative since the exporter started -- not a rate. One snapshot cannot produce one; two reads and the interval between them can."

// histogramNote states what a histogram's count is and is not.
const histogramNote = "observation count, cumulative since start. The distribution's buckets are not summarized here: a percentile read off one snapshot describes all of history, not now."

// Inspect maps a parsed exposition onto one adapter's catalog. Every capability in the catalog
// appears in the result whether or not the document carried it, because "assaio looked and it
// was not there" is the answer a reader needs and an output listing only what it found cannot
// give.
func Inspect(source, origin string, doc *openmetrics.Doc, catalog []Capability, observedAt time.Time) Snapshot {
	s := Snapshot{
		Source: source, Origin: origin, ObservedAt: observedAt,
		SkippedLines: doc.Skipped, Truncated: doc.Truncated,
		Findings: make([]Finding, 0, len(catalog)),
	}
	found := 0
	for _, c := range catalog {
		f := Finding{Capability: c}
		if fam := doc.Family(c.Metric); fam != nil {
			found++
			f.Present = true
			f.Kind = kindOf(fam.Type)
			f.Unit, f.UnitSource = unitOf(&c, fam)
			f.Readings, f.Note = readingsFor(fam)
		}
		s.Findings = append(s.Findings, f)
	}
	s.Availability = AvailabilityEmpty
	if found > 0 {
		s.Availability = AvailabilityServing
	}
	return s
}

// Unreachable is the snapshot for a source assaio could not read. It carries the same shape as
// a successful one, with every capability unknown rather than absent: a failed read establishes
// nothing about what the deployment exposes.
func Unreachable(source, origin, reason string, catalog []Capability, observedAt time.Time) Snapshot {
	s := Snapshot{
		Source: source, Origin: origin, ObservedAt: observedAt,
		Availability: AvailabilityUnreachable, Error: reason,
		Findings: make([]Finding, 0, len(catalog)),
	}
	for _, c := range catalog {
		s.Findings = append(s.Findings, Finding{Capability: c, Note: "not established: the source could not be read"})
	}
	return s
}

// unitOf prefers the exporter's own declared unit over the one assaio carries in its catalog,
// and says which it used. The catalog's unit is read off vendor documentation, which can go
// stale; a `# UNIT` line cannot.
func unitOf(c *Capability, fam *openmetrics.Family) (unit, source string) {
	if fam.Unit != "" {
		return fam.Unit, "the exporter's own UNIT line"
	}
	if c.Unit != "" {
		return c.Unit, "assaio's catalog, from the vendor's documentation"
	}
	return "", ""
}

func kindOf(t string) string {
	switch t {
	case openmetrics.TypeGauge:
		return KindGauge
	case openmetrics.TypeCounter:
		return KindCounter
	case openmetrics.TypeHistogram, openmetrics.TypeSummary:
		return KindHistogram
	default:
		return KindUnknown
	}
}

// readingsFor extracts the values worth printing for a family, and the caveat that has to
// travel with them. A histogram contributes only its _count: a percentile computed from one
// snapshot's buckets describes the whole life of the process, which is not what anybody reading
// a latency figure means.
func readingsFor(fam *openmetrics.Family) (readings []Reading, note string) {
	switch kindOf(fam.Type) {
	case KindGauge:
		return samplesOf(fam, ""), ""
	case KindCounter:
		return samplesOf(fam, "_total"), counterNote
	case KindHistogram:
		return samplesOf(fam, "_count"), histogramNote
	default:
		return samplesOf(fam, ""), "the exposition declared no type for this metric, so how it may be read is unknown"
	}
}

// samplesOf keeps the samples whose name carries the wanted suffix, falling back to the family
// name itself -- an exporter may write a counter as `x` or as `x_total`, and both are the same
// number.
func samplesOf(fam *openmetrics.Family, suffix string) []Reading {
	var out []Reading
	for _, s := range fam.Samples {
		if s.Name != fam.Name && s.Name != fam.Name+suffix {
			continue
		}
		out = append(out, Reading{Labels: s.Labels, Value: s.Value})
	}
	return out
}
