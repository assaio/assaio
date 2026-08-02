package event

import (
	"fmt"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// editIDSuffix namespaces the edit observation derived from the same record as its usage
// observation, so both are idempotent and neither can be mistaken for the other.
const editIDSuffix = "#edit"

// FromRecord adapts one parsed usage record into canonical observations: always a usage
// observation, and an edit observation only when the record actually carries activity. This
// is the whole of B89's migration story -- usage_record keeps its schema, its dedupe contract
// and its history, and the contract is used only on the new evidence path.
//
// observedAt is passed in rather than read from the clock so the adapter is deterministic and
// a caller adapting a batch stamps one reading time across it.
func FromRecord(r *usage.Record, build string, observedAt time.Time) ([]Event, error) {
	// Cwd, Subpath and GitBranch are deliberately not carried: the first is transient by
	// PRIVACY.md, and a branch name can name a client or a project the store never held.
	base := Event{
		SpecVersion: SpecVersion,
		Source:      Source{Name: r.Tool, Build: build},
		OccurredAt:  r.Timestamp,
		ObservedAt:  observedAt,
		TimeSource:  TimeStated,
		Grain:       r.Granularity,
		Privacy:     Pseudonymous,
		Provenance:  Parsed,
		Subject:     Subject{Project: r.Project, Session: r.SessionID, Member: r.Member},
	}

	events := []Event{base.with(TypeUsage, r.DedupeKey, Usage{
		Model:            r.Model,
		InputTokens:      r.InputTokens,
		OutputTokens:     r.OutputTokens,
		CacheReadTokens:  r.CacheReadTokens,
		CacheWriteTokens: r.CacheWriteTokens,
		ReasoningTokens:  r.ReasoningTokens,
	})}
	if edit := (Edit{
		LinesAdded:   r.LinesAdded,
		LinesRemoved: r.LinesRemoved,
		Edits:        r.Edits,
		ReworkLines:  r.ReworkLines,
	}); edit.hasActivity() {
		events = append(events, base.with(TypeEdit, r.DedupeKey+editIDSuffix, edit))
	}

	for i := range events {
		if err := events[i].Validate(); err != nil {
			return nil, fmt.Errorf("adapting %s record: %w", r.Tool, err)
		}
	}
	return events, nil
}

// with stamps a payload and its identity onto a copy of the shared envelope, leaving the
// original free to seed the next observation from the same record.
func (e *Event) with(eventType, id string, payload Payload) Event {
	out := *e
	out.Type, out.ID, out.Payload = eventType, id, payload
	return out
}
