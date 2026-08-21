package plugin

import (
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/trace"
)

// tracedInput is a window carrying one step sequence, so "did the timeline travel" is a
// question about the declaration rather than about the fixture.
func tracedInput() analyze.Input {
	now := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	usage := []store.UsageRow{{Day: "2026-07-16", Tool: "claude-code", Model: "m", In: 10, Out: 20}}
	in := analyze.BuildInput(usage, nil, pricing.Table{}, now, 7*24*time.Hour, analyze.Delegation{})
	in.Trace = trace.New([]store.Timeline{{
		SessionID: "s1", Tool: "claude-code", Timeline: "programmatic",
		Steps: []store.TimelineStep{{Kind: "assistant", Tokens: 30}},
	}})
	return in
}

// TestUndeclaredTraceIsWithheldNotSentEmpty is B168: the step timeline was measured at ~44 MB
// on a real store and every metric plugin received it whether or not it read one. It is now
// declared in config -- and a plugin that did not declare it is *told* the section is absent,
// because an empty array it cannot tell apart from a window with no sequences is the
// fabricated zero every other boundary here refuses.
func TestUndeclaredTraceIsWithheldNotSentEmpty(t *testing.T) {
	in := tracedInput()

	silent := buildMetricInput(&in, Config{Name: "quiet"})
	if len(silent.Trace) != 0 {
		t.Fatalf("Trace carried %d sequence(s) to a plugin that declared none", len(silent.Trace))
	}
	if len(silent.Withheld) != 1 || silent.Withheld[0] != analyze.CapTrace {
		t.Fatalf("Withheld = %v, want [%s] so the plugin can tell absent from empty", silent.Withheld, analyze.CapTrace)
	}

	asked := buildMetricInput(&in, tracingPlugin())
	if len(asked.Trace) != 1 {
		t.Fatalf("Trace carried %d sequence(s) to a plugin that declared it, want 1", len(asked.Trace))
	}
	if len(asked.Withheld) != 0 {
		t.Fatalf("Withheld = %v for a plugin that got everything it asked for", asked.Withheld)
	}
}

// TestUndeclaredTraceShrinksThePayload is the reason the declaration exists at all.
func TestUndeclaredTraceShrinksThePayload(t *testing.T) {
	in := tracedInput()
	quiet := buildMetricInput(&in, Config{Name: "quiet"})
	silent, err := quiet.marshal()
	if err != nil {
		t.Fatal(err)
	}
	loud := buildMetricInput(&in, tracingPlugin())
	asked, err := loud.marshal()
	if err != nil {
		t.Fatal(err)
	}
	if len(silent) >= len(asked) {
		t.Fatalf("withholding the timeline did not shrink the envelope: %d vs %d bytes", len(silent), len(asked))
	}
}

// TestAnUnknownNeedIsRejectedAtTheConfigBoundary keeps the capability set closed: a typo must
// fail loudly rather than silently withhold a section the plugin thought it had asked for.
func TestAnUnknownNeedIsRejectedAtTheConfigBoundary(t *testing.T) {
	_, err := ResolveMetric(pluginNeeding("tracce"))
	if err == nil || !strings.Contains(err.Error(), "unknown need") {
		t.Fatalf("Resolve error = %v, want it to reject an unknown need", err)
	}
}
