package openmetrics

import (
	"strings"
	"testing"
)

func parse(t *testing.T, body string) *Doc {
	t.Helper()
	doc, err := Parse(strings.NewReader(body), Limits{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return doc
}

func TestParseReadsTypesHelpAndUnits(t *testing.T) {
	doc := parse(t, `# HELP q Number of requests waiting.
# TYPE q gauge
# UNIT q requests
q{model="m"} 3
`)
	f := doc.Family("q")
	if f == nil {
		t.Fatal("family q missing")
	}
	if f.Type != TypeGauge || f.Unit != "requests" || f.Help != "Number of requests waiting." {
		t.Fatalf("family = %+v, want the declared type, unit and help", *f)
	}
	if len(f.Samples) != 1 || f.Samples[0].Value != 3 || f.Samples[0].Labels["model"] != "m" {
		t.Fatalf("samples = %+v", f.Samples)
	}
}

// TestParseNeverInfersAUnit: `_seconds` is a naming convention, not a promise. A unit assaio
// invented would travel as one the exporter stated.
func TestParseNeverInfersAUnit(t *testing.T) {
	doc := parse(t, "# TYPE latency_seconds gauge\nlatency_seconds 0.5\n")
	if u := doc.Family("latency_seconds").Unit; u != "" {
		t.Fatalf("Unit = %q, want empty: no UNIT line declared one", u)
	}
}

// TestParseKeepsAnUndeclaredTypeUntyped guards the decision every downstream reading turns on:
// whether a number may be read as a rate.
func TestParseKeepsAnUndeclaredTypeUntyped(t *testing.T) {
	doc := parse(t, "mystery 7\n")
	if got := doc.Family("mystery").Type; got != TypeUntyped {
		t.Fatalf("Type = %q, want %q", got, TypeUntyped)
	}
	doc = parse(t, "# TYPE odd stopwatch\nodd 7\n")
	if got := doc.Family("odd").Type; got != TypeUntyped {
		t.Fatalf("an unknown declared type became %q, want %q", got, TypeUntyped)
	}
}

func TestParseGroupsHistogramSuffixesOntoTheirFamily(t *testing.T) {
	doc := parse(t, `# TYPE ttft histogram
ttft_bucket{le="0.1"} 2
ttft_bucket{le="+Inf"} 5
ttft_sum 1.25
ttft_count 5
`)
	f := doc.Family("ttft")
	if f == nil || len(f.Samples) != 4 {
		t.Fatalf("histogram family = %+v, want its four samples grouped", f)
	}
	if doc.Family("ttft_bucket") != nil {
		t.Fatal("a _bucket sample created a family of its own")
	}
}

// TestParseKeepsAPlainCounterNamedCount separates a suffix from a coincidence: `foo_count` is
// its own metric when nothing declared a `foo`.
func TestParseKeepsAPlainCounterNamedCount(t *testing.T) {
	doc := parse(t, "# TYPE errors_count counter\nerrors_count 4\n")
	if doc.Family("errors_count") == nil {
		t.Fatal("errors_count was folded into a family nobody declared")
	}
}

func TestParseSkipsAndCountsACorruptLine(t *testing.T) {
	doc := parse(t, "# TYPE ok gauge\nok 1\nthis is not a sample\nok{bad label} 2\nok 3\n")
	if doc.Skipped != 2 {
		t.Fatalf("Skipped = %d, want 2", doc.Skipped)
	}
	if f := doc.Family("ok"); len(f.Samples) != 2 {
		t.Fatalf("samples = %+v, want the two readable ones kept", f.Samples)
	}
}

func TestParseReadsSpecialFloats(t *testing.T) {
	doc := parse(t, "a +Inf\nb NaN\nc -Inf\n")
	if len(doc.Families) != 3 || doc.Skipped != 0 {
		t.Fatalf("families = %d, skipped = %d; want 3 and 0", len(doc.Families), doc.Skipped)
	}
}

func TestParseAcceptsAndDropsATimestamp(t *testing.T) {
	doc := parse(t, "a 1 1699999999000\n")
	if doc.Skipped != 0 || doc.Family("a").Samples[0].Value != 1 {
		t.Fatalf("a timestamped sample was not read: skipped=%d", doc.Skipped)
	}
}

func TestParseHandlesACommaInsideALabelValue(t *testing.T) {
	doc := parse(t, `a{pod="x,y",gpu="0"} 1`+"\n")
	labels := doc.Family("a").Samples[0].Labels
	if labels["pod"] != "x,y" || labels["gpu"] != "0" {
		t.Fatalf("labels = %+v, want the comma kept inside the value", labels)
	}
}

// TestParseTruncatesRatherThanReadingUnbounded is the size limit: an oversized document stops
// and says so, because every absence in a truncated read is unproven.
func TestParseTruncatesRatherThanReadingUnbounded(t *testing.T) {
	var b strings.Builder
	for i := range 5000 {
		b.WriteString("m")
		b.WriteString(strings.Repeat("x", 20))
		b.WriteString(" 1\n")
		_ = i
	}
	doc, err := Parse(strings.NewReader(b.String()), Limits{MaxBytes: 512})
	if err != nil {
		t.Fatal(err)
	}
	if !doc.Truncated {
		t.Fatal("an oversized document was read without reporting truncation")
	}
}

func TestParseFailsALineOverTheLineLimit(t *testing.T) {
	long := "a{k=\"" + strings.Repeat("v", 5000) + "\"} 1\n"
	doc, err := Parse(strings.NewReader(long), Limits{MaxLineBytes: 64})
	if err == nil {
		t.Fatal("an over-long line parsed without error")
	}
	if !doc.Truncated {
		t.Fatal("an over-long line did not mark the document truncated")
	}
}

func TestParseRejectsAnOversizedLabelValue(t *testing.T) {
	doc := parse(t, `a{k="`+strings.Repeat("v", 600)+`"} 1`+"\n")
	if doc.Skipped != 1 {
		t.Fatalf("Skipped = %d, want the oversized label value rejected", doc.Skipped)
	}
}

func TestParseEmptyDocument(t *testing.T) {
	doc := parse(t, "")
	if len(doc.Families) != 0 || doc.Skipped != 0 || doc.Truncated {
		t.Fatalf("empty document = %+v", doc)
	}
}
