package calibration

import (
	"fmt"

	"github.com/assaio/assaio/internal/usage"
)

// Violation is one invariant a record set broke, named so a failure says which rule and
// which record rather than printing two totals that differ.
type Violation struct {
	Rule   string
	Detail string
}

func (v Violation) String() string { return v.Rule + ": " + v.Detail }

// Invariants are the relations that must hold whatever the numbers are, because the domain
// says so rather than because a parser produced them. They are the half of calibration that
// needs no expected value at all -- which is what lets them run over a whole real corpus,
// where no total is known in advance and 5,586 transcripts cannot be hand-counted.
func Invariants(recs []usage.Record, skipped int) []Violation {
	var out []Violation
	for _, check := range []func([]usage.Record, int) []Violation{
		oneResponseBilledOnce,
		subsetsStayWithinTheirWhole,
		reworkNeverExceedsWhatItUndoes,
		purposeSplitSumsToToolCalls,
		everyRecordIsAddressable,
	} {
		out = append(out, check(recs, skipped)...)
	}
	return out
}

// oneResponseBilledOnce: a dedupe key identifies the logical unit that was billed, so two
// records sharing one is the same request counted twice. This is v0.12's defect stated as a
// rule -- it billed one response once per content block -- and it holds for every source,
// because every source has some notion of the thing it charges for.
func oneResponseBilledOnce(recs []usage.Record, _ int) []Violation {
	seen := make(map[string]int, len(recs))
	var out []Violation
	for i := range recs {
		key := recs[i].Tool + "\x00" + recs[i].DedupeKey
		if first, dup := seen[key]; dup {
			out = append(out, Violation{
				Rule:   "one logical response is billed once",
				Detail: fmt.Sprintf("records %d and %d share dedupe key %q", first, i, recs[i].DedupeKey),
			})
			continue
		}
		seen[key] = i
	}
	return out
}

// subsetsStayWithinTheirWhole: the cache classes and the reasoning count are parts of
// figures reported beside them, and a part above its whole is not a rounding nicety -- it
// prices a negative remainder at the other tier, or renders a reasoning share above 100%.
func subsetsStayWithinTheirWhole(recs []usage.Record, _ int) []Violation {
	var out []Violation
	for i := range recs {
		r := &recs[i]
		if r.CacheWrite1hTokens > r.CacheWriteTokens {
			out = append(out, Violation{
				Rule:   "the cache classes sum to the write they are part of",
				Detail: fmt.Sprintf("record %d: cache_write_1h %d exceeds cache_write %d", i, r.CacheWrite1hTokens, r.CacheWriteTokens),
			})
		}
		if r.ReasoningTokens > r.OutputTokens {
			out = append(out, Violation{
				Rule:   "reasoning is part of output, never added to it",
				Detail: fmt.Sprintf("record %d: reasoning %d exceeds output %d", i, r.ReasoningTokens, r.OutputTokens),
			})
		}
	}
	return out
}

// reworkNeverExceedsWhatItUndoes: rework is the share of removals that undo lines the same
// session added, so it is bounded by both -- above the removals it is counting lines nobody
// removed, above the additions it is undoing lines nobody wrote. B132 was the first half;
// the created-file defect was the second, since a file born with zero counted additions gave
// its later removals nothing to be measured against.
func reworkNeverExceedsWhatItUndoes(recs []usage.Record, _ int) []Violation {
	type sums struct{ added, removed, rework int64 }
	bySession := make(map[string]*sums)
	order := make([]string, 0, len(recs))
	for i := range recs {
		r := &recs[i]
		s, ok := bySession[r.SessionID]
		if !ok {
			s = &sums{}
			bySession[r.SessionID] = s
			order = append(order, r.SessionID)
		}
		s.added += r.LinesAdded
		s.removed += r.LinesRemoved
		s.rework += r.ReworkLines
	}
	var out []Violation
	for _, id := range order {
		s := bySession[id]
		if s.rework > s.added {
			out = append(out, Violation{
				Rule:   "rework never exceeds the additions it is undoing",
				Detail: fmt.Sprintf("session %q: rework %d exceeds lines added %d", id, s.rework, s.added),
			})
		}
		if s.rework > s.removed {
			out = append(out, Violation{
				Rule:   "rework never exceeds the removals it is drawn from",
				Detail: fmt.Sprintf("session %q: rework %d exceeds lines removed %d", id, s.rework, s.removed),
			})
		}
	}
	return out
}

// purposeSplitSumsToToolCalls: the split is a partition of the calls, not a second count of
// them, so a reader rendering "reads / searches / commands" beside a total must get the same
// number either way. A source that records no split reports zero on both sides and passes.
func purposeSplitSumsToToolCalls(recs []usage.Record, _ int) []Violation {
	var out []Violation
	for i := range recs {
		r := &recs[i]
		split := r.ToolReads + r.ToolSearches + r.ToolCommands + r.ToolWrites + r.ToolOther
		if split != 0 && split != r.ToolCalls {
			out = append(out, Violation{
				Rule:   "the purpose split partitions the tool calls",
				Detail: fmt.Sprintf("record %d: split sums to %d against %d tool calls", i, split, r.ToolCalls),
			})
		}
	}
	return out
}

// everyRecordIsAddressable: a record with no dedupe key cannot be restated or deleted, and
// one with no timestamp counts toward the store's totals while sitting inside no window --
// present in one place and absent in every other. Evidence that cannot be read is skipped
// and counted instead, which is the number the drift canaries watch.
func everyRecordIsAddressable(recs []usage.Record, _ int) []Violation {
	var out []Violation
	for i := range recs {
		r := &recs[i]
		if r.DedupeKey == "" {
			out = append(out, Violation{
				Rule:   "every record can be addressed again",
				Detail: fmt.Sprintf("record %d has an empty dedupe key", i),
			})
		}
		if r.Timestamp.IsZero() {
			out = append(out, Violation{
				Rule:   "every stored record falls inside some window",
				Detail: fmt.Sprintf("record %d (%s) carries no timestamp", i, r.DedupeKey),
			})
		}
	}
	return out
}
