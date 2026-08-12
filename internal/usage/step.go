package usage

import "time"

// The closed vocabulary of what one step was. It mirrors the tool-call classification the
// parsers already apply (internal/parser.StepKind), plus the two kinds that are not tool calls
// at all, so a step and a turn's ToolReads..ToolOther counts can never become two opinions
// about the same tool name.
const (
	StepAssistant  = "assistant"
	StepRead       = "read"
	StepSearch     = "search"
	StepCommand    = "command"
	StepEdit       = "edit"
	StepOther      = "other"
	StepCompaction = "compaction"
)

// The closed vocabulary of how one step ended. "" is not a sixth value: it means the source
// said nothing, which is a different fact from OutcomeOK and must never be read as one.
//
// OutcomeTruncated exists because stop_reason:max_tokens is read, and is rare rather than
// absent: 5 of 5,706 audited transcripts. There is deliberately no "aborted": interrupted:true
// never occurs in that corpus and nothing reads it, so a vocabulary member for it would be a
// bucket no code can ever fill -- exactly the silent zero these rules exist against. Codex's
// turn_aborted is the same story and lands when Codex gains a step reading.
const (
	OutcomeOK        = "ok"
	OutcomeError     = "error"
	OutcomeDenied    = "denied"
	OutcomeTruncated = "truncated"
)

// Step is one observation in a session's sequence: what the agent did, in what order, at what
// token cost, and how it ended. Content-free by construction -- no prompt, no code, no file
// name and no path reaches this type.
//
// Scope (interactive vs SDK, main-loop vs sub-agent) is deliberately absent: it is a property
// of the session, already recorded on usage_record, and duplicating it per step would let the
// two disagree.
type Step struct {
	Tool      string
	SessionID string
	// Timeline separates the sequences that share a session. A sub-agent transcript records
	// its parent's SessionID, so ordering by session alone interleaved a main loop with every
	// sub-agent it launched and put 222 steps at position 1 on the worst session in the
	// maintainer's store. It is part of a step's identity too, not only its ordering: a forked
	// sub-agent replays its origin's prefix under a new agent id, so one message.id can belong
	// to three sequences. "" is the main loop; anything else is the agent id the sub-agent's
	// own lines carry. Ordinal is unique within (SessionID, Member, Timeline), never within
	// SessionID alone.
	Timeline string
	// DedupeKey identifies this step at its source, so re-reading a transcript rewrites the
	// same row instead of appending a second copy of the session.
	DedupeKey string
	Timestamp time.Time
	// Ordinal is the step's position within (SessionID, Member, Timeline), from 1. It orders
	// steps a shared timestamp cannot: a turn and its tool calls are frequently stamped the
	// same second. A stored sequence may start above 1 -- the retention horizon cuts the
	// opening off a timeline that straddles it -- so a reader allows for that rather than
	// reading it as loss.
	Ordinal int64
	Kind    string
	// Outcome is one of the constants above, or "" when the source did not say.
	Outcome string
	Model   string
	// Tokens is the token cost attributed to this step -- the response's own total on an
	// assistant step, 0 on a tool call, which no log read today accounts for separately.
	Tokens int64
	// TargetRef distinguishes the things a sequence acted on without naming any of them: an
	// integer assigned in first-seen order within one Timeline, 0 when a step has no target.
	// Within a timeline, never across a session: a session's main transcript and each of its
	// sub-agents are separate files, and 3 in one of them is unrelated to 3 in another.
	// Deliberately not a digest of the path: a digest is reversible by anyone holding the
	// repository, because paths carry almost no entropy. "The same target nine times" stays
	// answerable and "which file" stays unanswerable, permanently.
	TargetRef int64
}

// ValidStepKind reports whether k is in the closed vocabulary. The boundary rejects anything
// else rather than storing a value no validator can interpret.
func ValidStepKind(k string) bool {
	switch k {
	case StepAssistant, StepRead, StepSearch, StepCommand, StepEdit, StepOther, StepCompaction:
		return true
	}
	return false
}

// ValidStepOutcome reports whether o is in the closed vocabulary or is the empty "not stated".
func ValidStepOutcome(o string) bool {
	switch o {
	case "", OutcomeOK, OutcomeError, OutcomeDenied, OutcomeTruncated:
		return true
	}
	return false
}
