package server

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/usage"
)

// maxFieldValue bounds any single numeric field on a pushed record. Real usage, even
// session-aggregate granularity over a long session, tops out in the low millions; this
// cap is generous headroom while still rejecting overflow-magnitude garbage that would
// distort the shared dashboard's SUM() aggregates.
const maxFieldValue = 1_000_000_000

// maxStringField bounds any single string field on a pushed record: these are identities
// and labels, not free text. It blocks a client from smuggling a multi-megabyte model or
// dedupe_key into the shared store and the unauthenticated dashboard it feeds.
const maxStringField = 512

// tsFloor is the earliest plausible record timestamp; anything before it (including the
// zero value, year 1) is garbage. No AI-coding tool predates it.
var tsFloor = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

// tsFutureSkew is how far past "now" a record may be timestamped before it is rejected --
// generous headroom for clock skew across a team, while still blocking a far-future
// timestamp that would sit permanently in every recent window and never age out of the
// shared dashboard.
const tsFutureSkew = 48 * time.Hour

// knownTools is the exact set of Tool values assaio's built-in parsers emit, read from the
// depth matrix so a source becomes syncable the moment it declares its depth -- a list kept
// here instead is one a new parser silently fails to join.
var knownTools = toolSet(parser.Tools())

func toolSet(tools []string) map[string]bool {
	set := make(map[string]bool, len(tools))
	for _, t := range tools {
		set[t] = true
	}
	return set
}

// pluginToolPattern matches an out-of-tree plugin's Tool namespace, "plugin:<name>",
// with the same charset a plugin's configured name must match (internal/config's
// pluginNamePattern) -- so a pushed record can claim to be plugin-sourced only with a
// well-formed name, never an arbitrary string dressed up to look like one.
var pluginToolPattern = regexp.MustCompile(`^plugin:[a-z0-9-]+$`)

// cacheMissReasonPattern is the shape of a vendor's cache-miss vocabulary token, matched
// rather than enumerated: the vocabulary is the vendor's to grow, so pinning today's six
// values here would reject a record a future parser reads correctly. What it does block is
// the reason being free text -- the value is rendered as the "top miss cause" on the shared
// unauthenticated dashboard, and it keys a GROUP BY. Empty is allowed: most turns state none.
var cacheMissReasonPattern = regexp.MustCompile(`^[a-z0-9_]{0,64}$`)

// knownGranularities is the exact set of Granularity values usage.Record documents.
var knownGranularities = map[string]bool{"turn": true, "session": true}

// validateRecord rejects a pushed record that could poison the shared dashboard: an
// unrecognized Tool or Granularity (including a forged "plugin:x" namespace or a
// mislabeled granularity), a timestamp that is zero or out of plausible range, any numeric
// field that is negative or an overflow-magnitude outlier, a Sidechain flag that is neither
// 0 nor 1, or a tool-purpose split that contradicts the tool-call count it splits.
func validateRecord(r *usage.Record) error {
	return validateRecordAt(r, time.Now())
}

func validateRecordAt(r *usage.Record, now time.Time) error {
	// Only an empty dedupe_key is fatal: it is the ON CONFLICT(tool, dedupe_key) key, so a
	// blank one collapses rows. session_id can legitimately be empty (the Claude parser does
	// not guarantee it), and rejecting the whole push over it would break a client's entire
	// sync on one stale row. Member is not checked here -- it is overwritten server-side and
	// bounded by ValidateMember.
	if r.DedupeKey == "" {
		return errors.New("empty dedupe_key")
	}
	for _, s := range []string{r.Tool, r.SessionID, r.Model, r.DedupeKey, r.Project, r.Subpath, r.GitBranch, r.Entrypoint, r.Skill, r.Agent} {
		if len(s) > maxStringField {
			return fmt.Errorf("string field exceeds %d bytes", maxStringField)
		}
	}
	if !cacheMissReasonPattern.MatchString(r.CacheMissReason) {
		return fmt.Errorf("cache_miss_reason %q is not a vocabulary token", r.CacheMissReason)
	}
	if !knownTools[r.Tool] && !pluginToolPattern.MatchString(r.Tool) {
		return fmt.Errorf("unknown tool %q", r.Tool)
	}
	if !knownGranularities[r.Granularity] {
		return fmt.Errorf("unknown granularity %q", r.Granularity)
	}
	if r.Timestamp.Before(tsFloor) || r.Timestamp.After(now.Add(tsFutureSkew)) {
		return fmt.Errorf("timestamp %s out of range", r.Timestamp.UTC().Format(time.RFC3339))
	}
	// Every numeric field a record carries must appear here: an unbounded one is summed
	// into the shared aggregates, where a negative value renders impossible percentages and
	// an overflow-magnitude one makes SUM() fail for the whole team.
	fields := [...]int64{
		r.InputTokens, r.OutputTokens, r.CacheReadTokens, r.CacheWriteTokens, r.ReasoningTokens,
		r.CacheWrite1hTokens,
		r.LinesAdded, r.LinesRemoved, r.Edits, r.ToolCalls, r.Rejected, r.Compactions, r.ReworkLines,
		r.ToolReads, r.ToolSearches, r.ToolCommands, r.ToolWrites, r.ToolOther, r.ToolErrors,
		r.Sidechain,
	}
	for _, v := range fields {
		if v < 0 {
			return errors.New("negative count field")
		}
		if v > maxFieldValue {
			return fmt.Errorf("count field exceeds %d", maxFieldValue)
		}
	}
	if r.Sidechain != 0 && r.Sidechain != 1 {
		return fmt.Errorf("sidechain %d is not 0 or 1", r.Sidechain)
	}
	// The 1-hour portion is part of the cache write, and the two tiers are billed at
	// different rates: a portion larger than its whole prices a negative 5-minute remainder
	// into a total the whole team reads.
	if r.CacheWrite1hTokens > r.CacheWriteTokens {
		return fmt.Errorf("cache_write_1h %d exceeds cache_write %d", r.CacheWrite1hTokens, r.CacheWriteTokens)
	}
	return validateToolPurposeSplit(r)
}

// validateToolPurposeSplit holds the contract both parsers' fuzz tests assert: the purpose
// buckets split ToolCalls, so they sum to it. An all-zero split is the documented "not
// captured" state and stays accepted; a non-zero one that disagrees is rejected, because
// the team dashboard reads their ratio as coverage and would present a partial split as a
// complete one.
func validateToolPurposeSplit(r *usage.Record) error {
	split := r.ToolReads + r.ToolSearches + r.ToolCommands + r.ToolWrites + r.ToolOther
	if split == 0 || split == r.ToolCalls {
		return nil
	}
	return fmt.Errorf("tool purpose split sums to %d, not tool_calls %d", split, r.ToolCalls)
}
