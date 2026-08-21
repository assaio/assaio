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

// pluginRawMember is a name no pseudonym can contain, so the assertion is a substring
// search over the whole payload rather than a field-by-field walk that a new field escapes.
const pluginRawMember = "zzq-carol-example"

// TestMetricPayloadCarriesNoRawMember holds the boundary an out-of-tree subprocess sits on:
// a central store's members are other people, and the name they synced under is not this
// process's to hand over.
func TestMetricPayloadCarriesNoRawMember(t *testing.T) {
	now := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	usage := []store.UsageRow{{Day: "2026-07-16", Tool: "claude-code", Model: "m", Member: pluginRawMember, In: 10, Out: 20}}
	sessions := []store.SessionRow{{SessionID: "s1", Tool: "claude-code", Member: pluginRawMember, FirstTs: now.Add(-time.Hour), LastTs: now, Turns: 2}}
	in := analyze.BuildInput(usage, sessions, pricing.Table{}, now, 7*24*time.Hour, analyze.Delegation{})
	in.Trace = trace.New([]store.Timeline{{
		SessionID: "s1", Tool: "claude-code", Member: pluginRawMember, Timeline: "programmatic",
	}})

	envelope := buildMetricInput(&in, tracingPlugin())
	raw, err := envelope.marshal()
	if err != nil {
		t.Fatalf("marshal() err = %v", err)
	}
	if strings.Contains(string(raw), pluginRawMember) {
		t.Fatalf("metric plugin payload names the member raw:\n%s", raw)
	}
	if !strings.Contains(string(raw), "member-") {
		t.Fatalf("metric plugin payload dropped the member label entirely:\n%s", raw)
	}
}
