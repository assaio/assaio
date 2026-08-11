package plugin

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/assaio/assaio/internal/analyze"
	"github.com/assaio/assaio/internal/pricing"
	"github.com/assaio/assaio/internal/store"
)

// notOnTheWire lists the analyze.Input fields a metric plugin deliberately does not
// receive, each with the reason it cannot need one. Everything a built-in validator reads
// and a plugin cannot makes the extension surface a demonstration rather than a contract,
// so a new field belongs on the envelope unless it can be justified here.
var notOnTheWire = map[string]string{
	"Recent":   "sent as recentDays, in whole days rather than a Go duration",
	"Ingested": "the core stamps it onto the plugin's own Result (analyze.Stamp)",
	"ParsedBy": "the core stamps it onto the plugin's own Result (analyze.Stamp)",
}

func TestEveryInputFieldReachesAMetricPlugin(t *testing.T) {
	onWire := map[string]bool{}
	wire := reflect.TypeOf(metricInput{})
	for i := range wire.NumField() {
		onWire[wire.Field(i).Name] = true
	}
	in := reflect.TypeOf(analyze.Input{})
	for i := range in.NumField() {
		name := in.Field(i).Name
		if onWire[name] || notOnTheWire[name] != "" {
			continue
		}
		t.Errorf("analyze.Input.%s does not reach a metric plugin: add it to metricInput and "+
			"buildMetricInput, or list it in notOnTheWire with the reason a plugin cannot need it", name)
	}
}

// A stale exception is as misleading as a missing field: it claims a decision about a field
// that no longer exists.
func TestNoStaleWireExceptions(t *testing.T) {
	in := reflect.TypeOf(analyze.Input{})
	for name := range notOnTheWire {
		if _, ok := in.FieldByName(name); !ok {
			t.Errorf("notOnTheWire lists %q, which analyze.Input no longer has", name)
		}
	}
}

// fullInput carries a non-zero value in every analyze.Input field, so a mapping that is
// missing shows up as a zero on the wire. TestTheProbeInputIsComplete keeps it honest.
func fullInput() analyze.Input {
	cost := 12.5
	return analyze.Input{
		WindowStart:     time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Usage:           []store.UsageRow{{Day: "2026-08-02", Tool: "claude-code", Model: "m", In: 1}},
		Sessions:        []store.SessionRow{{SessionID: "s", Tool: "claude-code", Turns: 1}},
		Prices:          pricing.Table{"m": {Input: 0.1}},
		Now:             time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC),
		Recent:          7 * 24 * time.Hour,
		Delegation:      analyze.Delegation{Sub: 1, Total: 2},
		ByModel:         []analyze.ModelStat{{Model: "m", Tokens: 3}},
		ByProject:       []analyze.ProjectStat{{Project: "p", Lines: 4}},
		Totals:          analyze.Totals{Tokens: 5, Cost: &cost, Priced: true},
		PlanMonthlyCost: 200,
		Skills:          []store.AttributionRow{{Name: "s", Tokens: 6}},
		Agents:          []store.AttributionRow{{Name: "a", Tokens: 7}},
		TurnSizing:      []store.ModelTurns{{Model: "m", Turns: 8}},
		CacheMisses:     []store.CacheMissRow{{Tool: "claude-code", Reason: "r", Turns: 9}},
		Ingested:        time.Date(2026, 8, 9, 1, 0, 0, 0, time.UTC),
		ParsedBy:        "v0.17.0",
	}
}

// The probe is only evidence if it actually sets everything: a field left at its zero value
// here would make the mapping canary below pass for a mapping that does not exist.
func TestTheProbeInputIsComplete(t *testing.T) {
	v := reflect.ValueOf(fullInput())
	for i := range v.NumField() {
		if v.Field(i).IsZero() {
			t.Errorf("fullInput leaves %s at its zero value, so the mapping canary cannot see it",
				v.Type().Field(i).Name)
		}
	}
}

// Presence on the envelope is not enough: a field buildMetricInput never maps reaches every
// plugin as an honest-looking empty, and the name-level canary above cannot see it. Every
// wire field that shares a name with an Input field must come out non-zero from an Input
// where nothing is zero.
func TestEveryMappedWireFieldIsActuallyFilled(t *testing.T) {
	in := fullInput()
	got := reflect.ValueOf(buildMetricInput(&in))
	inputFields := reflect.TypeOf(analyze.Input{})
	for i := range got.NumField() {
		name := got.Type().Field(i).Name
		if _, shared := inputFields.FieldByName(name); !shared {
			continue // Version, RecentDays, Answers: wire-only, checked by their own tests
		}
		if got.Field(i).IsZero() {
			t.Errorf("metricInput.%s is on the envelope but buildMetricInput never fills it, so "+
				"every plugin reads it as empty", name)
		}
	}
}

// Presence on the envelope is not the same as being filled: a field buildMetricInput forgets
// to map reads as an honest empty to every plugin. Every field carries a distinct non-zero
// value, so a mapping crossed with the wrong source fails too.
func TestBuildMetricInputMapsTheWindowAggregates(t *testing.T) {
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	in := analyze.Input{
		WindowStart:     start,
		PlanMonthlyCost: 200,
		Skills:          []store.AttributionRow{{Name: "brainstorming", Tokens: 11, Lines: 12, Records: 13, Sessions: 14}},
		Agents:          []store.AttributionRow{{Name: "reviewer", Tokens: 21, Lines: 22, Records: 23, Sessions: 24}},
		TurnSizing:      []store.ModelTurns{{Model: "claude-opus-5", Turns: 31, SmallTurns: 32}},
		CacheMisses:     []store.CacheMissRow{{Tool: "claude-code", Reason: "ttl_expired", Turns: 41}},
	}

	got := buildMetricInput(&in)

	if !got.WindowStart.Equal(start) {
		t.Errorf("windowStart = %v, want %v", got.WindowStart, start)
	}
	if got.PlanMonthlyCost != 200 {
		t.Errorf("planMonthlyCost = %v, want 200", got.PlanMonthlyCost)
	}
	if len(got.Skills) != 1 || got.Skills[0] != (metricAttributionRow{Name: "brainstorming", Tokens: 11, Lines: 12, Records: 13, Sessions: 14}) {
		t.Errorf("skills = %+v", got.Skills)
	}
	if len(got.Agents) != 1 || got.Agents[0].Name != "reviewer" || got.Agents[0].Sessions != 24 {
		t.Errorf("agents = %+v", got.Agents)
	}
	if len(got.TurnSizing) != 1 || got.TurnSizing[0] != (metricModelTurns{Model: "claude-opus-5", Turns: 31, SmallTurns: 32}) {
		t.Errorf("turnSizing = %+v", got.TurnSizing)
	}
	if len(got.CacheMisses) != 1 || got.CacheMisses[0] != (metricCacheMissRow{Tool: "claude-code", Reason: "ttl_expired", Turns: 41}) {
		t.Errorf("cacheMisses = %+v", got.CacheMisses)
	}
}

// An empty window must reach the plugin as an empty list rather than a null, so a plugin
// that ranges over it reads "nothing was attributed" instead of failing to decode.
func TestWindowAggregatesMarshalAsListsWhenEmpty(t *testing.T) {
	envelope := buildMetricInput(&analyze.Input{})
	out, err := envelope.marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{`"skills":[]`, `"agents":[]`, `"turnSizing":[]`, `"cacheMisses":[]`} {
		if !strings.Contains(string(out), key) {
			t.Errorf("empty envelope missing %s, got %s", key, out)
		}
	}
}
