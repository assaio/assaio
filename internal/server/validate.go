package server

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/assaio/assaio/internal/usage"
)

// maxFieldValue bounds any single numeric field on a pushed record. Real usage, even
// session-aggregate granularity over a long session, tops out in the low millions; this
// cap is generous headroom while still rejecting overflow-magnitude garbage that would
// distort the shared dashboard's SUM() aggregates.
const maxFieldValue = 1_000_000_000

// tsFloor is the earliest plausible record timestamp; anything before it (including the
// zero value, year 1) is garbage. No AI-coding tool predates it.
var tsFloor = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

// tsFutureSkew is how far past "now" a record may be timestamped before it is rejected --
// generous headroom for clock skew across a team, while still blocking a far-future
// timestamp that would sit permanently in every recent window and never age out of the
// shared dashboard.
const tsFutureSkew = 48 * time.Hour

// knownTools is the exact set of Tool values assaio's built-in parsers emit (see
// internal/ingest.discoverSources and internal/parser/*).
var knownTools = map[string]bool{
	"claude-code": true,
	"codex":       true,
	"cline":       true,
	"gemini-cli":  true,
}

// pluginToolPattern matches an out-of-tree plugin's Tool namespace, "plugin:<name>",
// with the same charset a plugin's configured name must match (internal/config's
// pluginNamePattern) -- so a pushed record can claim to be plugin-sourced only with a
// well-formed name, never an arbitrary string dressed up to look like one.
var pluginToolPattern = regexp.MustCompile(`^plugin:[a-z0-9-]+$`)

// knownGranularities is the exact set of Granularity values usage.Record documents.
var knownGranularities = map[string]bool{"turn": true, "session": true}

// validateRecord rejects a pushed record that could poison the shared dashboard: an
// unrecognized Tool or Granularity (including a forged "plugin:x" namespace or a
// mislabeled granularity), a timestamp that is zero or out of plausible range, or any
// numeric field that is negative or an overflow-magnitude outlier.
func validateRecord(r *usage.Record) error {
	return validateRecordAt(r, time.Now())
}

func validateRecordAt(r *usage.Record, now time.Time) error {
	if !knownTools[r.Tool] && !pluginToolPattern.MatchString(r.Tool) {
		return fmt.Errorf("unknown tool %q", r.Tool)
	}
	if !knownGranularities[r.Granularity] {
		return fmt.Errorf("unknown granularity %q", r.Granularity)
	}
	if r.Timestamp.Before(tsFloor) || r.Timestamp.After(now.Add(tsFutureSkew)) {
		return fmt.Errorf("timestamp %s out of range", r.Timestamp.UTC().Format(time.RFC3339))
	}
	fields := [...]int64{
		r.InputTokens, r.OutputTokens, r.CacheReadTokens, r.CacheWriteTokens, r.ReasoningTokens,
		r.LinesAdded, r.LinesRemoved, r.Edits, r.ToolCalls, r.Rejected, r.Compactions, r.ReworkLines,
	}
	for _, v := range fields {
		if v < 0 {
			return errors.New("negative count field")
		}
		if v > maxFieldValue {
			return fmt.Errorf("count field exceeds %d", maxFieldValue)
		}
	}
	return nil
}
