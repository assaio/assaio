// Package copilot parses GitHub Copilot CLI session logs into normalized usage records.
package copilot

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

const tool = "copilot-cli"

// event is one line of a session's events.jsonl. Only the two events that carry accounting
// are modeled: session.start for identity and working directory, session.shutdown for the
// totals. Everything else -- messages, permissions, tool calls -- is content assaio does
// not read.
type event struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      struct {
		SessionID string `json:"sessionId"`
		Context   struct {
			Cwd string `json:"cwd"`
		} `json:"context"`
		CodeChanges struct {
			LinesAdded   int64 `json:"linesAdded"`
			LinesRemoved int64 `json:"linesRemoved"`
		} `json:"codeChanges"`
		ModelMetrics map[string]modelMetrics `json:"modelMetrics"`
	} `json:"data"`
}

// modelMetrics is one model's share of a session. tokenDetails is the authoritative split:
// usage.inputTokens is the whole prompt *including* tokens written to cache, so reading it
// as the uncached input beside cacheWriteTokens would count those tokens twice and inflate
// cost by the size of the cache write -- which on a real session is most of the prompt.
type modelMetrics struct {
	Requests struct {
		Count int64 `json:"count"`
	} `json:"requests"`
	Usage struct {
		ReasoningTokens int64 `json:"reasoningTokens"`
	} `json:"usage"`
	TokenDetails struct {
		Input      tokenCount `json:"input"`
		CacheRead  tokenCount `json:"cache_read"`
		CacheWrite tokenCount `json:"cache_write"`
		Output     tokenCount `json:"output"`
	} `json:"tokenDetails"`
}

type tokenCount struct {
	TokenCount int64 `json:"tokenCount"`
}

// Parse reads one Copilot CLI session's events.jsonl and returns one record per model the
// session used. Copilot only totals a session's usage when it shuts down, so the records
// are session-granularity by construction: a session still running, or one killed before
// it could write that event, yields nothing rather than a row of zeros.
//
// skipped counts lines that failed to unmarshal as JSON, plus a session that carries no id
// or no timestamp and so cannot be stored under a key or a date; a scanner-level error still
// aborts the parse.
func Parse(r io.Reader) ([]usage.Record, int, error) {
	sc := parser.NewScanner(r)
	var skipped int
	var start, shutdown event
	for sc.Scan() {
		raw := sc.Bytes()
		if len(raw) == 0 {
			continue
		}
		var e event
		if err := json.Unmarshal(raw, &e); err != nil {
			skipped++
			continue
		}
		switch e.Type {
		case "session.start":
			start = e
		case "session.shutdown":
			shutdown = e
		}
	}
	if err := sc.Err(); err != nil {
		return nil, skipped, fmt.Errorf("scan copilot session: %w", err)
	}
	recs, dropped := records(&start, &shutdown)
	return recs, skipped + dropped, nil
}

// records turns one session's start and shutdown events into one record per model, and
// reports the session as skipped when it carries no identity or no clock: the dedupe key is
// "<session>:<model>", so an empty id would collapse every such session into one stored row,
// and a zero timestamp would date real usage to year one.
func records(start, shutdown *event) (recs []usage.Record, skipped int) {
	if len(shutdown.Data.ModelMetrics) == 0 {
		return nil, 0
	}
	sessionID := start.Data.SessionID
	if sessionID == "" {
		sessionID = shutdown.Data.SessionID
	}
	at := start.Timestamp
	if at.IsZero() {
		at = shutdown.Timestamp
	}
	if sessionID == "" || at.IsZero() {
		return nil, 1
	}
	models := sortedModels(shutdown.Data.ModelMetrics)
	credited := dominantModel(shutdown.Data.ModelMetrics)
	out := make([]usage.Record, 0, len(models))
	for _, model := range models {
		m := shutdown.Data.ModelMetrics[model]
		rec := usage.Record{
			Tool:             tool,
			SessionID:        sessionID,
			Timestamp:        at,
			Model:            model,
			InputTokens:      parser.NonNeg(m.TokenDetails.Input.TokenCount),
			OutputTokens:     parser.NonNeg(m.TokenDetails.Output.TokenCount),
			CacheReadTokens:  parser.NonNeg(m.TokenDetails.CacheRead.TokenCount),
			CacheWriteTokens: parser.NonNeg(m.TokenDetails.CacheWrite.TokenCount),
			ReasoningTokens:  parser.NonNeg(m.Usage.ReasoningTokens),
			DedupeKey:        sessionID + ":" + model,
			Cwd:              start.Data.Context.Cwd,
			Granularity:      "session",
		}
		// Copilot counts code changes once for the whole session with no per-model
		// breakdown, so they go whole to the model that did the most requests. Splitting
		// them by a token or request ratio would be a number nobody measured.
		if model == credited {
			rec.LinesAdded = parser.NonNeg(shutdown.Data.CodeChanges.LinesAdded)
			rec.LinesRemoved = parser.NonNeg(shutdown.Data.CodeChanges.LinesRemoved)
		}
		out = append(out, rec)
	}
	return out, 0
}

// sortedModels keeps record order deterministic across runs, which golden tests and stable
// dedupe both rely on.
func sortedModels(m map[string]modelMetrics) []string {
	out := make([]string, 0, len(m))
	for name := range m {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// dominantModel is the one that served the most requests, ties broken by name so the
// choice never depends on map iteration order.
func dominantModel(m map[string]modelMetrics) string {
	best, bestCount := "", int64(-1)
	for _, name := range sortedModels(m) {
		if c := m[name].Requests.Count; c > bestCount {
			best, bestCount = name, c
		}
	}
	return best
}
