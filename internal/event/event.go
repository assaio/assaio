// Package event defines the canonical observation contract of the evidence graph: the shape
// every collector emits and every analyzer reads, so a git reader, a connector and a log
// parser can be added without each one teaching the analyzers a new record.
//
// An event is an interface contract, not a storage format -- there is no event table and no
// migration behind it. See docs/adr/0007-canonical-event-contract.md.
package event

import (
	"errors"
	"fmt"
	"time"
)

// SpecVersion is the contract version every envelope this build produces conforms to.
// Fields may be added within a version; removing or retyping one is a new version.
const SpecVersion = 1

// Event is one observation: a versioned envelope and exactly one closed payload. Everything
// on the envelope answers a question a consumer would otherwise have to ask the collector --
// where this came from, when it happened as opposed to when it was read, how coarse it is,
// and whether it is safe to send anywhere.
//
// Confidence is deliberately absent. An observation either happened or was not observed; a
// verdict's confidence (B81) and an edge's attribution confidence (B85) are computed *from*
// these fields, not carried alongside them.
type Event struct {
	SpecVersion int    `json:"specVersion"`
	Type        string `json:"type"`
	// ID is the source's own key for this observation, never a generated value, so
	// re-reading the same artifact produces the same event. Idempotency is a property of
	// the identifier rather than of a de-duplicating consumer.
	ID         string    `json:"id"`
	Source     Source    `json:"source"`
	OccurredAt time.Time `json:"occurredAt"`
	ObservedAt time.Time `json:"observedAt"`
	TimeSource string    `json:"timeSource"`
	Grain      string    `json:"grain"`
	Privacy    string    `json:"privacy"`
	Provenance string    `json:"provenance"`
	Subject    Subject   `json:"subject"`
	Payload    Payload   `json:"payload"`
}

// Source is who produced the observation and from what. Version is the source's own format
// or API version where it states one -- today's log parsers read formats that carry no
// version at all, so it is usually empty and that emptiness is the honest answer.
type Source struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Build   string `json:"build,omitempty"`
}

// Subject is what the observation is about. Every field is optional because not every
// observation has all three: a commit has no session, and purely local usage has no member.
type Subject struct {
	Project string `json:"project,omitempty"`
	Session string `json:"session,omitempty"`
	Member  string `json:"member,omitempty"`
}

// Payload is one event type's body. The interface is closed by its unexported methods: a
// payload can only be defined in this package, which is what makes "there is nowhere to put
// a prompt, a diff or a commit message" a property of the contract rather than a promise.
type Payload interface {
	eventType() string
	validate() error
}

// Validate rejects an envelope the contract cannot vouch for. Collectors call it where the
// event is constructed, so a malformed observation never reaches a consumer -- the same
// reject-never-fabricate posture the exec plugin protocols take at their boundary.
func (e *Event) Validate() error {
	if e.SpecVersion != SpecVersion {
		return fmt.Errorf("unknown spec version %d (want %d)", e.SpecVersion, SpecVersion)
	}
	if !known(e.Type) {
		return fmt.Errorf("unknown event type %q", e.Type)
	}
	if e.Payload == nil {
		return errors.New("event carries no payload")
	}
	if got := e.Payload.eventType(); got != e.Type {
		return fmt.Errorf("payload is %s but envelope says %s", got, e.Type)
	}
	if e.ID == "" {
		return errors.New("event has no id, so re-observing it would duplicate it")
	}
	if e.Source.Name == "" {
		return errors.New("event has no source name")
	}
	if err := e.validateVocabularies(); err != nil {
		return err
	}
	if err := e.validateClocks(); err != nil {
		return err
	}
	return e.Payload.validate()
}

func (e *Event) validateVocabularies() error {
	for _, f := range []struct {
		name, value string
		vocabulary  []string
	}{
		{"grain", e.Grain, grains},
		{"privacy", e.Privacy, privacies},
		{"provenance", e.Provenance, provenances},
		{"time source", e.TimeSource, timeSources},
	} {
		if !valid(f.vocabulary, f.value) {
			return fmt.Errorf("unknown %s %q (want %v)", f.name, f.value, f.vocabulary)
		}
	}
	return nil
}

// validateClocks requires both clocks and deliberately does not order them. An earlier draft
// rejected OccurredAt after ObservedAt as a reversed clock; run against 324,289 real records
// it dropped 51 of them, all from sessions still being written. ObservedAt is a *reading*
// time, stamped once for a batch, so a file read late in a pass legitimately contains records
// newer than the stamp -- and a vendor's clock is not ours to police. Rejecting here would
// also make the evidence path stricter than the store, silently losing records the store
// keeps, which is two disagreeing views of the same data rather than one strict one.
func (e *Event) validateClocks() error {
	if e.OccurredAt.IsZero() {
		return errors.New("event has no occurrence time")
	}
	if e.ObservedAt.IsZero() {
		return errors.New("event has no observation time")
	}
	return nil
}
