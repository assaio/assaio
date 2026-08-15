package analyze

import (
	"github.com/assaio/assaio/internal/humanize"
	"github.com/assaio/assaio/internal/layer"
	"github.com/assaio/assaio/internal/parser"
	"github.com/assaio/assaio/internal/report"
)

const (
	cacheName      = "cache-hygiene"
	cacheTitle     = "Cache Hygiene"
	cacheDescribe  = "Prompt-cache reuse: how much billed input was served from cache vs re-sent, which lifetime the writes bought, and why a miss happened."
	cacheHowToRead = "High cache reuse means repeated context is served cheaply from cache instead of re-billed as fresh input. It is a cost signal, not a quality one -- a big one-shot task legitimately shows low reuse. Where a source states it, the lifetime a write bought and the vendor's own reason for a miss say which of those two a low share is."
	// cacheGoodReuse is the cache-read share above which reuse reads as healthy.
	cacheGoodReuse = 0.5
)

func init() { Register(cacheValidator{}) }

// cacheValidator reads prompt-cache efficiency: the share of input tokens served from
// cache (cheaper) versus re-sent as fresh input, and whether cache writes are paying off.
type cacheValidator struct{}

func (cacheValidator) Name() string       { return cacheName }
func (cacheValidator) Title() string      { return cacheTitle }
func (cacheValidator) Describe() string   { return cacheDescribe }
func (cacheValidator) Layer() layer.Layer { return layer.Activity } // how much billed input was served from cache

//nolint:gocritic // Input is a small value bundle required by the Validator interface; analyzed once per CLI run, not a hot path.
func (cacheValidator) Analyze(in Input) Result {
	r := Result{Name: cacheName, Title: cacheTitle, Describe: cacheDescribe, HowToRead: cacheHowToRead}
	t := in.Totals
	if t.Input == 0 && t.CacheRead == 0 && t.CacheWrite == 0 {
		r.noData("active days", "No usage in this window.")
		return r
	}

	r.restsOn(activeDays(&in), "active days")
	reuse := t.CacheEfficiency // CacheRead / (CacheRead + Input)
	healthy := reuse >= cacheGoodReuse

	r.Read = readFor(healthy, "Efficient")
	r.Purity = clamp01(reuse)
	r.Figures = []Figure{
		{Label: "cache-read share", Value: humanize.Percent(reuse), Note: "of billed input"},
		{Label: "cache reads", Value: humanize.Count(t.CacheRead)},
		{Label: "cache writes", Value: humanize.Count(t.CacheWrite), Note: cacheWriteNote(t.CacheRead, t.CacheWrite)},
		longTierFigure(&in),
		missCauseFigure(&in),
	}
	r.Takeaway = cacheTakeaway(healthy, t.CacheRead, t.CacheWrite)
	r.Caveats = []string{
		"High reuse is cheaper, not better work -- a large one-shot task legitimately shows low reuse.",
		"Reuse is measured at day grain, so a write and a read on the same day count together whatever the gap between them was.",
	}
	if !answersCacheDetail(&in) {
		r.Caveats = append(r.Caveats,
			"No source in this window states the lifetime a write bought or why a miss happened, so both figures read as absent, not zero.")
	} else {
		r.Caveats = append(r.Caveats,
			"The tier is gated per source, not per row, so a row parsed before assaio read the tier -- a team member still on an older build -- contributes its write with no tier and lowers the 1-hour share.")
	}
	return r
}

// longTierFigure reports how much of the window's cache-write bought the 1-hour lifetime,
// which bills at a higher rate than the 5-minute default. Sources that do not state the tier
// are excluded from the denominator: their silence is not "every write was the cheap one".
func longTierFigure(in *Input) Figure {
	capable := report.UsageAnswering(in.Usage, parser.SignalTokensCacheWrite1h)
	var write, long int64
	for i := range capable {
		write += capable[i].CacheWrite
		long += capable[i].CacheWrite1h
	}
	return Figure{
		Label: "1-hour cache writes",
		Value: humanize.PercentOrDash(long, write, 1),
		Note:  "of cache writes that state a tier; billed above the 5-minute rate",
	}
}

// missCauseFigure names the most common reason the vendor could not serve a prompt from
// cache. It is the tool's own label, so a low read share becomes something to act on rather
// than a number to stare at. Counts are folded by reason across sources rather than read off
// one source's row: two tools reporting the same cause are one cause, and taking the top row
// as-is would rank a single tool's count against every tool's total.
func missCauseFigure(in *Input) Figure {
	byReason := make(map[string]int64, len(in.CacheMisses))
	var total int64
	for i := range in.CacheMisses {
		m := &in.CacheMisses[i]
		byReason[m.Reason] += m.Turns
		total += m.Turns
	}
	if total == 0 {
		return Figure{Label: "top miss cause", Value: "—", Note: "no turn in this window stated one"}
	}
	var top string
	var turns int64
	for reason, n := range byReason {
		// Ties break on the name so the figure does not change between identical runs.
		if n > turns || (n == turns && reason < top) {
			top, turns = reason, n
		}
	}
	return Figure{
		Label: "top miss cause",
		Value: top,
		Note:  humanize.PercentOrDash(turns, total, 0) + " of turns that stated a reason",
	}
}

// answersCacheDetail reports whether any source in the window publishes either the cache tier
// or the miss reason, so the caveat names an absent capability rather than an absent number.
// Either is enough: the caveat denies both figures, and a source publishing one alone would
// otherwise have its real percentage printed directly under a sentence calling it absent.
// The figure that is genuinely unanswered prints an em dash of its own.
func answersCacheDetail(in *Input) bool {
	return len(report.UsageAnswering(in.Usage, parser.SignalTokensCacheWrite1h)) > 0 ||
		len(report.UsageAnswering(in.Usage, parser.SignalCacheMissReason)) > 0
}

func cacheTakeaway(healthy bool, read, write int64) string {
	switch {
	case healthy:
		return "Most repeated context is served from cache, keeping input cost down."
	case write > 0 && read < write:
		return "Cache is written more than it is reused -- short or churning sessions may be paying to cache context that is never read back."
	default:
		return "Little input is served from cache this window -- expected for one-shot or exploratory work."
	}
}

// cacheWriteNote flags cache writes that outweigh reads: paying to cache context that is
// not (yet) being read back.
func cacheWriteNote(read, write int64) string {
	if write > 0 && read < write {
		return "written more than reused"
	}
	return ""
}
