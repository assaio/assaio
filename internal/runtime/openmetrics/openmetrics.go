// Package openmetrics parses the Prometheus/OpenMetrics text exposition format into families
// and samples. It exists so the vLLM and DCGM adapters do not each grow their own text
// scanner; it deliberately knows nothing about either, and nothing about coding sessions --
// runtime telemetry and session logs are separate evidence planes and share no parser.
//
// It reads a snapshot and returns what that snapshot said. It never infers a rate, never fills
// an absent metric with zero, and never rewrites a vendor's metric name: an adapter compares
// against the original name, because that is the only thing a vendor's own documentation is
// about.
package openmetrics

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// A sample's kind, as the exposition's own `# TYPE` line states it. Untyped is what a document
// that never declared one leaves behind, and it is reported as untyped rather than guessed:
// whether a number is a counter decides whether reading it as a rate is even possible.
const (
	TypeGauge     = "gauge"
	TypeCounter   = "counter"
	TypeHistogram = "histogram"
	TypeSummary   = "summary"
	TypeUntyped   = "untyped"
)

// Sample is one exposed value with the labels that identify it.
type Sample struct {
	// Name is the sample's own name, which for a histogram carries the _bucket, _sum or
	// _count suffix its family does not.
	Name   string
	Labels map[string]string
	Value  float64
}

// Family is one metric name with everything the document said about it.
type Family struct {
	Name string
	Type string
	Help string
	// Unit is what `# UNIT` declared, empty when the document declared none. It is never
	// inferred from the name: `_seconds` is a convention, not a promise, and a unit assaio
	// invented would travel as one the exporter stated.
	Unit    string
	Samples []Sample
}

// Doc is a parsed exposition.
type Doc struct {
	// Families in the order the document first mentioned each, so rendering is stable.
	Families []Family
	// Skipped counts lines this parser could not read. Reported rather than silently dropped:
	// a snapshot that half-parsed is partial evidence, and partial evidence has to say so.
	Skipped int
	// Truncated reports that a limit stopped the read before the input ended, which makes
	// every absence in this document unproven.
	Truncated bool
}

// Family returns the named family, or nil.
func (d *Doc) Family(name string) *Family {
	for i := range d.Families {
		if d.Families[i].Name == name {
			return &d.Families[i]
		}
	}
	return nil
}

// Parse reads an exposition from r under the given limits. It returns what it could read plus
// the count it could not, following the same skip-and-count policy the session parsers use: a
// malformed line loses its own sample and nothing else.
func Parse(r io.Reader, lim Limits) (*Doc, error) {
	lim = lim.orDefaults()
	doc := &Doc{}
	byName := map[string]int{}
	sc := bufio.NewScanner(&io.LimitedReader{R: r, N: lim.MaxBytes + 1})
	// bufio takes the larger of the cap and the max, so the starting buffer must not exceed
	// the limit -- sizing it at a comfortable 64 KiB would silently raise a smaller bound.
	sc.Buffer(make([]byte, 0, min(64*1024, lim.MaxLineBytes)), lim.MaxLineBytes)

	read := int64(0)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		read += int64(len(sc.Bytes())) + 1
		if read > lim.MaxBytes {
			doc.Truncated = true
			break
		}
		switch {
		case line == "" || line == "# EOF":
			continue
		case strings.HasPrefix(line, "#"):
			if !applyMeta(doc, byName, line) {
				doc.Skipped++
			}
		default:
			if !addSample(doc, byName, line, lim) {
				doc.Skipped++
			}
		}
		if len(doc.Families) >= lim.MaxFamilies {
			doc.Truncated = true
			break
		}
	}
	if err := sc.Err(); err != nil {
		if strings.Contains(err.Error(), "token too long") {
			doc.Truncated = true
			return doc, fmt.Errorf("line longer than the %d-byte limit: %w", lim.MaxLineBytes, err)
		}
		return doc, err
	}
	return doc, nil
}

// applyMeta handles a HELP, TYPE or UNIT line, reporting whether it was one this parser
// understands. A comment that is none of the three is a comment, not a failure.
func applyMeta(doc *Doc, byName map[string]int, line string) bool {
	fields := strings.SplitN(strings.TrimSpace(strings.TrimPrefix(line, "#")), " ", 3)
	if len(fields) < 2 {
		return true // a bare comment
	}
	keyword, name := fields[0], fields[1]
	var value string
	if len(fields) == 3 {
		value = fields[2]
	}
	switch keyword {
	case "HELP", "TYPE", "UNIT":
	default:
		return true // a comment that merely starts with a word
	}
	if !validName(name) {
		return false
	}
	f := family(doc, byName, name)
	switch keyword {
	case "HELP":
		f.Help = value
	case "TYPE":
		f.Type = declaredType(value)
	case "UNIT":
		f.Unit = value
	}
	return true
}

// declaredType keeps the vocabulary closed: a type this parser does not know is recorded as
// untyped rather than passed through, because every downstream decision -- above all "may this
// be read as a rate" -- turns on it.
func declaredType(v string) string {
	switch v {
	case TypeGauge, TypeCounter, TypeHistogram, TypeSummary:
		return v
	default:
		return TypeUntyped
	}
}

// addSample parses one `name{labels} value [timestamp]` line, reporting whether it was
// readable. A timestamp is accepted and dropped: this is a snapshot reader, and keeping a
// per-sample time would invite exactly the rate arithmetic one snapshot cannot support.
func addSample(doc *Doc, byName map[string]int, line string, lim Limits) bool {
	name, labels, rest, ok := splitSample(line, lim)
	if !ok {
		return false
	}
	valueField := strings.Fields(rest)
	if len(valueField) == 0 || len(valueField) > 2 {
		return false
	}
	value, err := strconv.ParseFloat(valueField[0], 64)
	if err != nil {
		return false
	}
	f := family(doc, byName, baseName(name, doc, byName))
	if len(f.Samples) >= lim.MaxSamplesPerFamily {
		doc.Truncated = true
		return true
	}
	f.Samples = append(f.Samples, Sample{Name: name, Labels: labels, Value: value})
	return true
}

// baseName maps a histogram or summary sample onto the family that declared it, so
// `x_bucket` joins `x` rather than inventing a family of its own. A suffix is only stripped
// when a family of the base name was actually declared: a plain counter named `foo_count` is
// its own metric, not a summary nobody wrote.
func baseName(name string, doc *Doc, byName map[string]int) string {
	for _, suffix := range []string{"_bucket", "_sum", "_count", "_created", "_total"} {
		base, found := strings.CutSuffix(name, suffix)
		if !found {
			continue
		}
		if i, ok := byName[base]; ok {
			switch doc.Families[i].Type {
			case TypeHistogram, TypeSummary, TypeCounter:
				return base
			}
		}
	}
	return name
}

// family returns the named family, creating it as untyped when the document has not declared
// it yet -- an exporter may emit a sample before its TYPE line, and a later TYPE fills it in.
func family(doc *Doc, byName map[string]int, name string) *Family {
	if i, ok := byName[name]; ok {
		return &doc.Families[i]
	}
	doc.Families = append(doc.Families, Family{Name: name, Type: TypeUntyped})
	byName[name] = len(doc.Families) - 1
	return &doc.Families[len(doc.Families)-1]
}
