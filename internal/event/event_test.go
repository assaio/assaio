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
		Type:        TypeCommit,
		ID:          "0f1e2d3c4b5a",
		Source:      Source{Name: "git", Build: "v0.24.0"},
		OccurredAt:  occurred,
		ObservedAt:  observed,
		TimeSource:  TimeStated,
		Grain:       GrainCommit,
		Privacy:     LocalOnly,
		Provenance:  Parsed,
		Subject:     Subject{Project: "assaio"},
		Payload:     Commit{Parents: 1, FilesChanged: 1, LinesAdded: 3, Files: FileCategories{Source: 1}},
	}
}

// mislabelled answers to a type it is not filed under. It exists because the contract registers
// a single type today, so no pair of real payloads can reach the envelope-versus-payload check;
// the check guards the release a second type lands in.
type mislabelled struct{}

func (mislabelled) eventType() string { return "vcs.tag.observed" }
func (mislabelled) validate() error   { return nil }

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
			e.Payload = mislabelled{}
		}, "payload is vcs.tag.observed but envelope says vcs.commit.observed"},
		{"no id", func(e *Event) { e.ID = "" }, "no id"},
		{"no source name", func(e *Event) { e.Source.Name = "" }, "no source name"},
		{"unknown grain", func(e *Event) { e.Grain = "hour" }, "unknown grain"},
		{"unknown privacy", func(e *Event) { e.Privacy = "secret" }, "unknown privacy"},
		{"unknown provenance", func(e *Event) { e.Provenance = "estimated" }, "unknown provenance"},
		{"unknown time source", func(e *Event) { e.TimeSource = "guessed" }, "unknown time source"},
		{"no occurrence time", func(e *Event) { e.OccurredAt = time.Time{} }, "no occurrence time"},
		{"no observation time", func(e *Event) { e.ObservedAt = time.Time{} }, "no observation time"},
		{"payload invalid", func(e *Event) {
			e.Payload = Commit{LinesAdded: -1}
		}, "linesAdded is negative"},
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

// A batch stamps one reading time, so an artifact read late in the pass can be newer than it.
// Rejecting that dropped 51 of 324,289 real records, all from live sessions -- see ADR 0007.
func TestValidateAcceptsAnObservationNewerThanTheBatchReadingTime(t *testing.T) {
	e := validEvent()
	e.OccurredAt = e.ObservedAt.Add(5 * time.Second)
	if err := e.Validate(); err != nil {
		t.Fatalf("an artifact written while the pass ran must still be observable: %v", err)
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
