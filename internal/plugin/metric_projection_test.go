package plugin

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/trace"
)

// The envelope struct is the protocol's published vocabulary and the document is the projection
// of it, so a field added to one and forgotten in the other reaches every plugin as an absence
// -- which this protocol says is never a zero, and which nothing else would catch.
func TestEveryEnvelopeFieldReachesTheDocument(t *testing.T) {
	in := tracedInput()
	full := everything()
	full.Withheld = []analyze.Capability{analyze.CapTrace}
	doc := documentOf(&in, full)

	envelope := reflect.TypeOf(metricInput{})
	for i := range envelope.NumField() {
		key, _, _ := strings.Cut(envelope.Field(i).Tag.Get("json"), ",")
		if _, present := doc[key]; !present {
			t.Errorf("metricInput.%s is on the envelope but %q never reaches the document",
				envelope.Field(i).Name, key)
		}
	}
}

func TestProjectedColumnsAreTheOnlyOnesSent(t *testing.T) {
	in := tracedInput()
	p := granting(analyze.CapUsage)
	p.Fields = map[string][]string{"usage": {"day", "in"}}

	rows, _ := documentOf(&in, p)["usage"].([]map[string]any)
	if len(rows) != 1 {
		t.Fatalf("usage carried %d rows, want 1", len(rows))
	}
	if got := slices.Sorted(maps(rows[0])); !slices.Equal(got, []string{"day", "in"}) {
		t.Fatalf("usage row carries %v, want only the projected columns", got)
	}
}

// A section nobody projected keeps every column: the narrowing is the plugin's declaration, and
// a core that trimmed a column on its own would hand out an absence the plugin cannot explain.
func TestAnUnprojectedSectionKeepsEveryColumn(t *testing.T) {
	in := tracedInput()
	doc := documentOf(&in, granting(analyze.CapUsage))
	if _, projected := doc["usage"].([]map[string]any); projected {
		t.Fatal("usage was rendered as projected documents with no projection declared")
	}
}

// trace[].steps is where the bytes are -- 95% of the largest section -- so a step column list
// has to reach the steps rather than stopping at the sequence around them.
func TestANestedProjectionNarrowsTheSteps(t *testing.T) {
	in := tracedInput()
	p := granting(analyze.CapTrace)
	p.Fields = map[string][]string{"trace": {"sessionId", "steps"}, "trace.steps": {"kind"}}

	sequences, _ := documentOf(&in, p)["trace"].([]map[string]any)
	if len(sequences) != 1 {
		t.Fatalf("trace carried %d sequences, want 1", len(sequences))
	}
	if got := slices.Sorted(maps(sequences[0])); !slices.Equal(got, []string{"sessionId", "steps"}) {
		t.Fatalf("sequence carries %v, want only the projected columns", got)
	}
	steps, _ := sequences[0]["steps"].([]map[string]any)
	if len(steps) != 1 {
		t.Fatalf("sequence carried %d steps, want 1", len(steps))
	}
	if got := slices.Sorted(maps(steps[0])); !slices.Equal(got, []string{"kind"}) {
		t.Fatalf("step carries %v, want only the projected column", got)
	}
}

// A predicate is the plugin's own denominator choice, so the envelope states what it excluded.
// Without the available count a pushed-down filter is a new way to publish "all of them are X"
// over a set the plugin picked -- the fabricated denominator this boundary refuses everywhere
// else.
func TestAPredicateDropsRowsAndStatesTheDenominator(t *testing.T) {
	in := tracedInput()
	in.Usage = append(in.Usage, store.UsageRow{Day: "2026-07-16", Tool: "codex", Model: "m", In: 5})
	p := granting(analyze.CapUsage)
	p.Where = map[string][]string{"usage.tool": {"claude-code"}}

	doc := documentOf(&in, p)
	rows := reflect.ValueOf(doc["usage"])
	if rows.Len() != 1 {
		t.Fatalf("usage carried %d rows, want only the tool the predicate admits", rows.Len())
	}
	counts := doc["projection"].(metricProjection).Rows["usage"]
	if counts != (rowCount{Sent: 1, Available: 2}) {
		t.Fatalf("projection.rows[usage] = %+v, want 1 of 2", counts)
	}
}

// The document has to say what it is: which capabilities were granted, and what narrowed them.
// A projected document that did not would be indistinguishable from a truncated one.
func TestProjectionIsEchoedToThePlugin(t *testing.T) {
	in := tracedInput()
	p := granting(analyze.CapUsage)
	p.Fields = map[string][]string{"usage": {"day"}}
	p.Where = map[string][]string{"usage.tool": {"claude-code"}}

	echoed := documentOf(&in, p)["projection"].(metricProjection)
	if !slices.Equal(echoed.Needs, []analyze.Capability{analyze.CapUsage}) {
		t.Errorf("projection.needs = %v", echoed.Needs)
	}
	if !slices.Equal(echoed.Fields["usage"], []string{"day"}) {
		t.Errorf("projection.fields = %v", echoed.Fields)
	}
	if !slices.Equal(echoed.Where["usage.tool"], []string{"claude-code"}) {
		t.Errorf("projection.where = %v", echoed.Where)
	}
}

// The reason the declaration exists at all: a metric that reads three columns of one section no
// longer pays for the whole window.
func TestProjectionShrinksTheEnvelope(t *testing.T) {
	in := tracedInput()
	in.Trace = trace.New(append(in.Trace.All(), storeTimelines(64)...))

	full := envelopeBytes(t, &in, everything())
	narrow := granting(analyze.CapUsage)
	narrow.Fields = map[string][]string{"usage": {"day", "in", "out"}}
	projected := envelopeBytes(t, &in, narrow)
	if len(projected) >= len(full) {
		t.Fatalf("projecting did not shrink the envelope: %d vs %d bytes", len(projected), len(full))
	}
}

// storeTimelines builds n one-step sequences, enough that the trace section dominates the
// envelope the way it does on a real store.
func storeTimelines(n int) []store.Timeline {
	out := make([]store.Timeline, n)
	for i := range out {
		out[i] = store.Timeline{
			SessionID: "s", Tool: "claude-code", Timeline: "programmatic",
			Steps: []store.TimelineStep{{Ordinal: 1, Kind: "assistant", Tokens: 30}},
		}
	}
	return out
}

func maps(m map[string]any) func(func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}
