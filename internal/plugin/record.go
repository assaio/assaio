package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

// maxWireStringLen bounds any single string field a plugin emits: these are identities
// and labels, not free text. With the stdout line cap it stops a plugin from smuggling a
// multi-megabyte field into the store (the metric-result boundary caps strings the same
// way, see metric_result.go).
const maxWireStringLen = 512

// wireRecord is the JSONL record shape a plugin emits, snake_case per the protocol spec
// in docs/extending.md. The token counters are pointers so an omitted counter stays
// distinguishable from a stated zero (see countsTokens).
type wireRecord struct {
	SessionID        string `json:"session_id"`
	Timestamp        string `json:"timestamp"`
	Model            string `json:"model"`
	InputTokens      *int64 `json:"input_tokens"`
	OutputTokens     *int64 `json:"output_tokens"`
	CacheReadTokens  *int64 `json:"cache_read_tokens"`
	CacheWriteTokens *int64 `json:"cache_write_tokens"`
	ReasoningTokens  *int64 `json:"reasoning_tokens"`
	DedupeKey        string `json:"dedupe_key"`
	Project          string `json:"project"`
	GitBranch        string `json:"git_branch"`
	Entrypoint       string `json:"entrypoint"`
	Granularity      string `json:"granularity"`
}

// errNoTokenCounter refuses a record that states no token count at all. The protocol is
// token-denominated and every plugin source is reported as counting tokens, so storing such a
// record would render a source that measured nothing as "0 tokens, $0.00" -- absence as zero.
var errNoTokenCounter = errors.New(
	"no token counter: a record carries at least one of input_tokens, output_tokens, cache_read_tokens, cache_write_tokens")

// countsTokens reports whether the record states at least one token counter. The protocol has
// no way to say a source does not keep a counter, so one omitted beside a stated one defaults to
// 0; reasoning_tokens alone does not count, because it is a subset of output_tokens.
func (w *wireRecord) countsTokens() bool {
	return w.InputTokens != nil || w.OutputTokens != nil || w.CacheReadTokens != nil || w.CacheWriteTokens != nil
}

func orZero(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// toRecordAt validates the boundary invariants (honesty rules enforced at ingest) and converts
// a wire record into a namespaced usage.Record. pluginName is the plugin's config name; the
// stored Tool is always "plugin:<name>" so a plugin can never impersonate a built-in source.
// The clock is injected so the range a timestamp is judged against is the caller's to choose.
func (w *wireRecord) toRecordAt(pluginName string, now time.Time) (usage.Record, error) {
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
	ts, err := time.Parse(time.RFC3339, w.Timestamp)
	if err != nil {
		return usage.Record{}, fmt.Errorf("bad timestamp %q: %w", w.Timestamp, err)
	}
	// The same range and magnitude rules the sync endpoint applies to the same shape. A
	// plugin is opt-in config, not a stranger -- but an unbounded timestamp from one sits in
	// every --since window forever, which is the store's problem rather than the plugin's.
	if err := usage.CheckTimestamp(ts, now); err != nil {
		return usage.Record{}, err
	}
	if !w.countsTokens() {
		return usage.Record{}, errNoTokenCounter
	}
	rec := usage.Record{
		Tool:             parser.PluginPrefix + pluginName,
		SessionID:        w.SessionID,
		Timestamp:        ts,
		Model:            w.Model,
		InputTokens:      orZero(w.InputTokens),
		OutputTokens:     orZero(w.OutputTokens),
		CacheReadTokens:  orZero(w.CacheReadTokens),
		CacheWriteTokens: orZero(w.CacheWriteTokens),
		ReasoningTokens:  orZero(w.ReasoningTokens),
		DedupeKey:        w.DedupeKey,
		Project:          w.Project,
		GitBranch:        w.GitBranch,
		Entrypoint:       w.Entrypoint,
		Granularity:      w.Granularity,
	}
	if err := usage.CheckCounts(&rec); err != nil {
		return usage.Record{}, err
	}
	return rec, nil
}

// parseRecordLine unmarshals one JSONL line and validates it against the protocol's
// boundary invariants. The returned error, when non-nil, is the skip reason.
//
// Decoding is strict, as the metric and rule protocols are: an unknown field is a named
// violation rather than a stored zero (ADR 0003), and a line carries exactly one JSON value --
// a second object after the first would otherwise be dropped while the line counted as valid.
func parseRecordLine(line []byte, pluginName string) (usage.Record, error) {
	return parseRecordLineAt(line, pluginName, time.Now())
}

// parseRecordLineAt is parseRecordLine with the clock injected, so the published conformance
// vectors can pin a timestamp that would otherwise drift out of range as the wall clock moves.
func parseRecordLineAt(line []byte, pluginName string, now time.Time) (usage.Record, error) {
	var w wireRecord
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&w); err != nil {
		return usage.Record{}, fmt.Errorf("invalid JSON: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return usage.Record{}, errors.New("trailing data after the record: a line carries exactly one JSON object")
	}
	return w.toRecordAt(pluginName, now)
}
