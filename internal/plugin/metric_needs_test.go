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

// TestAnUndeclaredSectionIsAbsentNotEmpty is the protocol's central refusal: a section the
// plugin did not ask for is not on the document at all. Sending it empty would be a window with
// nothing in it, which is a measurement -- and the one thing a plugin must never read out of an
// absence it did not cause.
func TestAnUndeclaredSectionIsAbsentNotEmpty(t *testing.T) {
	in := tracedInput()
	doc := documentOf(&in, granting(analyze.CapUsage))

	for _, key := range []string{"sessions", "trace", "skills", "agents", "turnSizing", "cacheMisses", "prices"} {
		if _, present := doc[key]; present {
			t.Errorf("%q is on the document of a plugin that declared only usage", key)
		}
	}
	if _, present := doc["usage"]; !present {
		t.Error("usage is absent from the document of a plugin that declared it")
	}
	if _, present := doc["withheld"]; present {
		t.Error("withheld names a capability nobody denied; it is only ever this install's own refusal")
	}
}

// TestConfigDenialIsNamedAsWithheld: the veto stays with the reader, and the plugin is told by
// name what it asked for and did not get. Without the naming its verdict over an absent section
// is indistinguishable from one over a window that holds none.
func TestConfigDenialIsNamedAsWithheld(t *testing.T) {
	in := tracedInput()
	declared := Declaration{Needs: []analyze.Capability{analyze.CapUsage, analyze.CapTrace}}

	p := negotiate(declared, []analyze.Capability{analyze.CapUsage})
	doc := documentOf(&in, p)

	if _, present := doc["trace"]; present {
		t.Error("trace is on the document of a plugin whose config denies it")
	}
	withheld, _ := doc["withheld"].([]analyze.Capability)
	if len(withheld) != 1 || withheld[0] != analyze.CapTrace {
		t.Fatalf("withheld = %v, want [%s]", doc["withheld"], analyze.CapTrace)
	}
	if got := deniedCaveat("quiet", withheld); !strings.Contains(got, "trace") || !strings.Contains(got, "quiet") {
		t.Fatalf("caveat = %q, want it to name both the plugin and the capability", got)
	}
}

// TestConfigAllowingMoreGrantsOnlyWhatWasDeclared: config constrains, it never defines. A
// reader who permits everything still gets the plugin's own reading list, so widening the veto
// cannot silently widen what a subprocess is handed.
func TestConfigAllowingMoreGrantsOnlyWhatWasDeclared(t *testing.T) {
	p := negotiate(Declaration{Needs: []analyze.Capability{analyze.CapUsage}}, analyze.Capabilities())
	if len(p.Needs) != 1 || p.Needs[0] != analyze.CapUsage {
		t.Fatalf("granted %v, want only the declared usage", p.Needs)
	}
	if len(p.Withheld) != 0 {
		t.Fatalf("withheld = %v for a plugin that got everything it asked for", p.Withheld)
	}
}

// TestNoAllowListIsNoConstraint: almost no `metrics:` entry carries the key, and reading that
// silence as a denial would starve every plugin the moment this shipped.
func TestNoAllowListIsNoConstraint(t *testing.T) {
	declared := Declaration{Needs: []analyze.Capability{analyze.CapUsage, analyze.CapTrace}}
	p := negotiate(declared, nil)
	if len(p.Needs) != 2 || len(p.Withheld) != 0 {
		t.Fatalf("negotiate with no allow list granted %v and withheld %v", p.Needs, p.Withheld)
	}
}

// A denied capability's columns and predicates go with it: echoing a projection over a section
// the document does not carry is a contradiction the plugin would have to resolve by guessing.
func TestADeniedSectionTakesItsProjectionWithIt(t *testing.T) {
	declared := Declaration{
		Needs:  []analyze.Capability{analyze.CapUsage, analyze.CapTrace},
		Fields: map[string][]string{"usage": {"day"}, "trace": {"scope"}},
		Where:  map[string][]string{"trace.scope": {"interactive"}},
	}
	p := negotiate(declared, []analyze.Capability{analyze.CapUsage})
	if _, echoed := p.Fields["trace"]; echoed {
		t.Errorf("fields still projects a denied section: %v", p.Fields)
	}
	if len(p.Where) != 0 {
		t.Errorf("where still filters a denied section: %v", p.Where)
	}
	if _, kept := p.Fields["usage"]; !kept {
		t.Errorf("fields dropped the granted section's own projection: %v", p.Fields)
	}
}

// TestAnUnknownNeedIsRejectedAtTheConfigBoundary keeps the capability set closed: a typo in the
// reader's veto must fail loudly rather than silently deny a section the plugin asked for.
func TestAnUnknownNeedIsRejectedAtTheConfigBoundary(t *testing.T) {
	_, err := ResolveMetric(pluginNeeding("tracce"))
	if err == nil || !strings.Contains(err.Error(), "unknown need") {
		t.Fatalf("Resolve error = %v, want it to reject an unknown need", err)
	}
}
