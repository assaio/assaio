// Package codex parses Codex CLI rollout logs into normalized usage records.
package codex

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
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

type sessionMeta struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Model     string    `json:"model"`
	Cwd       string    `json:"cwd"`
}

type turnContext struct {
	Model string `json:"model"`
}

type tokenCount struct {
	Type string `json:"type"`
	Info struct {
		Total struct {
			Input     int64 `json:"input_tokens"`
			Cached    int64 `json:"cached_input_tokens"`
			Output    int64 `json:"output_tokens"`
			Reasoning int64 `json:"reasoning_output_tokens"`
		} `json:"total_token_usage"`
	} `json:"info"`
}

type totals struct{ input, cached, output, reasoning int64 }

// parseState accumulates carry-forward fields (session, model, project), the previous
// cumulative token totals, and pending activity across a single rollout's records. ts is
// the most recently seen line timestamp -- seeded from session_meta, then advanced by
// every later line's own timestamp -- and is what each emitted record is stamped with.
type parseState struct {
	session string
	ts      time.Time
	model   string
	project string
	cwd     string
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
	sc := parser.NewScanner(r)
	st := &parseState{addedSoFar: make(map[string]int64)}
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
			st.pending.compactions++
		}
	}
	if err := sc.Err(); err != nil {
		return st.out, st.skipped, fmt.Errorf("scan codex rollout: %w", err)
	}
	st.flushTrailingActivity()
	return st.out, st.skipped, nil
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

func (st *parseState) applySessionMeta(payload json.RawMessage) {
	var m sessionMeta
	if err := json.Unmarshal(payload, &m); err != nil {
		st.skipped++
		return
	}
	st.session, st.ts = m.ID, m.Timestamp
	if st.model == "" {
		st.model = m.Model
	}
	if m.Cwd != "" {
		st.project = filepath.Base(m.Cwd)
		st.cwd = m.Cwd
	}
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

func (st *parseState) applyTokenCount(payload json.RawMessage) {
	rec, ok, err := st.tokenCountRecord(payload)
	if err != nil {
		st.skipped++
		return
	}
	if !ok {
		return
	}
	st.pending.flushInto(&rec)
	st.out = append(st.out, rec)
	st.turn++
}

func (st *parseState) tokenCountRecord(payload json.RawMessage) (usage.Record, bool, error) {
	var tk tokenCount
	if err := json.Unmarshal(payload, &tk); err != nil {
		return usage.Record{}, false, err
	}
	if tk.Type != "token_count" {
		return usage.Record{}, false, nil
	}
	cur := totals{tk.Info.Total.Input, tk.Info.Total.Cached, tk.Info.Total.Output, tk.Info.Total.Reasoning}
	d := totals{cur.input - st.prev.input, cur.cached - st.prev.cached, cur.output - st.prev.output, cur.reasoning - st.prev.reasoning}
	st.prev = cur
	if d.input <= 0 && d.output <= 0 && d.cached <= 0 {
		return usage.Record{}, false, nil
	}
	// Clamping can under-report non-cached input on cache-dominant turns (delta.cached > delta.input).
	return usage.Record{
		Tool:            tool,
		SessionID:       st.session,
		Timestamp:       st.ts,
		Model:           st.model,
		InputTokens:     parser.NonNeg(d.input - d.cached),
		CacheReadTokens: parser.NonNeg(d.cached),
		OutputTokens:    parser.NonNeg(d.output),
		ReasoningTokens: parser.NonNeg(d.reasoning),
		DedupeKey:       fmt.Sprintf("%s:%s:%d", st.fileFP, st.session, st.turn),
		Cwd:             st.cwd,
		Project:         st.project,
		Granularity:     "turn",
	}, true, nil
}
