package analyze

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

// stampHorizon appends the history line to a trending Result: how far back the store itself goes,
// and whether that reaches past the earlier span the trend leans on.
//
// The boundary comes from recentCutoff -- the same function weekOverWeek buckets with -- rather
// than from arithmetic on Now, which was a day earlier and told a covered trend it was not.
//
// The comparison runs against the store's own earliest record, deliberately not against the
// window's rows: those are queried WHERE ts >= start, so their earliest day can never precede the
// window and every narrow --since would accuse the source of deleting history the store holds.
//
// What a MIN(ts) supports is one-directional, and the wording says only that much. Later than the
// boundary proves the span is short, because nothing exists before it. Earlier proves only that the
// store reaches back that far -- not that the span holds data, which a source added later or a
// quiet fortnight can still leave empty.
func stampHorizon(r *Result, in *Input) {
	priorFrom := recentCutoff(in.Now, 2*in.Recent)
	if in.HistoryStart.IsZero() {
		r.Caveats = append(r.Caveats, "Prov.: how far back this store's own history goes could not be read, so the span this trend compares against is unverified.")
		return
	}
	begins := in.HistoryStart.UTC().Format("2006-01-02")
	if begins > priorFrom {
		r.Caveats = append(r.Caveats, "Prov.: this store's history begins "+begins+
			", inside the earlier span this trend compares against, which starts "+priorFrom+
			". That span is therefore only part of a span, and the change above reads as growth from a base the store never held. "+
			"History a source has already deleted cannot be re-read: `doctor` reports the retention behind it.")
		return
	}
	r.Caveats = append(r.Caveats, "Prov.: this store's history begins "+begins+
		", which reaches past the earlier span this trend compares against ("+priorFrom+
		"). That the store reaches back is not that the span holds data: a source added later starts where it was first read, and a quiet fortnight leaves a real gap.")
}
