package plugin

import (
	"time"

	"github.com/assaio/assaio/internal/store"
	"github.com/assaio/assaio/internal/trace"
)

// metricTimeline is one step sequence: a session's main loop, or one sub-agent inside it.
type metricTimeline struct {
	Tool      string `json:"tool"`
	SessionID string `json:"sessionId"`
	Member    string `json:"member"`
	// Timeline is "" for a session's main loop, otherwise the id of the sub-agent whose
	// transcript records its parent's session id. Part of a sequence's identity rather than
	// decoration: a forked sub-agent replays its origin's prefix, so one step can legitimately
	// appear in three sequences, and ordinal is unique only within (sessionId, member, timeline).
	Timeline   string `json:"timeline"`
	Entrypoint string `json:"entrypoint"`
	Project    string `json:"project"`
	// Scope is the population this sequence belongs to: "interactive", "sub-agent",
	// "programmatic" or "unstated" (internal/trace). Precomputed here on purpose -- 89% of the
	// sequences on the audited store are programmatic and hold 5.7% of its steps, so a rate that
	// spans two scopes describes neither.
	Scope string       `json:"scope"`
	Steps []metricStep `json:"steps"`
}

// metricStep is one observation in a sequence. Content-free by construction: no prompt, no code,
// no file name and no path (PRIVACY.md).
type metricStep struct {
	// Ordinal is the position within this sequence, from 1. A sequence may legitimately start
	// above 1 -- the retention horizon cuts the opening off one that straddles it.
	Ordinal int64     `json:"ordinal"`
	At      time.Time `json:"at"`
	// Kind is one of assistant/read/search/command/edit/other/compaction.
	Kind string `json:"kind"`
	// Outcome is ok/error/denied/truncated, or "" when the source said nothing -- which is a
	// different fact from ok and must never be read as one.
	Outcome string `json:"outcome"`
	Model   string `json:"model"`
	// Tokens is the response's own total on an assistant step and 0 on a tool call, which no log
	// read today accounts for separately.
	Tokens int64 `json:"tokens"`
	// TargetRef stands for the file this step named, 0 when it named none, and is comparable only
	// within one sequence. Never a path, and never a digest of one: a digest is reversible by
	// anyone holding the repository.
	TargetRef int64 `json:"targetRef"`
}

// traceWire maps the window's sequences onto the wire, stamping each with the scope the core
// classified it as. Nothing is filtered: the core cannot know which scope a plugin will declare,
// and dropping a scope here would hand it a denominator it never chose.
func traceWire(set *trace.Set) []metricTimeline {
	sequences := set.All()
	out := make([]metricTimeline, 0, len(sequences))
	for i := range sequences {
		t := &sequences[i]
		out = append(out, metricTimeline{
			Tool: t.Tool, SessionID: t.SessionID, Member: t.Member, Timeline: t.Timeline,
			Entrypoint: t.Entrypoint, Project: t.Project, Scope: trace.Scope(t),
			Steps: stepWire(t.Steps),
		})
	}
	return out
}

func stepWire(steps []store.TimelineStep) []metricStep {
	out := make([]metricStep, 0, len(steps))
	for i := range steps {
		s := &steps[i]
		out = append(out, metricStep{
			Ordinal: s.Ordinal, At: s.Timestamp, Kind: s.Kind, Outcome: s.Outcome,
			Model: s.Model, Tokens: s.Tokens, TargetRef: s.TargetRef,
		})
	}
	return out
}
