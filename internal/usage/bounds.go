package usage

import (
	"errors"
	"fmt"
	"time"
)

// The bounds every record crossing into assaio from outside this process must satisfy, in
// one place because there are two such boundaries -- an exec plugin's stdout and a team
// member's HTTP push -- and they were drifting: the server bounded a timestamp and the
// plugin did not, so the same year-9999 record was rejected over the wire and accepted from
// a subprocess, where it then sat in every --since window forever.

// TimestampFloor is the earliest plausible record timestamp; anything before it (including
// the zero value, year 1) is garbage. No AI-coding tool predates it.
var TimestampFloor = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

// FutureSkew is how far past "now" a record may be timestamped before it is rejected --
// generous headroom for clock skew across a team, while still blocking a far-future
// timestamp that would sit permanently in every recent window and never age out.
const FutureSkew = 48 * time.Hour

// MaxCount bounds any single numeric field. Real usage, even session-aggregate granularity
// over a long session, tops out in the low millions; this cap is generous headroom while
// still rejecting overflow-magnitude garbage that would distort a SUM() aggregate.
const MaxCount = 1_000_000_000

// CheckTimestamp rejects a timestamp outside the plausible range.
func CheckTimestamp(ts, now time.Time) error {
	if ts.Before(TimestampFloor) || ts.After(now.Add(FutureSkew)) {
		return fmt.Errorf("timestamp %s out of range", ts.UTC().Format(time.RFC3339))
	}
	return nil
}

// CheckCounts rejects a record whose numeric fields are negative, of overflow magnitude, or
// contradict a documented subset relationship. A subset larger than its whole is not a
// bounds nicety: cache_write_1h is billed at its own rate against the remainder, and
// reasoning_tokens is rendered as a share of output, so either one over its whole publishes
// a negative price or a percentage above 100.
func CheckCounts(r *Record) error {
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
		if v > MaxCount {
			return fmt.Errorf("count field exceeds %d", MaxCount)
		}
	}
	if r.CacheWrite1hTokens > r.CacheWriteTokens {
		return fmt.Errorf("cache_write_1h %d exceeds cache_write %d", r.CacheWrite1hTokens, r.CacheWriteTokens)
	}
	if r.ReasoningTokens > r.OutputTokens {
		return fmt.Errorf("reasoning_tokens %d exceeds output_tokens %d", r.ReasoningTokens, r.OutputTokens)
	}
	return nil
}
