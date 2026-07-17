// Package gemini parses Gemini CLI chat logs into normalized usage records.
package gemini

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

const tool = "gemini-cli"

type message struct {
	SessionID string    `json:"sessionId"`
	Timestamp time.Time `json:"timestamp"`
	Model     string    `json:"model"`
	Tokens    struct {
		Input    int64 `json:"input"`
		Output   int64 `json:"output"`
		Cached   int64 `json:"cached"`
		Thoughts int64 `json:"thoughts"`
		Tool     int64 `json:"tool"`
		Total    int64 `json:"total"`
	} `json:"tokens"`
}

// Parse reads a Gemini CLI chat log (JSONL). Model is read per message since Gemini can
// switch models mid-session (e.g. flash fallback). tokens.input includes cached tokens,
// so non-cached input and cache-read tokens are stored separately. tokens.tool and
// tokens.thoughts are both folded into OutputTokens: Gemini bills tool-use and thinking
// tokens as generated output, at the output rate, not a separate class (LiteLLM's
// gemini-2.5 entries price reasoning output at the same per-token rate as output, where
// they price it at all). ReasoningTokens still keeps the raw thoughts count, as an
// informational subset never priced on its own -- so it is never double-charged. Records
// without any token usage are skipped. skipped counts lines that failed to unmarshal as
// JSON; a scanner-level error still aborts the parse. DedupeKey is prefixed with a
// fingerprint of the file's first line, so a session id reused across two files (e.g. a
// resumed session) never collides (see parser.FileFingerprint).
func Parse(r io.Reader) ([]usage.Record, int, error) {
	sc := parser.NewScanner(r)
	var out []usage.Record
	var skipped int
	index := 0
	var fileFP string
	for sc.Scan() {
		raw := sc.Bytes()
		if len(raw) == 0 {
			continue
		}
		if fileFP == "" {
			fileFP = parser.FileFingerprint(raw)
		}
		var m message
		if err := json.Unmarshal(raw, &m); err != nil {
			skipped++
			continue
		}
		if m.Tokens.Total == 0 {
			continue
		}
		out = append(out, usage.Record{
			Tool:            tool,
			SessionID:       m.SessionID,
			Timestamp:       m.Timestamp,
			Model:           m.Model,
			InputTokens:     parser.NonNeg(m.Tokens.Input - m.Tokens.Cached),
			CacheReadTokens: parser.NonNeg(m.Tokens.Cached),
			OutputTokens:    parser.NonNeg(m.Tokens.Output + m.Tokens.Tool + m.Tokens.Thoughts),
			ReasoningTokens: parser.NonNeg(m.Tokens.Thoughts),
			DedupeKey:       fmt.Sprintf("%s:%s:%d", fileFP, m.SessionID, index),
			Granularity:     "turn",
		})
		index++
	}
	if err := sc.Err(); err != nil {
		return out, skipped, fmt.Errorf("scan gemini session: %w", err)
	}
	return out, skipped, nil
}
