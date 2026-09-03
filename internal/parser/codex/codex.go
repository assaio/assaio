// Package codex parses Codex CLI rollout logs into normalized usage records.
package codex

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

const tool = "codex"

// envelope is one rollout line's wrapper: every line, not just session_meta, carries its
// own top-level timestamp (RFC3339) alongside the type-discriminated payload.
type envelope struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

type turnContext struct {
	Model string `json:"model"`
}

// parseState accumulates carry-forward fields (session, model, project), the previous
// cumulative token totals, and pending activity across a single rollout's records. ts is
// the most recently seen line timestamp -- seeded from session_meta, then advanced by
// every later line's own timestamp -- and is what each emitted record is stamped with.
type parseState struct {
	session    string
	ts         time.Time
	model      string
	project    string
	cwd        string
	entrypoint string
	// fileFP is a content fingerprint of this rollout's first line, prefixed onto every
	// DedupeKey so a session id reused across two files (a resumed session) never
	// collides; see parser.FileFingerprint.
	fileFP  string
	prev    totals
	turn    int
	out     []usage.Record
	skipped int
	// pending holds edit/tool-call/compaction activity seen since the last emitted
	// record; applyTokenCount flushes it onto the record a token_count closes.
	pending activity
	// addedSoFar tracks AI-added lines per file path across the whole rollout, in memory
	// only, to detect rework; the file path is never copied onto a Record.
	addedSoFar map[string]int64
	// steps is the second reading of the same scan: the sequence behind the records.
	steps *stepRecorder
}

// Parse reads a Codex rollout (JSONL). token_count events carry cumulative totals, so
// each record is the delta from the previous event. input_tokens includes cached
// tokens, so non-cached input and cache-read tokens are stored separately. The model is
// carried forward from the most recent turn_context. Edit (patch_apply_end), tool-call
// (function_call/custom_tool_call), and compaction activity is attributed to the record
// the *next* token_count closes -- it accumulates in pending and is flushed there; any
// activity trailing the last token_count is flushed onto the last emitted record instead
// of being dropped (see flushTrailingActivity). Each record is stamped with its own
// line's timestamp (every rollout line carries one), falling back to the last known
// timestamp -- session_meta's, until a later line supplies one -- when a line lacks it.
// skipped counts lines that failed to unmarshal as JSON; a scanner-level error still
// aborts the parse.
func Parse(r io.Reader) ([]usage.Record, int, error) {
	recs, _, skipped, err := ParseAll(r)
	return recs, skipped, err
}

// ParseAll reads a rollout once and returns both readings of it: the usage records Parse
// documents, and the step sequence behind them. One pass rather than two, for the reason
// Claude's carries: two passes meant two orders of checks over the same line, and the readings
// drifted apart in exactly the ways a shared state could not prevent. Parse is a wrapper so no
//
// The sequence records the model response each turn opens with, the calls it made, one step per
// file a patch touched, and each context compaction. It states no outcome on a call Codex marks
// "completed" -- see stepRecorder.toolCall for why that word is not a success.
func ParseAll(r io.Reader) ([]usage.Record, []usage.Step, int, error) {
	sc := parser.NewScanner(r)
	st := &parseState{addedSoFar: make(map[string]int64), steps: newStepRecorder()}
	for sc.Scan() {
		raw := sc.Bytes()
		if len(raw) == 0 {
			continue
		}
		if st.fileFP == "" {
			st.fileFP = parser.FileFingerprint(raw)
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			st.skipped++
			continue
		}
		if !env.Timestamp.IsZero() {
			st.ts = env.Timestamp
		}
		switch env.Type {
		case "session_meta":
			st.applySessionMeta(env.Payload)
		case "turn_context":
			st.applyTurnContext(env.Payload)
		case "event_msg":
			st.applyEventMsg(env.Payload)
		case "response_item":
			st.applyResponseItem(env.Payload)
		case "compacted":
			// Newer builds also write an event_msg/context_compacted for the same overflow --
			// on the audited corpus it is present in 14 of the 18 rollouts that compacted at
			// all, one for one, and in none that lack this line. Reading only this one keeps
			// the sequence and the record counting the same event once.
			st.pending.compactions++
			st.steps.compaction(st)
		}
	}
	if err := sc.Err(); err != nil {
		return st.out, st.steps.stamp(st.session), st.skipped, fmt.Errorf("scan codex rollout: %w", err)
	}
	st.flushTrailingActivity()
	return st.out, st.steps.stamp(st.session), st.skipped, nil
}

// flushTrailingActivity attributes activity that occurred after the last token_count to
// the last emitted record, so it is never silently dropped for want of a closing turn
// boundary. Dropped, like any activity, when no record has been emitted yet.
func (st *parseState) flushTrailingActivity() {
	if len(st.out) == 0 {
		return
	}
	st.pending.flushInto(&st.out[len(st.out)-1])
}

func (st *parseState) applyTurnContext(payload json.RawMessage) {
	var tc turnContext
	if err := json.Unmarshal(payload, &tc); err != nil {
		st.skipped++
		return
	}
	if tc.Model != "" {
		st.model = tc.Model
	}
}
