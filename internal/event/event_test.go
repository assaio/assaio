package event

import (
	"strings"
	"testing"
	"time"
)

var (
	occurred = time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	observed = time.Date(2026, 8, 2, 11, 0, 0, 0, time.UTC)
)

func validEvent() Event {
	return Event{
		SpecVersion: SpecVersion,
		Type:        TypeUsage,
		ID:          "turn-1",
		Source:      Source{Name: "claude-code", Build: "v0.6.0"},
		OccurredAt:  occurred,
		ObservedAt:  observed,
		TimeSource:  TimeStated,
		Grain:       GrainTurn,
		Privacy:     Pseudonymous,
		Provenance:  Parsed,
		Subject:     Subject{Project: "assaio", Session: "s1"},
		Payload:     Usage{Model: "claude-opus-5", InputTokens: 10, OutputTokens: 20},
	}
}

func TestValidateAcceptsAWellFormedEvent(t *testing.T) {
	e := validEvent()
	if err := e.Validate(); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}
}

func TestValidateRejects(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Event)
		wantErr string
	}{
		{"unknown spec version", func(e *Event) { e.SpecVersion = 99 }, "unknown spec version"},
		{"unknown type", func(e *Event) { e.Type = "ai.vibes.observed" }, "unknown event type"},
		{"no payload", func(e *Event) { e.Payload = nil }, "carries no payload"},
		{"payload disagrees with type", func(e *Event) {
			e.Type, e.Payload = TypeEdit, Usage{OutputTokens: 1}
		}, "payload is ai.usage.observed but envelope says ai.edit.observed"},
		{"no id", func(e *Event) { e.ID = "" }, "no id"},
		{"no source name", func(e *Event) { e.Source.Name = "" }, "no source name"},
		{"unknown grain", func(e *Event) { e.Grain = "hour" }, "unknown grain"},
		{"unknown privacy", func(e *Event) { e.Privacy = "secret" }, "unknown privacy"},
		{"unknown provenance", func(e *Event) { e.Provenance = "estimated" }, "unknown provenance"},
		{"unknown time source", func(e *Event) { e.TimeSource = "guessed" }, "unknown time source"},
		{"no occurrence time", func(e *Event) { e.OccurredAt = time.Time{} }, "no occurrence time"},
		{"no observation time", func(e *Event) { e.ObservedAt = time.Time{} }, "no observation time"},
		{"payload invalid", func(e *Event) {
			e.Payload = Usage{OutputTokens: -1}
		}, "outputTokens is negative"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := validEvent()
			tc.mutate(&e)
			err := e.Validate()
			if err == nil {
				t.Fatal("want an error, got none")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not mention %q", err, tc.wantErr)
			}
		})
	}
}

// A batch stamps one reading time, so a record parsed late in the pass can be newer than it.
// Rejecting that dropped 51 of 324,289 real records, all from live sessions -- see ADR 0007.
func TestValidateAcceptsARecordNewerThanTheBatchReadingTime(t *testing.T) {
	e := validEvent()
	e.OccurredAt = e.ObservedAt.Add(5 * time.Second)
	if err := e.Validate(); err != nil {
		t.Fatalf("a record written while the pass ran must still adapt: %v", err)
	}
}

// Estimated and attributed describe a signal or an attribution edge, never an observation.
// ADR 0007 says so; this asserts nobody quietly widens the vocabulary to include them.
func TestProvenanceExcludesEstimatedAndAttributed(t *testing.T) {
	for _, forbidden := range []string{"estimated", "attributed", "inferred"} {
		if valid(provenances, forbidden) {
			t.Fatalf("%q must not be an observation provenance", forbidden)
		}
	}
}
