// Package agy parses Antigravity CLI conversation transcripts into normalized usage records.
//
// This source carries no token accounting of any kind. Antigravity CLI writes a model's
// context-window occupancy into an unnamed protobuf field of its per-conversation SQLite
// database, and nothing else: no input, output, cache or reasoning counter exists anywhere
// under its data root, and the account it runs on is a plan rather than API billing. Every
// token field on every record this parser emits is therefore zero *and* the depth matrix
// declares the token signals unanswered -- two separate statements, because an undeclared
// zero is exactly the fabricated figure ADR 0011 exists to prevent.
package agy

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

const tool = "agy"

// line is one transcript entry, reduced to the fields that carry accounting. The
// content-bearing fields the vendor writes beside them -- content, thinking, and every
// tool-call argument -- have no field in this struct and therefore no path into memory, into
// a record, or into an error string. That omission is this parser's privacy boundary, it is
// asserted by TestNoPromptContentEscapes, and adding a field here moves it.
type line struct {
	StepIndex int    `json:"step_index"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
	ToolCalls []struct {
		Name string `json:"name"`
	} `json:"tool_calls"`
}

// sourceModel marks the entries a model authored. The other two values the vendor writes --
// USER_EXPLICIT for a person's prompt and SYSTEM for a stream or platform error -- are not
// model turns, so they are filtered rather than counted: on the captured corpus 252 of 500
// conversations hold a single USER_EXPLICIT line and nothing else, and emitting a row of
// zeros for each would report 500 sessions of AI usage where 248 happened.
const sourceModel = "MODEL"

// ParseTranscript reads one conversation's transcript.jsonl and returns one record per model
// turn -- every consecutive MODEL entry sharing one created_at, not every MODEL line; the fold
// below says why those differ and what the grouping costs.
//
// conversationID is the transcript's own directory name -- the vendor writes no id inside the
// file -- and is passed in rather than derived here so the parser stays hermetic and a re-read
// produces the same dedupe keys. An empty id yields nothing: the key is "<conversation>:<step>",
// so every unidentified conversation would collide on one stored row.
//
// skipped counts lines that failed to unmarshal as JSON, plus a model turn with no usable
// created_at -- every window is bounded by `ts >= ?`, so an undatable record would be stored
// and then invisible to every figure over it -- plus a turn repeating a step_index already
// emitted. A scanner-level error is a problem with the whole file rather than one line and
// aborts the parse.
func ParseTranscript(r io.Reader, conversationID string) ([]usage.Record, int, error) {
	sc := parser.NewScanner(r)
	var recs []usage.Record
	var skipped int
	// step_index is the dedupe key's only varying part, and the vendor writes it once per
	// entry: 0 repeats across the 500 captured conversations. A repeat is therefore damage,
	// and emitting it would hand the store two records under one key for ON CONFLICT to drop
	// the second of -- a loss nobody counts. Counting it here is the same skip-and-count
	// policy a corrupt line gets, for the same reason.
	seen := make(map[int]struct{})
	for sc.Scan() {
		raw := sc.Bytes()
		if len(raw) == 0 {
			continue
		}
		var l line
		if err := json.Unmarshal(raw, &l); err != nil {
			skipped++
			continue
		}
		if l.Source != sourceModel || conversationID == "" {
			continue
		}
		// The zero time is judged on the value, not on the parse: RFC 3339 accepts
		// "0001-01-01T0:00:00Z", so an undatable line can arrive through a successful parse
		// as easily as through a failed one, and both are undatable for the same reason.
		at, err := time.Parse(time.RFC3339, l.CreatedAt)
		if err != nil || at.IsZero() {
			skipped++
			continue
		}
		if _, repeat := seen[l.StepIndex]; repeat {
			skipped++
			continue
		}
		seen[l.StepIndex] = struct{}{}
		// One response, two entries: the vendor splits a model turn into a GENERIC entry
		// carrying the content and a PLANNER_RESPONSE entry carrying the thinking and the
		// tool calls, both stamped with the same second. Counted apart they would report two
		// turns where the model answered once -- 8 of the 653 MODEL lines across the 500
		// captured conversations, in 8 pairs, so 645 turns rather than 653.
		//
		// The boundary this accepts: two genuinely distinct responses inside one second fold
		// into one turn. The vendor writes no response id, so a timestamp is the only thing
		// there is to group on, and this is the safe side of that trade -- under-counting
		// turns understates activity, where counting a split response twice would inflate
		// every per-turn figure with a turn nobody took. Only consecutive entries fold: a
		// later entry returning to an earlier second is a new turn, not a continuation.
		if n := len(recs); n == 0 || !recs[n-1].Timestamp.Equal(at) {
			// The dedupe key keeps the first entry's step_index, so a re-read of an unchanged
			// file folds the same entries onto the same key and inserts nothing new.
			recs = append(recs, record(conversationID, at, l.StepIndex))
		}
		addToolCalls(&recs[len(recs)-1], &l)
	}
	if err := sc.Err(); err != nil {
		return recs, skipped, fmt.Errorf("scan agy transcript: %w", err)
	}
	return recs, skipped, nil
}

// record opens one model turn as a usage record, with no activity in it yet: every entry the
// turn is made of contributes through addToolCalls, including the first. Every token field is
// left at zero -- see the package doc for why that is this source's declared state rather than
// an unfinished read.
func record(conversationID string, at time.Time, stepIndex int) usage.Record {
	return usage.Record{
		Tool:      tool,
		SessionID: conversationID,
		Timestamp: at,
		// step_index is the vendor's own position within the conversation and is unique
		// within one transcript, so it needs none of the file-fingerprint prefixing a
		// parser-side counter would: two parses of an unchanged file agree by construction.
		// Formatted verbatim, negatives included, because mapping -1 onto 0 would key a
		// corrupt line onto a real turn and let ON CONFLICT drop the real one.
		DedupeKey:   conversationID + ":" + strconv.Itoa(stepIndex),
		Granularity: "turn",
	}
}

// addToolCalls folds one entry's tool calls into the turn it belongs to, classified by name.
// The class split and ToolCalls are the same calls counted two ways, so both sides grow
// together or the record contradicts itself.
func addToolCalls(r *usage.Record, l *line) {
	var counts parser.ToolCounts
	for i := range l.ToolCalls {
		counts.Add(l.ToolCalls[i].Name)
	}
	r.ToolCalls += counts.Total()
	r.Edits += counts.Writes
	r.ToolReads += counts.Reads
	r.ToolSearches += counts.Searches
	r.ToolCommands += counts.Commands
	r.ToolWrites += counts.Writes
	r.ToolOther += counts.Other
}
