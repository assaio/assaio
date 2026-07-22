package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// maxWireStringLen bounds any single string field a plugin emits: these are identities
// and labels, not free text. With the stdout line cap it stops a plugin from smuggling a
// multi-megabyte field into the store (the metric-result boundary caps strings the same
// way, see metric_result.go).
const maxWireStringLen = 512

// wireRecord is the JSONL record shape a plugin emits, snake_case per the protocol spec
// in docs/extending.md.
type wireRecord struct {
	SessionID        string `json:"session_id"`
	Timestamp        string `json:"timestamp"`
	Model            string `json:"model"`
	InputTokens      int64  `json:"input_tokens"`
	OutputTokens     int64  `json:"output_tokens"`
	CacheReadTokens  int64  `json:"cache_read_tokens"`
	CacheWriteTokens int64  `json:"cache_write_tokens"`
	ReasoningTokens  int64  `json:"reasoning_tokens"`
	DedupeKey        string `json:"dedupe_key"`
	Project          string `json:"project"`
	GitBranch        string `json:"git_branch"`
	Entrypoint       string `json:"entrypoint"`
	Granularity      string `json:"granularity"`
}

// toRecord validates the boundary invariants (honesty rules enforced at ingest) and
// converts a wire record into a namespaced usage.Record. pluginName is the plugin's
// config name; the stored Tool is always "plugin:<name>" so a plugin can never
// impersonate a built-in source.
func (w *wireRecord) toRecord(pluginName string) (usage.Record, error) {
	if w.SessionID == "" {
		return usage.Record{}, errors.New("empty session_id")
	}
	if w.DedupeKey == "" {
		return usage.Record{}, errors.New("empty dedupe_key")
	}
	for _, s := range []string{w.SessionID, w.Model, w.DedupeKey, w.Project, w.GitBranch, w.Entrypoint} {
		if len(s) > maxWireStringLen {
			return usage.Record{}, fmt.Errorf("string field exceeds %d bytes", maxWireStringLen)
		}
	}
	if w.Granularity != "turn" && w.Granularity != "session" {
		return usage.Record{}, fmt.Errorf("invalid granularity %q (want turn|session)", w.Granularity)
	}
	if w.InputTokens < 0 || w.OutputTokens < 0 || w.CacheReadTokens < 0 ||
		w.CacheWriteTokens < 0 || w.ReasoningTokens < 0 {
		return usage.Record{}, errors.New("negative token field")
	}
	ts, err := time.Parse(time.RFC3339, w.Timestamp)
	if err != nil {
		return usage.Record{}, fmt.Errorf("bad timestamp %q: %w", w.Timestamp, err)
	}
	return usage.Record{
		Tool:             "plugin:" + pluginName,
		SessionID:        w.SessionID,
		Timestamp:        ts,
		Model:            w.Model,
		InputTokens:      w.InputTokens,
		OutputTokens:     w.OutputTokens,
		CacheReadTokens:  w.CacheReadTokens,
		CacheWriteTokens: w.CacheWriteTokens,
		ReasoningTokens:  w.ReasoningTokens,
		DedupeKey:        w.DedupeKey,
		Project:          w.Project,
		GitBranch:        w.GitBranch,
		Entrypoint:       w.Entrypoint,
		Granularity:      w.Granularity,
	}, nil
}

// parseRecordLine unmarshals one JSONL line and validates it against the protocol's
// boundary invariants. The returned error, when non-nil, is the skip reason.
func parseRecordLine(line []byte, pluginName string) (usage.Record, error) {
	var w wireRecord
	if err := json.Unmarshal(line, &w); err != nil {
		return usage.Record{}, fmt.Errorf("invalid JSON: %w", err)
	}
	return w.toRecord(pluginName)
}
