package analyze

import "time"

// Trending marks a Validator whose figures compare one span of the window against an earlier one.
// Such a figure is worth exactly as much as the history behind it, and that history is not the
// install's: several tools delete their own transcripts -- Claude Code after 30 days by default --
// so "the prior week" can be a week no store ever held, and a trend read off it reports growth
// where the truth is a shorter memory (B156).
//
// A validator declares this; Evaluate stamps the horizon line onto the ones that do and onto no
// others, so a within-window figure never carries a caveat about history it does not use.
type Trending interface {
	Trending()
}

// horizonCovers reports whether the store's own history reaches back to the first day of the
// earlier span a trend compares against; known is false when that history could not be read. It
// is the one rule behind both the horizon line and a trend's refusal to read a direction.
//
// The comparison runs against the store's own earliest record, deliberately not against the
// window's rows: those are queried WHERE ts >= start, so their earliest day can never precede the
// window and every narrow --since would accuse the source of deleting history the store holds.
func horizonCovers(in *Input) (covers, known bool) {
	if in.HistoryStart.IsZero() {
		return false, false
	}
	_, prior := trendSpans(in.Now, in.Recent)
	return in.HistoryStart.UTC().Format(time.DateOnly) <= prior.from, true
}

// stampHorizon appends the history line to a trending Result: how far back the store itself goes,
// and whether that reaches the earlier span the trend leans on.
//
// What a MIN(ts) supports is one-directional, and the wording says only that much. Later than the
// span's first day proves the span is short, because nothing exists before it. Earlier proves only
// that the store reaches back that far -- not that the span holds data, which a source added later
// or a quiet fortnight can still leave empty.
func stampHorizon(r *Result, in *Input) {
	_, prior := trendSpans(in.Now, in.Recent)
	covers, known := horizonCovers(in)
	if !known {
		r.Caveats = append(r.Caveats, "Prov.: how far back this store's own history goes could not be read, so the span this trend compares against is unverified.")
		return
	}
	begins := in.HistoryStart.UTC().Format(time.DateOnly)
	if !covers {
		r.Caveats = append(r.Caveats, "Prov.: this store's history begins "+begins+", inside the earlier span, which starts "+prior.from+
			"; the base was never held, so no direction is stated. Deleted source history cannot be re-read; `doctor` reports the retention behind it.")
		return
	}
	r.Caveats = append(r.Caveats, "Prov.: this store's history begins "+begins+
		", which reaches past the earlier span this trend compares against ("+prior.from+
		"). That the store reaches back is not that the span holds data: a source added later starts where it was first read, and a quiet fortnight leaves a real gap.")
}
